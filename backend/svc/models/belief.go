package models

// ConfidenceRating represents a confidence score for a belief.
type ConfidenceRating struct {
	ConfidenceScore float64 `json:"confidence_score"`
	Default         bool    `json:"default"`
}

// Content represents the content of a belief in natural language.
type Content struct {
	RawStr string `json:"raw_str"`
}

// BeliefType represents the type of belief, either causal or statement.
type BeliefType int32

const (
	Causal    BeliefType = 0
	Statement BeliefType = 1
)

// Belief represents a user's belief.
type Belief struct {
	ID                  string               `json:"id"`
	UserID              string               `json:"user_id"`
	Version             int32                `json:"version"`
	ConfidenceRatings   []ConfidenceRating   `json:"confidence_ratings"`
	Sources             []Source             `json:"sources"`
	Content             []Content            `json:"content"`
	Type                BeliefType           `json:"type"`
	CausalBelief        *CausalBelief        `json:"causal_belief,omitempty"`
	TemporalInformation *TemporalInformation `json:"temporal_information,omitempty"`
}

// CausalBelief represents the details of a causal belief.
type CausalBelief struct {
	InterventionID         int32  `json:"intervention_id"`
	InterventionName       string `json:"intervention_name"`
	ObservationContextID   int32  `json:"observation_context_id"`
	ObservationContextName string `json:"observation_context_name"`
}

// BeliefSystem represents a summary of a user's beliefs.
type BeliefSystem struct {
	RawStr                  string  `json:"raw_str"`
	OverallConfidenceRating float64 `json:"overall_confidence_rating"`
	ConflictScore           float64 `json:"conflict_score"`
}
