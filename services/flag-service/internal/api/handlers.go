package api

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	ddbClient *dynamodb.Client
	tableName string
	cache     *cache.RulesetCache
	sseChan   chan string
}

func NewFlagHandler(client *dynamodb.Client, table string, c *cache.RulesetCache) *FlagHandler {
	return &FlagHandler{
		ddbClient: client,
		tableName: table,
		cache:     c,
		sseChan:   make(chan string, 100),
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
	userID := c.Param("userId")

	var req SetOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pk := "TENANT#acme#ENV#local"
	sk := fmt.Sprintf("FLAG#%s", flagKey)

	valAV, err := attributevalue.Marshal(req.Variation)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "serialization error"})
		return
	}

	// 1. UpdateItem in DynamoDB
	_, err = h.ddbClient.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
		TableName: aws.String(h.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
		UpdateExpression: aws.String("SET UserOverrides.#uid = :val, UpdatedAt = :now"),
		ExpressionAttributeNames: map[string]string{
			"#uid": userID,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":val": valAV,
			":now": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", time.Now().Unix())},
		},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 2. Invalidate local Redis cache
	_ = h.cache.Invalidate(c.Request.Context(), "acme", "local")

	// 3. Broadcast SSE event
	eventMsg := fmt.Sprintf(`{"type":"FLAG_OVERRIDE_UPDATED","flagKey":"%s","userId":"%s","variation":%v}`, flagKey, userID, req.Variation)
	select {
	case h.sseChan <- eventMsg:
	default:
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"flagKey":   flagKey,
		"userId":    userID,
		"variation": req.Variation,
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
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	c.Stream(func(w io.Writer) bool {
		select {
		case msg, ok := <-h.sseChan:
			if !ok {
				return false
			}
			c.SSEvent("message", msg)
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

// getOrFetchFlags tries Redis cache first, falling back to DynamoDB Local
func (h *FlagHandler) getOrFetchFlags(ctx context.Context, tenant, env string) ([]domain.FeatureFlag, string, error) {
	// Try Redis
	cachedFlags, etag, err := h.cache.GetRuleset(ctx, tenant, env)
	if err == nil && cachedFlags != nil {
		return cachedFlags, etag, nil
	}

	// Single-Table DynamoDB Query
	pk := fmt.Sprintf("TENANT#%s#ENV#%s", tenant, env)
	out, err := h.ddbClient.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(h.tableName),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :skPrefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":       &types.AttributeValueMemberS{Value: pk},
			":skPrefix": &types.AttributeValueMemberS{Value: "FLAG#"},
		},
	})
	if err != nil {
		return nil, "", err
	}

	var flags []domain.FeatureFlag
	for _, item := range out.Items {
		var flag domain.FeatureFlag
		if err := attributevalue.UnmarshalMap(item, &flag); err == nil {
			flags = append(flags, flag)
		}
	}

	// Compute ETag hash
	bytes, _ := json.Marshal(flags)
	hash := md5.Sum(bytes)
	computedETag := fmt.Sprintf(`"%s"`, hex.EncodeToString(hash[:]))

	// Store in Redis
	_ = h.cache.SetRuleset(ctx, tenant, env, flags, computedETag)

	return flags, computedETag, nil
}
