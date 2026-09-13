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
	rdb *redis.Client
}

func NewRulesetCache(redisAddr string) *RulesetCache {
	rdb := redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
		PoolSize:     20,
	})
	return &RulesetCache{rdb: rdb}
}

// GetRuleset returns cached rules and ETag (<1ms)
func (c *RulesetCache) GetRuleset(ctx context.Context, tenant, env string) ([]domain.FeatureFlag, string, error) {
	if c.rdb == nil {
		return nil, "", nil
	}
	key := fmt.Sprintf("ruleset:%s:%s", tenant, env)
	val, err := c.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, "", nil // Cache Miss
	} else if err != nil {
		return nil, "", err // Redis fail-open to DB
	}

	var cached CachedRuleset
	if err := json.Unmarshal([]byte(val), &cached); err != nil {
		return nil, "", err
	}
	return cached.Flags, cached.ETag, nil
}

// SetRuleset populates Redis with compiled flags and computed ETag
func (c *RulesetCache) SetRuleset(ctx context.Context, tenant, env string, flags []domain.FeatureFlag, etag string) error {
	if c.rdb == nil {
		return nil
	}
	key := fmt.Sprintf("ruleset:%s:%s", tenant, env)
	payload, err := json.Marshal(CachedRuleset{ETag: etag, Flags: flags})
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, key, payload, 24*time.Hour).Err()
}

// Invalidate purges cache when an admin mutates a flag or override
func (c *RulesetCache) Invalidate(ctx context.Context, tenant, env string) error {
	if c.rdb == nil {
		return nil
	}
	key := fmt.Sprintf("ruleset:%s:%s", tenant, env)
	return c.rdb.Del(ctx, key).Err()
}
