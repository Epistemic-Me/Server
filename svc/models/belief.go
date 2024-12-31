package models

import (
	"strings"

	pbmodels "epistemic-me-core/pb/models"
)

type BeliefType int32

const (
	Statement BeliefType = iota + 1
	Falsifiable
	Causal
	Clarification
)

type Content struct {
	RawStr string `json:"raw_str"`
}

type EpistemicContext struct {
	PredictiveProcessingContext *PredictiveProcessingContext `json:"context,omitempty"`
}

func (ec EpistemicContext) ToProto() *pbmodels.EpistemicContext {
	if ec.PredictiveProcessingContext != nil {
		return &pbmodels.EpistemicContext{
			Context: ec.PredictiveProcessingContext.toProto(),
		}
	}
	return nil
}

// Base Belief structure (simplified)
type Belief struct {
	ID          string     `json:"id"`
	SelfModelID string     `json:"self_model_id"`
	Version     int32      `json:"version"`
	Type        BeliefType `json:"type"`
	Content     []Content  `json:"content"`
}

// BeliefSystem with BeliefContexts
type BeliefSystem struct {
	Beliefs           []*Belief           `json:"beliefs"`
	EpistemicContexts []*EpistemicContext `json:"epistemic_context"`
}

func (bs BeliefSystem) ToProto() *pbmodels.BeliefSystem {
	protoBeliefs := make([]*pbmodels.Belief, len(bs.Beliefs))
	for i, belief := range bs.Beliefs {
		protoBeliefs[i] = belief.ToProto()
	}
	protoEpistemicContexts := make([]*pbmodels.EpistemicContext, len(bs.EpistemicContexts))
	for i, ec := range bs.EpistemicContexts {
		protoEpistemicContexts[i] = ec.ToProto()
	}

	return &pbmodels.BeliefSystem{
		Beliefs: protoBeliefs,
		EpistemicContexts: &pbmodels.EpistemicContexts{
			EpistemicContexts: protoEpistemicContexts,
		},
	}
}

func (b Belief) ToProto() *pbmodels.Belief {
	return &pbmodels.Belief{
		Id:          b.ID,
		SelfModelId: b.SelfModelID,
		Version:     b.Version,
		Type:        pbmodels.BeliefType(b.Type),
		Content:     contentToProto(b.Content),
	}
}

func contentToProto(content []Content) []*pbmodels.Content {
	result := make([]*pbmodels.Content, len(content))
	for i, c := range content {
		result[i] = &pbmodels.Content{RawStr: c.RawStr}
	}
	return result
}

func confidenceRatingsToProto(ratings []ConfidenceRating) []*pbmodels.ConfidenceRating {
	result := make([]*pbmodels.ConfidenceRating, len(ratings))
	for i, r := range ratings {
		result[i] = &pbmodels.ConfidenceRating{
			ConfidenceScore: r.ConfidenceScore,
			Default:         r.Default,
		}
	}
	return result
}

func beliefsToProto(beliefs []*Belief) []*pbmodels.Belief {
	result := make([]*pbmodels.Belief, len(beliefs))
	for i, b := range beliefs {
		result[i] = b.ToProto()
	}
	return result
}

func metricsToProto(m *BeliefSystemMetrics) *pbmodels.Metrics {
	if m == nil {
		return nil
	}
	return &pbmodels.Metrics{
		ClarificationScore:      m.ClarificationScore,
		TotalBeliefs:            m.TotalBeliefs,
		TotalFalsifiableBeliefs: m.TotalFalsifiableBeliefs,
		TotalCausalBeliefs:      m.TotalCausalBeliefs,
		TotalBeliefStatements:   m.TotalBeliefStatements,
	}
}

func (b *Belief) GetContentAsString() string {
	var contentStrings []string
	for _, content := range b.Content {
		contentStrings = append(contentStrings, content.RawStr)
	}
	return strings.Join(contentStrings, " ")
}
