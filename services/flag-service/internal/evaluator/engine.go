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
func Evaluate(featureFlag domain.FeatureFlag, evaluationContext domain.EvaluationContext) domain.VariationValue {
	if !featureFlag.Enabled {
		return featureFlag.DefaultValue
	}

	// 1. Explicit User-Specific Overrides (Highest Priority for Admin POC)
	if featureFlag.UserOverrides != nil {
		if overrideValue, hasOverride := featureFlag.UserOverrides[evaluationContext.UserID]; hasOverride {
			return overrideValue
		}
	}

	// 2. Targeting Rules (Attribute Matching)
	for _, targetingRule := range featureFlag.TargetingRules {
		if attributeValue, attributeExists := evaluationContext.Attributes[targetingRule.Attribute]; attributeExists {
			for _, targetValue := range targetingRule.Values {
				if attributeValue == targetValue {
					return targetingRule.Variation
				}
			}
		}
	}

	// 3. Percentage Rollout Bucketing
	if len(featureFlag.Rollout) > 0 {
		userBucketValue := CalculateBucket(evaluationContext.UserID, featureFlag.Key)
		cumulativeBucketSum := 0
		for _, rolloutVariant := range featureFlag.Rollout {
			cumulativeBucketSum += rolloutVariant.BucketPercentage
			if userBucketValue < cumulativeBucketSum {
				return rolloutVariant.Variation
			}
		}
	}

	return featureFlag.DefaultValue
}
