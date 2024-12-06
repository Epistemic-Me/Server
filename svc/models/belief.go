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

type ConfidenceRating struct {
	ConfidenceScore float64 `json:"confidence_score"`
	Default         bool    `json:"default"`
}

type ObservationContext struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	ParentID       string   `json:"parent_id"`
	PossibleStates []string `json:"possible_states"`
}

func (oc ObservationContext) ToProto() *pbmodels.ObservationContext {
	states := make([]*pbmodels.State, len(oc.PossibleStates))
	for i, s := range oc.PossibleStates {
		states[i] = &pbmodels.State{Name: s}
	}
	return &pbmodels.ObservationContext{
		Id:             oc.ID,
		Name:           oc.Name,
		ParentId:       oc.ParentID,
		PossibleStates: states,
	}
}

type EpistemicEmotion int32

const (
	EmotionInvalid EpistemicEmotion = iota
	Confirmation
	Surprise
	Curiosity
	Confusion
)

type ObservationType int32

const (
	ObservationInvalid ObservationType = iota
	Answer
	ObservationEvidence
	Outcome
)

// Discrepancy represents the difference between prediction and observation
type Discrepancy struct {
	DialecticInteractionID string             `json:"dialectic_interaction_id"`
	PriorProbabilities     map[string]float32 `json:"prior_probabilities"`
	PosteriorProbabilities map[string]float32 `json:"posterior_probabilities"`
	IsCounterfactual       bool               `json:"is_counterfactual"`
	Timestamp              int64              `json:"timestamp"`
	KlDivergence           float32            `json:"kl_divergence"`
	PointwiseKlTerms       map[string]float32 `json:"pointwise_kl_terms"`
}

// BeliefContext represents the relationship between a Belief and an ObservationContext
type BeliefContext struct {
	BeliefID                string             `json:"belief_id"`
	ObservationContextID    string             `json:"observation_context_id"`
	ConfidenceRatings       []ConfidenceRating `json:"confidence_ratings"`
	Evidence                []*Source          `json:"evidence"`
	Action                  string             `json:"action,omitempty"`
	ExpectedResult          string             `json:"expected_result,omitempty"`
	ConditionalProbs        map[string]float32 `json:"conditional_probabilities,omitempty"`
	DialecticInteractionIDs []string           `json:"dialectic_interaction_ids"`
	EpistemicEmotion        EpistemicEmotion   `json:"epistemic_emotion"`
	EmotionIntensity        float32            `json:"emotion_intensity"`
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
	Beliefs             []*Belief             `json:"beliefs"`
	ObservationContexts []*ObservationContext `json:"observation_contexts"`
	BeliefContexts      []*BeliefContext      `json:"belief_contexts"`
	Metrics             *BeliefSystemMetrics  `json:"metrics,omitempty"`
	Ontology            *Ontology             `json:"ontology,omitempty"`
}

type BeliefSystemMetrics struct {
	ClarificationScore      float64 `json:"clarification_score"`
	TotalBeliefs            int32   `json:"total_beliefs"`
	TotalFalsifiableBeliefs int32   `json:"total_falsifiable_beliefs"`
	TotalCausalBeliefs      int32   `json:"total_causal_beliefs"`
	TotalBeliefStatements   int32   `json:"total_belief_statements"`
}

// Update Ontology struct to use BeliefContexts instead of ObservationContexts
type Ontology struct {
	RawStr      string           `json:"raw_str"`
	GeneratedAt int64            `json:"generated_at"`
	Contexts    []*BeliefContext `json:"contexts"` // Changed from ObservationContext to BeliefContext
}

func ontologyToProto(o *Ontology) *pbmodels.BeliefSystem_Ontology {
	if o == nil {
		return nil
	}
	protoContexts := make([]*pbmodels.BeliefContext, len(o.Contexts))
	for i, ctx := range o.Contexts {
		protoContexts[i] = ctx.ToProto()
	}
	return &pbmodels.BeliefSystem_Ontology{
		RawStr:      o.RawStr,
		GeneratedAt: o.GeneratedAt,
		Contexts:    protoContexts,
	}
}

func (bs BeliefSystem) ToProto() *pbmodels.BeliefSystem {
	protoBeliefs := make([]*pbmodels.Belief, len(bs.Beliefs))
	for i, belief := range bs.Beliefs {
		protoBeliefs[i] = belief.ToProto()
	}
	protoObservationContexts := make([]*pbmodels.ObservationContext, len(bs.ObservationContexts))
	for i, oc := range bs.ObservationContexts {
		protoObservationContexts[i] = oc.ToProto()
	}
	return &pbmodels.BeliefSystem{
		Beliefs:             protoBeliefs,
		ObservationContexts: protoObservationContexts,
		Metrics:             metricsToProto(bs.Metrics),
		Ontology:            ontologyToProto(bs.Ontology),
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

func (bc BeliefContext) ToProto() *pbmodels.BeliefContext {
	return &pbmodels.BeliefContext{
		BeliefId:                 bc.BeliefID,
		ObservationContextId:     bc.ObservationContextID,
		ConfidenceRatings:        confidenceRatingsToProto(bc.ConfidenceRatings),
		ConditionalProbabilities: bc.ConditionalProbs,
		DiscrepancyIds:           bc.DialecticInteractionIDs,
		EpistemicEmotion:         pbmodels.EpistemicEmotion(bc.EpistemicEmotion),
		EmotionIntensity:         bc.EmotionIntensity,
	}
}

func discrepanciesToProto(discrepancies []Discrepancy) []*pbmodels.Discrepancy {
	result := make([]*pbmodels.Discrepancy, len(discrepancies))
	for i, d := range discrepancies {
		result[i] = d.ToProto()
	}
	return result
}

func observationContextsToProto(contexts []*ObservationContext) []*pbmodels.ObservationContext {
	result := make([]*pbmodels.ObservationContext, len(contexts))
	for i, c := range contexts {
		states := make([]*pbmodels.State, len(c.PossibleStates))
		for j, s := range c.PossibleStates {
			states[j] = &pbmodels.State{Name: s}
		}
		result[i] = &pbmodels.ObservationContext{
			Id:             c.ID,
			Name:           c.Name,
			ParentId:       c.ParentID,
			PossibleStates: states,
		}
	}
	return result
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

func metricsToProto(m *BeliefSystemMetrics) *pbmodels.BeliefSystem_Metrics {
	if m == nil {
		return nil
	}
	return &pbmodels.BeliefSystem_Metrics{
		ClarificationScore:      m.ClarificationScore,
		TotalBeliefs:            m.TotalBeliefs,
		TotalFalsifiableBeliefs: m.TotalFalsifiableBeliefs,
		TotalCausalBeliefs:      m.TotalCausalBeliefs,
		TotalBeliefStatements:   m.TotalBeliefStatements,
	}
}

func (d Discrepancy) ToProto() *pbmodels.Discrepancy {
	return &pbmodels.Discrepancy{
		DialecticInteractionId: d.DialecticInteractionID,
		KlDivergence:           float64(d.KlDivergence),
		PointwiseKlTerms:       d.PointwiseKlTerms,
		Timestamp:              d.Timestamp,
	}
}

func (b *Belief) GetContentAsString() string {
	var contentStrings []string
	for _, content := range b.Content {
		contentStrings = append(contentStrings, content.RawStr)
	}
	return strings.Join(contentStrings, " ")
}
