package evaluator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"flag-service/internal/domain"
	"flag-service/internal/evaluator"
)

func TestEvaluation_ParityAndOverrides(t *testing.T) {
	flag := domain.FeatureFlag{
		Key:          "ds-button-v2",
		Enabled:      true,
		DefaultValue: "v1",
		UserOverrides: map[string]domain.VariationValue{
			"user-admin-99": "v2",
			"user-beta-01":  "compact",
		},
		Rollout: []domain.RolloutVariation{
			{Variation: "v1", BucketPercentage: 50},
			{Variation: "v2", BucketPercentage: 50},
		},
	}

	t.Run("Explicit user override takes precedence", func(t *testing.T) {
		ctx := domain.EvaluationContext{UserID: "user-admin-99"}
		result := evaluator.Evaluate(flag, ctx)
		assert.Equal(t, "v2", result)
	})

	t.Run("Non-overridden user falls through to deterministic rollout", func(t *testing.T) {
		ctx := domain.EvaluationContext{UserID: "user-regular-01"}
		bucket := evaluator.CalculateBucket(ctx.UserID, flag.Key)
		result := evaluator.Evaluate(flag, ctx)

		if bucket < 50 {
			assert.Equal(t, "v1", result)
		} else {
			assert.Equal(t, "v2", result)
		}
	})

	t.Run("Disabled flag returns default value unconditionally", func(t *testing.T) {
		disabledFlag := flag
		disabledFlag.Enabled = false
		ctx := domain.EvaluationContext{UserID: "user-admin-99"}
		assert.Equal(t, "v1", evaluator.Evaluate(disabledFlag, ctx))
	})
}
