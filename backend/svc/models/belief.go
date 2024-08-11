package models

import (
	pbmodels "epistemic-me-backend/pb/models"
	"epistemic-me-backend/svc/models"
)

// ConfidenceRating represents a confidence score for a belief.
type ConfidenceRating struct {
	ConfidenceScore float64 `json:"confidence_score"`
	Default         bool    `json:"default"`
}

func (cr ConfidenceRating) ToProto() *pbmodels.ConfidenceRating {
	return &pbmodels.ConfidenceRating{
		ConfidenceScore: cr.ConfidenceScore,
		Default:         cr.Default,
	}
}

// Content represents the content of a belief in natural language.
type Content struct {
	RawStr string `json:"raw_str"`
}

func (c Content) ToProto() *pbmodels.Content {
	return &pbmodels.Content{
		RawStr: c.RawStr,
	}
}

// BeliefType represents the type of belief, either causal or statement.
type BeliefType int32

// todo: @deen this may imply a state machine on beleifs
// hypothesis -> revisit
// a belief begins as a statement, may be clarified and updated, and
// eventually instantiated as a habit. when a habit is instantiated
// a belief is "locked" until the observation context associated with
// a belief ends
const (
	Causal        BeliefType = 0
	Statement     BeliefType = 1
	Clarification BeliefType = 2
)

func (bt BeliefType) ToProto() pbmodels.BeliefType {
	switch bt {
	case Causal:
		return pbmodels.BeliefType_CAUSAL
	case Statement:
		return pbmodels.BeliefType_STATEMENT
	default:
		return pbmodels.BeliefType_STATEMENT // Default case
	}
}

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

func (b Belief) GetContentAsString() string {
	var contentStrings string
	for _, content := range b.Content {
		contentStrings += content.RawStr
	}
	return contentStrings
}

func (b Belief) ToProto() *pbmodels.Belief {
	confidenceRatingsPb := make([]*pbmodels.ConfidenceRating, len(b.ConfidenceRatings))
	for i, cr := range b.ConfidenceRatings {
		confidenceRatingsPb[i] = cr.ToProto()
	}

	sourcesPb := make([]*pbmodels.Source, len(b.Sources))
	for i, s := range b.Sources {
		sourcesPb[i] = s.ToProto()
	}

	contentPb := make([]*pbmodels.Content, len(b.Content))
	for i, c := range b.Content {
		contentPb[i] = c.ToProto()
	}

	var causalBeliefPb *pbmodels.Belief_CausalBelief
	if b.CausalBelief != nil {
		causalBeliefPb = b.CausalBelief.ToProto()
	}

	var temporalInfoPb *pbmodels.TemporalInformation
	if b.TemporalInformation != nil {
		temporalInfoPb = b.TemporalInformation.ToProto()
	}

	return &pbmodels.Belief{
		Id:                  b.ID,
		UserId:              b.UserID,
		Version:             b.Version,
		ConfidenceRatings:   confidenceRatingsPb,
		Sources:             sourcesPb,
		Content:             contentPb,
		Type:                b.Type.ToProto(),
		CausalBelief:        causalBeliefPb,
		TemporalInformation: temporalInfoPb,
	}
}

// CausalBelief represents the details of a causal belief.
type CausalBelief struct {
	InterventionID         int32  `json:"intervention_id"`
	InterventionName       string `json:"intervention_name"`
	ObservationContextID   int32  `json:"observation_context_id"`
	ObservationContextName string `json:"observation_context_name"`
}

func (cb CausalBelief) ToProto() *pbmodels.Belief_CausalBelief {
	return &pbmodels.Belief_CausalBelief{
		InterventionId:         cb.InterventionID,
		InterventionName:       cb.InterventionName,
		ObservationContextId:   cb.ObservationContextID,
		ObservationContextName: cb.ObservationContextName,
	}
}

// BeliefSystem represents a summary of a user's beliefs.
type BeliefSystem struct {
	RawStr                  string        `json:"raw_str"`
	OverallConfidenceRating float64       `json:"overall_confidence_rating"`
	ClarifiedMetric         models.Metric `json:"conflict_score"`
}

func (bs BeliefSystem) ToProto() *pbmodels.BeliefSystem {
	return &pbmodels.BeliefSystem{
		RawStr:                  bs.RawStr,
		OverallConfidenceRating: bs.OverallConfidenceRating,
		ClarificationScore:      bs.ClarifiedMetric.ToPercentage(),
	}
}

// Assuming Source and TemporalInformation are defined elsewhere
type Source struct {
	// Fields for Source
}

func (s Source) ToProto() *pbmodels.Source {
	// Implement the conversion
	return &pbmodels.Source{}
}

type TemporalInformation struct {
	// Fields for TemporalInformation
}

func (ti TemporalInformation) ToProto() *pbmodels.TemporalInformation {
	// Implement the conversion
	return &pbmodels.TemporalInformation{}
}
