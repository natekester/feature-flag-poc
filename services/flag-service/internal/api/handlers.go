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
	ddbClient     *dynamodb.Client
	tableName     string
	cache         *cache.RulesetCache
	subscribers   map[chan string]bool
	subMu         sync.RWMutex
	inMemoryFlags map[string]domain.FeatureFlag
	mu            sync.RWMutex
}

func NewFlagHandler(client *dynamodb.Client, table string, c *cache.RulesetCache) *FlagHandler {
	defaultFlag := domain.FeatureFlag{
		Key:           "ds-button-v2",
		Enabled:       true,
		DefaultValue:  "v1",
		Rollout:       nil,
		UserOverrides: map[string]domain.VariationValue{},
		UpdatedAt:     time.Now().Unix(),
	}

	return &FlagHandler{
		ddbClient:   client,
		tableName:   table,
		cache:       c,
		subscribers: make(map[chan string]bool),
		inMemoryFlags: map[string]domain.FeatureFlag{
			"ds-button-v2": defaultFlag,
		},
	}
}

func (h *FlagHandler) broadcastSSE(msg string) {
	h.subMu.RLock()
	defer h.subMu.RUnlock()
	for ch := range h.subscribers {
		select {
		case ch <- msg:
		default:
		}
	}
}

type SetOverrideRequest struct {
	Variation any `json:"variation" binding:"required"`
}

// ListFlags retrieves all flags for tenant & env
// GET /api/v1/admin/flags
func (h *FlagHandler) ListFlags(c *gin.Context) {
	flags, _, err := h.getOrFetchFlags(c.Request.Context(), "acme", "local")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, flags)
}

// SetUserOverride updates a flag's user override in DynamoDB and invalidates Redis
// PUT /api/v1/admin/flags/:key/overrides/users/:userId
func (h *FlagHandler) SetUserOverride(c *gin.Context) {
	flagKey := c.Param("key")
	userID := strings.TrimPrefix(c.Param("userId"), "/")

	var req SetOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Always update in-memory cache as fallback
	h.mu.Lock()
	if flag, exists := h.inMemoryFlags[flagKey]; exists {
		if flag.UserOverrides == nil {
			flag.UserOverrides = make(map[string]domain.VariationValue)
		}
		flag.UserOverrides[userID] = req.Variation
		flag.UpdatedAt = time.Now().Unix()
		h.inMemoryFlags[flagKey] = flag
	}
	h.mu.Unlock()

	pk := "TENANT#acme#ENV#local"
	sk := fmt.Sprintf("FLAG#%s", flagKey)

	valAV, err := attributevalue.Marshal(req.Variation)
	if err == nil && h.ddbClient != nil {
		// Attempt UpdateItem in DynamoDB
		_, _ = h.ddbClient.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
			TableName: aws.String(h.tableName),
			Key: map[string]types.AttributeValue{
				"PK": &types.AttributeValueMemberS{Value: pk},
				"SK": &types.AttributeValueMemberS{Value: sk},
			},
			UpdateExpression: aws.String("SET UserOverrides = if_not_exists(UserOverrides, :empty_map), UserOverrides.#uid = :val, UpdatedAt = :now"),
			ExpressionAttributeNames: map[string]string{
				"#uid": userID,
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":val":       valAV,
				":empty_map": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{}},
				":now":       &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", time.Now().Unix())},
			},
		})
	}

	// Invalidate local Redis cache if available
	if h.cache != nil {
		_ = h.cache.Invalidate(c.Request.Context(), "acme", "local")
	}

	// Broadcast SSE event
	varBytes, _ := json.Marshal(req.Variation)
	eventMsg := fmt.Sprintf(`{"type":"FLAG_OVERRIDE_UPDATED","flagKey":"%s","userId":"%s","variation":%s}`, flagKey, userID, string(varBytes))
	h.broadcastSSE(eventMsg)

	c.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"flagKey":   flagKey,
		"userId":    userID,
		"variation": req.Variation,
	})
}

// RemoveUserOverride deletes a flag's user override in DynamoDB and invalidates Redis
// DELETE /api/v1/admin/flags/:key/overrides/users/:userId
func (h *FlagHandler) RemoveUserOverride(c *gin.Context) {
	flagKey := c.Param("key")
	userID := strings.TrimPrefix(c.Param("userId"), "/")

	// Update in-memory cache as fallback
	h.mu.Lock()
	if flag, exists := h.inMemoryFlags[flagKey]; exists {
		if flag.UserOverrides != nil {
			delete(flag.UserOverrides, userID)
			flag.UpdatedAt = time.Now().Unix()
			h.inMemoryFlags[flagKey] = flag
		}
	}
	h.mu.Unlock()

	pk := "TENANT#acme#ENV#local"
	sk := fmt.Sprintf("FLAG#%s", flagKey)

	if h.ddbClient != nil {
		_, _ = h.ddbClient.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
			TableName: aws.String(h.tableName),
			Key: map[string]types.AttributeValue{
				"PK": &types.AttributeValueMemberS{Value: pk},
				"SK": &types.AttributeValueMemberS{Value: sk},
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
	if h.cache != nil {
		_ = h.cache.Invalidate(c.Request.Context(), "acme", "local")
	}

	// Broadcast SSE event
	eventMsg := fmt.Sprintf(`{"type":"FLAG_OVERRIDE_REMOVED","flagKey":"%s","userId":"%s"}`, flagKey, userID)
	h.broadcastSSE(eventMsg)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"flagKey": flagKey,
		"userId":  userID,
	})
}

// EvaluateUser evaluates all flags for a given userId
// GET /api/v1/evaluate?userId=...
func (h *FlagHandler) EvaluateUser(c *gin.Context) {
	userID := c.Query("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId query param required"})
		return
	}

	flags, etag, err := h.getOrFetchFlags(c.Request.Context(), "acme", "local")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if ifNoneMatch := c.GetHeader("If-None-Match"); ifNoneMatch != "" && ifNoneMatch == etag {
		c.Status(http.StatusNotModified)
		return
	}

	evalContext := domain.EvaluationContext{
		UserID:     userID,
		TenantID:   "acme",
		Attributes: map[string]string{},
	}

	evaluated := make(map[string]any)
	for _, flag := range flags {
		evaluated[flag.Key] = evaluator.Evaluate(flag, evalContext)
	}

	c.Header("ETag", etag)
	c.JSON(http.StatusOK, evaluated)
}

// SSEStream streams live flag mutation events to connected clients
// GET /api/v1/stream
func (h *FlagHandler) SSEStream(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache, no-transform")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

	// Flush HTTP headers and initial connection ping immediately
	c.Writer.WriteString(":connected\n\n")
	c.Writer.Flush()

	clientChan := make(chan string, 10)

	h.subMu.Lock()
	h.subscribers[clientChan] = true
	h.subMu.Unlock()

	defer func() {
		h.subMu.Lock()
		delete(h.subscribers, clientChan)
		h.subMu.Unlock()
		close(clientChan)
	}()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	c.Stream(func(w io.Writer) bool {
		select {
		case msg, ok := <-clientChan:
			if !ok {
				return false
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			c.Writer.Flush()
			return true
		case <-ticker.C:
			// Heartbeat comment ping
			c.Writer.WriteString(":keepalive\n\n")
			c.Writer.Flush()
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

// getOrFetchFlags tries Redis cache first, falling back to DynamoDB Local, and finally in-memory fallback
func (h *FlagHandler) getOrFetchFlags(ctx context.Context, tenant, env string) ([]domain.FeatureFlag, string, error) {
	// Try Redis if available
	if h.cache != nil {
		cachedFlags, etag, err := h.cache.GetRuleset(ctx, tenant, env)
		if err == nil && cachedFlags != nil && len(cachedFlags) > 0 {
			return cachedFlags, etag, nil
		}
	}

	var flags []domain.FeatureFlag

	// Single-Table DynamoDB Query if client is initialized
	if h.ddbClient != nil {
		dbCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		defer cancel()
		pk := fmt.Sprintf("TENANT#%s#ENV#%s", tenant, env)
		out, err := h.ddbClient.Query(dbCtx, &dynamodb.QueryInput{
			TableName:              aws.String(h.tableName),
			KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :skPrefix)"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk":       &types.AttributeValueMemberS{Value: pk},
				":skPrefix": &types.AttributeValueMemberS{Value: "FLAG#"},
			},
		})
		if err == nil && out != nil {
			for _, item := range out.Items {
				var flag domain.FeatureFlag
				if err := attributevalue.UnmarshalMap(item, &flag); err == nil {
					flags = append(flags, flag)
				}
			}
		}
	}

	// Fallback to in-memory store if DynamoDB returned no flags
	if len(flags) == 0 {
		h.mu.RLock()
		for _, f := range h.inMemoryFlags {
			flags = append(flags, f)
		}
		h.mu.RUnlock()
	}

	// Compute ETag hash
	bytes, _ := json.Marshal(flags)
	hash := md5.Sum(bytes)
	computedETag := fmt.Sprintf(`"%s"`, hex.EncodeToString(hash[:]))

	// Store in Redis if available
	if h.cache != nil {
		_ = h.cache.SetRuleset(ctx, tenant, env, flags, computedETag)
	}

	return flags, computedETag, nil
}
