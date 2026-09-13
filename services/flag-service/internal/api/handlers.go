package api

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gin-gonic/gin"

	"flag-service/internal/cache"
	"flag-service/internal/domain"
	"flag-service/internal/evaluator"
)

type FlagHandler struct {
	dynamoDbClient   *dynamodb.Client
	tableName        string
	cache            *cache.RulesetCache
	subscribers      map[chan string]bool
	subscribersMutex sync.RWMutex
	inMemoryFlags    map[string]domain.FeatureFlag
	inMemoryMutex    sync.RWMutex
}

func NewFlagHandler(dynamoDbClient *dynamodb.Client, tableName string, rulesetCache *cache.RulesetCache) *FlagHandler {
	defaultFlag := domain.FeatureFlag{
		Key:           "ds-button-v2",
		Enabled:       true,
		DefaultValue:  "v1",
		Rollout:       nil,
		UserOverrides: map[string]domain.VariationValue{},
		UpdatedAt:     time.Now().Unix(),
	}

	return &FlagHandler{
		dynamoDbClient: dynamoDbClient,
		tableName:      tableName,
		cache:          rulesetCache,
		subscribers:    make(map[chan string]bool),
		inMemoryFlags: map[string]domain.FeatureFlag{
			"ds-button-v2": defaultFlag,
		},
	}
}

func (handler *FlagHandler) broadcastSSE(messagePayload string) {
	handler.subscribersMutex.RLock()
	defer handler.subscribersMutex.RUnlock()
	for clientChannel := range handler.subscribers {
		select {
		case clientChannel <- messagePayload:
		default:
		}
	}
}

type SetOverrideRequest struct {
	Variation any `json:"variation" binding:"required"`
}

// ListFlags retrieves all flags for tenant & env
// GET /api/v1/admin/flags
func (handler *FlagHandler) ListFlags(ginContext *gin.Context) {
	flags, _, err := handler.getOrFetchFlags(ginContext.Request.Context(), "acme", "local")
	if err != nil {
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ginContext.JSON(http.StatusOK, flags)
}

// SetUserOverride updates a flag's user override in DynamoDB and invalidates Redis
// PUT /api/v1/admin/flags/:key/overrides/users/:userId
func (handler *FlagHandler) SetUserOverride(ginContext *gin.Context) {
	flagKey := ginContext.Param("key")
	userID := strings.TrimPrefix(ginContext.Param("userId"), "/")

	var requestBody SetOverrideRequest
	if err := ginContext.ShouldBindJSON(&requestBody); err != nil {
		ginContext.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Always update in-memory cache as fallback
	handler.inMemoryMutex.Lock()
	if flag, exists := handler.inMemoryFlags[flagKey]; exists {
		if flag.UserOverrides == nil {
			flag.UserOverrides = make(map[string]domain.VariationValue)
		}
		flag.UserOverrides[userID] = requestBody.Variation
		flag.UpdatedAt = time.Now().Unix()
		handler.inMemoryFlags[flagKey] = flag
	}
	handler.inMemoryMutex.Unlock()

	partitionKey := "TENANT#acme#ENV#local"
	sortKey := fmt.Sprintf("FLAG#%s", flagKey)

	valueAttributeValue, err := attributevalue.Marshal(requestBody.Variation)
	if err == nil && handler.dynamoDbClient != nil {
		// Attempt UpdateItem in DynamoDB
		_, _ = handler.dynamoDbClient.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
			TableName: aws.String(handler.tableName),
			Key: map[string]types.AttributeValue{
				"PK": &types.AttributeValueMemberS{Value: partitionKey},
				"SK": &types.AttributeValueMemberS{Value: sortKey},
			},
			UpdateExpression: aws.String("SET UserOverrides = if_not_exists(UserOverrides, :empty_map), UserOverrides.#uid = :val, UpdatedAt = :now"),
			ExpressionAttributeNames: map[string]string{
				"#uid": userID,
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":val":       valueAttributeValue,
				":empty_map": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{}},
				":now":       &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", time.Now().Unix())},
			},
		})
	}

	// Invalidate local Redis cache if available
	if handler.cache != nil {
		_ = handler.cache.Invalidate(ginContext.Request.Context(), "acme", "local")
	}

	// Broadcast SSE event
	variationBytes, _ := json.Marshal(requestBody.Variation)
	eventMessage := fmt.Sprintf(`{"type":"FLAG_OVERRIDE_UPDATED","flagKey":"%s","userId":"%s","variation":%s}`, flagKey, userID, string(variationBytes))
	handler.broadcastSSE(eventMessage)

	ginContext.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"flagKey":   flagKey,
		"userId":    userID,
		"variation": requestBody.Variation,
	})
}

// RemoveUserOverride deletes a flag's user override in DynamoDB and invalidates Redis
// DELETE /api/v1/admin/flags/:key/overrides/users/:userId
func (handler *FlagHandler) RemoveUserOverride(ginContext *gin.Context) {
	flagKey := ginContext.Param("key")
	userID := strings.TrimPrefix(ginContext.Param("userId"), "/")

	// Update in-memory cache as fallback
	handler.inMemoryMutex.Lock()
	if flag, exists := handler.inMemoryFlags[flagKey]; exists {
		if flag.UserOverrides != nil {
			delete(flag.UserOverrides, userID)
			flag.UpdatedAt = time.Now().Unix()
			handler.inMemoryFlags[flagKey] = flag
		}
	}
	handler.inMemoryMutex.Unlock()

	partitionKey := "TENANT#acme#ENV#local"
	sortKey := fmt.Sprintf("FLAG#%s", flagKey)

	if handler.dynamoDbClient != nil {
		_, _ = handler.dynamoDbClient.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
			TableName: aws.String(handler.tableName),
			Key: map[string]types.AttributeValue{
				"PK": &types.AttributeValueMemberS{Value: partitionKey},
				"SK": &types.AttributeValueMemberS{Value: sortKey},
			},
			UpdateExpression: aws.String("REMOVE UserOverrides.#uid SET UpdatedAt = :now"),
			ExpressionAttributeNames: map[string]string{
				"#uid": userID,
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":now": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", time.Now().Unix())},
			},
		})
	}

	// Invalidate local Redis cache if available
	if handler.cache != nil {
		_ = handler.cache.Invalidate(ginContext.Request.Context(), "acme", "local")
	}

	// Broadcast SSE event
	eventMessage := fmt.Sprintf(`{"type":"FLAG_OVERRIDE_REMOVED","flagKey":"%s","userId":"%s"}`, flagKey, userID)
	handler.broadcastSSE(eventMessage)

	ginContext.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"flagKey": flagKey,
		"userId":  userID,
	})
}

// EvaluateUser evaluates all flags for a given userId
// GET /api/v1/evaluate?userId=...
func (handler *FlagHandler) EvaluateUser(ginContext *gin.Context) {
	userID := ginContext.Query("userId")
	if userID == "" {
		ginContext.JSON(http.StatusBadRequest, gin.H{"error": "userId query param required"})
		return
	}

	flags, etag, err := handler.getOrFetchFlags(ginContext.Request.Context(), "acme", "local")
	if err != nil {
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if ifNoneMatchHeader := ginContext.GetHeader("If-None-Match"); ifNoneMatchHeader != "" && ifNoneMatchHeader == etag {
		ginContext.Status(http.StatusNotModified)
		return
	}

	evalContext := domain.EvaluationContext{
		UserID:     userID,
		TenantID:   "acme",
		Attributes: map[string]string{},
	}

	evaluatedFlags := make(map[string]any)
	for _, flag := range flags {
		evaluatedFlags[flag.Key] = evaluator.Evaluate(flag, evalContext)
	}

	ginContext.Header("ETag", etag)
	ginContext.JSON(http.StatusOK, evaluatedFlags)
}

// SSEStream streams live flag mutation events to connected clients
// GET /api/v1/stream
func (handler *FlagHandler) SSEStream(ginContext *gin.Context) {
	ginContext.Writer.Header().Set("Content-Type", "text/event-stream")
	ginContext.Writer.Header().Set("Cache-Control", "no-cache, no-transform")
	ginContext.Writer.Header().Set("Connection", "keep-alive")
	ginContext.Writer.Header().Set("X-Accel-Buffering", "no")
	ginContext.Writer.Header().Set("Access-Control-Allow-Origin", "*")

	// Flush HTTP headers and initial connection ping immediately
	ginContext.Writer.WriteString(":connected\n\n")
	ginContext.Writer.Flush()

	clientChannel := make(chan string, 10)

	handler.subscribersMutex.Lock()
	handler.subscribers[clientChannel] = true
	handler.subscribersMutex.Unlock()

	defer func() {
		handler.subscribersMutex.Lock()
		delete(handler.subscribers, clientChannel)
		handler.subscribersMutex.Unlock()
		close(clientChannel)
	}()

	heartbeatTicker := time.NewTicker(15 * time.Second)
	defer heartbeatTicker.Stop()

	ginContext.Stream(func(responseWriter io.Writer) bool {
		select {
		case eventMessagePayload, isChannelOpen := <-clientChannel:
			if !isChannelOpen {
				return false
			}
			fmt.Fprintf(responseWriter, "data: %s\n\n", eventMessagePayload)
			ginContext.Writer.Flush()
			return true
		case <-heartbeatTicker.C:
			// Heartbeat comment ping
			ginContext.Writer.WriteString(":keepalive\n\n")
			ginContext.Writer.Flush()
			return true
		case <-ginContext.Request.Context().Done():
			return false
		}
	})
}

// getOrFetchFlags tries Redis cache first, falling back to DynamoDB Local, and finally in-memory fallback
func (handler *FlagHandler) getOrFetchFlags(requestContext context.Context, tenantIdentifier, environmentName string) ([]domain.FeatureFlag, string, error) {
	// Try Redis if available
	if handler.cache != nil {
		cachedFlags, cachedETag, fetchError := handler.cache.GetRuleset(requestContext, tenantIdentifier, environmentName)
		if fetchError == nil && cachedFlags != nil && len(cachedFlags) > 0 {
			return cachedFlags, cachedETag, nil
		}
	}

	var featureFlags []domain.FeatureFlag

	// Single-Table DynamoDB Query if client is initialized
	if handler.dynamoDbClient != nil {
		databaseContext, cancelTimeout := context.WithTimeout(requestContext, 500*time.Millisecond)
		defer cancelTimeout()
		partitionKey := fmt.Sprintf("TENANT#%s#ENV#%s", tenantIdentifier, environmentName)
		queryOutput, queryError := handler.dynamoDbClient.Query(databaseContext, &dynamodb.QueryInput{
			TableName:              aws.String(handler.tableName),
			KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :skPrefix)"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk":       &types.AttributeValueMemberS{Value: partitionKey},
				":skPrefix": &types.AttributeValueMemberS{Value: "FLAG#"},
			},
		})
		if queryError == nil && queryOutput != nil {
			for _, rawDynamoItem := range queryOutput.Items {
				var parsedFlag domain.FeatureFlag
				if unmarshalError := attributevalue.UnmarshalMap(rawDynamoItem, &parsedFlag); unmarshalError == nil {
					featureFlags = append(featureFlags, parsedFlag)
				}
			}
		}
	}

	// Fallback to in-memory store if DynamoDB returned no flags
	if len(featureFlags) == 0 {
		handler.inMemoryMutex.RLock()
		for _, memoryFlag := range handler.inMemoryFlags {
			featureFlags = append(featureFlags, memoryFlag)
		}
		handler.inMemoryMutex.RUnlock()
	}

	// Compute ETag hash
	marshaledFlagsBytes, _ := json.Marshal(featureFlags)
	md5HashChecksum := md5.Sum(marshaledFlagsBytes)
	computedETagHeader := fmt.Sprintf(`"%s"`, hex.EncodeToString(md5HashChecksum[:]))

	// Store in Redis if available
	if handler.cache != nil {
		_ = handler.cache.SetRuleset(requestContext, tenantIdentifier, environmentName, featureFlags, computedETagHeader)
	}

	return featureFlags, computedETagHeader, nil
}
