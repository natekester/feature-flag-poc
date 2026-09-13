package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"flag-service/internal/domain"
)

type CachedRuleset struct {
	ETag  string               `json:"etag"`
	Flags []domain.FeatureFlag `json:"flags"`
}

type RulesetCache struct {
	redisClient *redis.Client
}

func NewRulesetCache(redisServerAddress string) *RulesetCache {
	redisClient := redis.NewClient(&redis.Options{
		Addr:         redisServerAddress,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
		PoolSize:     20,
	})
	return &RulesetCache{redisClient: redisClient}
}

// GetRuleset returns cached rules and ETag (<1ms)
func (rulesetCache *RulesetCache) GetRuleset(requestContext context.Context, tenantIdentifier, environmentName string) ([]domain.FeatureFlag, string, error) {
	if rulesetCache.redisClient == nil {
		return nil, "", nil
	}
	cacheKey := fmt.Sprintf("ruleset:%s:%s", tenantIdentifier, environmentName)
	rawCachedString, redisError := rulesetCache.redisClient.Get(requestContext, cacheKey).Result()
	if redisError == redis.Nil {
		return nil, "", nil // Cache Miss
	} else if redisError != nil {
		return nil, "", redisError // Redis fail-open to DB
	}

	var cachedRulesetPayload CachedRuleset
	if unmarshalError := json.Unmarshal([]byte(rawCachedString), &cachedRulesetPayload); unmarshalError != nil {
		return nil, "", unmarshalError
	}
	return cachedRulesetPayload.Flags, cachedRulesetPayload.ETag, nil
}

// SetRuleset populates Redis with compiled flags and computed ETag
func (rulesetCache *RulesetCache) SetRuleset(requestContext context.Context, tenantIdentifier, environmentName string, featureFlags []domain.FeatureFlag, etagHeaderValue string) error {
	if rulesetCache.redisClient == nil {
		return nil
	}
	cacheKey := fmt.Sprintf("ruleset:%s:%s", tenantIdentifier, environmentName)
	serializedPayloadBytes, marshalError := json.Marshal(CachedRuleset{ETag: etagHeaderValue, Flags: featureFlags})
	if marshalError != nil {
		return marshalError
	}
	return rulesetCache.redisClient.Set(requestContext, cacheKey, serializedPayloadBytes, 24*time.Hour).Err()
}

// Invalidate purges cache when an admin mutates a flag or override
func (rulesetCache *RulesetCache) Invalidate(requestContext context.Context, tenantIdentifier, environmentName string) error {
	if rulesetCache.redisClient == nil {
		return nil
	}
	cacheKey := fmt.Sprintf("ruleset:%s:%s", tenantIdentifier, environmentName)
	return rulesetCache.redisClient.Del(requestContext, cacheKey).Err()
}
