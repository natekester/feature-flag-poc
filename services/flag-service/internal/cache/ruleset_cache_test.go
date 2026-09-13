package cache

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"flag-service/internal/domain"
)

func TestRulesetCache(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	cache := NewRulesetCache(mr.Addr())
	ctx := context.Background()

	// 1. Get empty cache
	flags, etag, err := cache.GetRuleset(ctx, "acme", "local")
	assert.NoError(t, err)
	assert.Nil(t, flags)
	assert.Empty(t, etag)

	// 2. Set cache
	testFlags := []domain.FeatureFlag{
		{
			Key:          "ds-button-v2",
			Enabled:      true,
			DefaultValue: "v1",
			UpdatedAt:    123456789,
		},
	}
	err = cache.SetRuleset(ctx, "acme", "local", testFlags, "etag-123")
	assert.NoError(t, err)

	// 3. Get cached data
	cachedFlags, cachedETag, err := cache.GetRuleset(ctx, "acme", "local")
	assert.NoError(t, err)
	assert.Equal(t, "etag-123", cachedETag)
	assert.Len(t, cachedFlags, 1)
	assert.Equal(t, "ds-button-v2", cachedFlags[0].Key)

	// 4. Invalidate cache
	err = cache.Invalidate(ctx, "acme", "local")
	assert.NoError(t, err)

	flags, etag, err = cache.GetRuleset(ctx, "acme", "local")
	assert.NoError(t, err)
	assert.Nil(t, flags)
	assert.Empty(t, etag)
}

func TestNilRulesetCache(t *testing.T) {
	cache := &RulesetCache{rdb: nil}
	ctx := context.Background()

	flags, etag, err := cache.GetRuleset(ctx, "acme", "local")
	assert.NoError(t, err)
	assert.Nil(t, flags)
	assert.Empty(t, etag)

	err = cache.SetRuleset(ctx, "acme", "local", nil, "")
	assert.NoError(t, err)

	err = cache.Invalidate(ctx, "acme", "local")
	assert.NoError(t, err)
}
