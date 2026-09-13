package evaluator

import (
	"fmt"

	"github.com/spaolacci/murmur3"

	"flag-service/internal/domain"
)

// CalculateBucket generates a deterministic 0-99 bucket matching the React/TypeScript SDK
func CalculateBucket(userID string, flagKey string) int {
	seed := fmt.Sprintf("%s:%s", userID, flagKey)
	hash := murmur3.Sum32([]byte(seed))
	return int(hash % 100)
}

// Evaluate determines the active variation for a given user context in <0.01ms
func Evaluate(flag domain.FeatureFlag, ctx domain.EvaluationContext) domain.VariationValue {
	if !flag.Enabled {
		return flag.DefaultValue
	}

	// 1. Explicit User-Specific Overrides (Highest Priority for Admin POC)
	if flag.UserOverrides != nil {
		if overrideVal, exists := flag.UserOverrides[ctx.UserID]; exists {
			return overrideVal
		}
	}

	// 2. Targeting Rules (Attribute Matching)
	for _, rule := range flag.TargetingRules {
		if val, exists := ctx.Attributes[rule.Attribute]; exists {
			for _, targetVal := range rule.Values {
				if val == targetVal {
					return rule.Variation
				}
			}
		}
	}

	// 3. Percentage Rollout Bucketing
	if len(flag.Rollout) > 0 {
		bucket := CalculateBucket(ctx.UserID, flag.Key)
		cumulative := 0
		for _, variant := range flag.Rollout {
			cumulative += variant.BucketPercentage
			if bucket < cumulative {
				return variant.Variation
			}
		}
	}

	return flag.DefaultValue
}
