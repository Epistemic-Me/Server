package models

import (
	pbmodels "epistemic-me-core/pb/models"
)

type PredictiveProcessingContext struct {
	ObservationContexts []*ObservationContext `json:"observation_contexts"`
	BeliefContexts      []*BeliefContext      `json:"belief_contexts"`
	Ontology            *Ontology             `json:"ontology,omitempty"`
	Metrics             *BeliefSystemMetrics  `json:"metrics,omitempty"`
}

type EpistemicEmotion int32

const (
	EmotionInvalid EpistemicEmotion = iota
	Confirmation
	Surprise
	Curiosity
	Confusion
)

type BeliefSystemMetrics struct {
	ClarificationScore      float64 `json:"clarification_score"`
	TotalBeliefs            int32   `json:"total_beliefs"`
	TotalFalsifiableBeliefs int32   `json:"total_falsifiable_beliefs"`
	TotalCausalBeliefs      int32   `json:"total_causal_beliefs"`
	TotalBeliefStatements   int32   `json:"total_belief_statements"`
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

// Observation represents an observed state after an action
type Observation struct {
	DialecticInteractionID string             `json:"dialectic_interaction_id"`
	Type                   ObservationType    `json:"type"`
	Resource               *Resource          `json:"resource"`
	StateDistribution      map[string]float32 `json:"state_distribution"`
	Timestamp              int64              `json:"timestamp"`
}

func (o Observation) ToProto() *pbmodels.Observation {
	obs := &pbmodels.Observation{
		DialecticInteractionId: o.DialecticInteractionID,
		Type:                   pbmodels.ObservationType(o.Type),
		StateDistribution:      o.StateDistribution,
		Timestamp:              o.Timestamp,
	}

	// Only convert Resource if it exists
	if o.Resource != nil {
		obs.Resource = o.Resource.ToProto()
	}

	return obs
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

func (d Discrepancy) ToProto() *pbmodels.Discrepancy {
	return &pbmodels.Discrepancy{
		DialecticInteractionId: d.DialecticInteractionID,
		KlDivergence:           float64(d.KlDivergence),
		PointwiseKlTerms:       d.PointwiseKlTerms,
		Timestamp:              d.Timestamp,
	}
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
