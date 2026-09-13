package domain

// VariationValue represents a typed multivariate or boolean value (e.g. "v1", "v2", true, 42)
type VariationValue any

// TargetingRule defines an attribute-matching rule
type TargetingRule struct {
	Attribute string         `json:"attribute" dynamodbav:"attribute"`
	Values    []string       `json:"values" dynamodbav:"values"`
	Variation VariationValue `json:"variation" dynamodbav:"variation"`
}

// RolloutVariation defines bucket percentages for A/B testing
type RolloutVariation struct {
	Variation        VariationValue `json:"variation" dynamodbav:"variation"`
	BucketPercentage int            `json:"bucketPercentage" dynamodbav:"bucketPercentage"`
}

// FeatureFlag is the core configuration entity stored in DynamoDB
type FeatureFlag struct {
	Key            string                    `json:"key" dynamodbav:"Key"`
	Enabled        bool                      `json:"enabled" dynamodbav:"Enabled"`
	DefaultValue   VariationValue            `json:"defaultValue" dynamodbav:"DefaultValue"`
	UserOverrides  map[string]VariationValue `json:"userOverrides,omitempty" dynamodbav:"UserOverrides,omitempty"`
	TargetingRules []TargetingRule           `json:"targetingRules,omitempty" dynamodbav:"TargetingRules,omitempty"`
	Rollout        []RolloutVariation        `json:"rollout,omitempty" dynamodbav:"Rollout,omitempty"`
	UpdatedAt      int64                     `json:"updatedAt" dynamodbav:"UpdatedAt"`
}

// EvaluationContext encapsulates incoming user request parameters
type EvaluationContext struct {
	UserID     string            `json:"userId"`
	TenantID   string            `json:"tenantId"`
	Attributes map[string]string `json:"attributes"`
}
