package models

import (
	pbmodels "epistemic-me-core/pb/models"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

type PredictiveProcessingContext struct {
	ObservationContexts []*ObservationContext `json:"observation_contexts"`
	BeliefContexts      []*BeliefContext      `json:"belief_contexts"`
}

func (ppc PredictiveProcessingContext) ToProto() *pbmodels.EpistemicContext_PredictiveProcessingContext {
	return &pbmodels.EpistemicContext_PredictiveProcessingContext{
		PredictiveProcessingContext: &pbmodels.PredictiveProcessingContext{
			ObservationContexts: observationContextsToProto(ppc.ObservationContexts),
			BeliefContexts:      beliefContextsToProto(ppc.BeliefContexts),
		},
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

type Prediction struct {
	Action               *Action      `json:"action,omitempty"`
	Observation          *Observation `json:"observation,omitempty"`
	Discrepancy          *Discrepancy `json:"discrepancy,omitempty"`
	PredictedObservation *Observation `json:"predicted_observation,omitempty"`
}

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

func ontologyToProto(o *Ontology) *pbmodels.Ontology {
	if o == nil {
		return nil
	}
	protoContexts := make([]*pbmodels.BeliefContext, len(o.Contexts))
	for i, ctx := range o.Contexts {
		protoContexts[i] = ctx.ToProto()
	}
	return &pbmodels.Ontology{
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

func beliefContextsToProto(contexts []*BeliefContext) []*pbmodels.BeliefContext {
	result := make([]*pbmodels.BeliefContext, len(contexts))
	for i, c := range contexts {
		result[i] = c.ToProto()
	}
	return result
}

// ExtrapolateObservationContexts parses the Experiential Narrative section of a markdown philosophy description.
// It extracts [[C: ...]] as ObservationContext names and [[S: ...]] as possible states, using the existing ObservationContext model.
// Each context gets a generated UUID, and states are added to the most recent context at the current depth. ParentID is set based on indentation (2 spaces = one level).
// Only the Experiential Narrative section is parsed.
func ExtrapolateObservationContexts(description string) []*ObservationContext {
	// Find the Experiential Narrative section
	expSection := extractExperientialNarrativeSection(description)
	if expSection == "" {
		return nil
	}

	contextRe := regexp.MustCompile(`\[\[C:([^\]]+)\]\]`)
	stateRe := regexp.MustCompile(`\[\[S:([^\]]+)\]\]`)

	var contexts []*ObservationContext
	// Stack of most recent context at each depth
	contextStack := make(map[int]*ObservationContext)

	lines := strings.Split(expSection, "\n")
	for _, line := range lines {
		// Count leading spaces for depth (2 spaces = one level)
		depth := 0
		for i := 0; i < len(line); i++ {
			if line[i] == ' ' {
				depth++
			} else {
				break
			}
		}
		depth = depth / 2

		// Find all contexts in the line
		contextMatches := contextRe.FindAllStringSubmatch(line, -1)
		for _, match := range contextMatches {
			ctxName := strings.TrimSpace(match[1])
			ctx := &ObservationContext{
				ID:             uuid.New().String(),
				Name:           ctxName,
				ParentID:       "",
				PossibleStates: []string{},
			}
			// Set ParentID if there is a context at depth-1
			if parent, ok := contextStack[depth-1]; ok && depth > 0 {
				ctx.ParentID = parent.ID
			}
			contexts = append(contexts, ctx)
			contextStack[depth] = ctx
			// Remove deeper contexts from stack
			for d := depth + 1; ; d++ {
				if _, ok := contextStack[d]; ok {
					delete(contextStack, d)
				} else {
					break
				}
			}
		}

		// Find all states in the line
		stateMatches := stateRe.FindAllStringSubmatch(line, -1)
		for _, match := range stateMatches {
			stateName := strings.TrimSpace(match[1])
			if ctx, ok := contextStack[depth]; ok {
				ctx.PossibleStates = append(ctx.PossibleStates, stateName)
			}
		}
	}

	return contexts
}

// extractExperientialNarrativeSection extracts the Experiential Narrative section from the markdown.
func extractExperientialNarrativeSection(md string) string {
	start := strings.Index(md, "## Experiential Narrative")
	if start == -1 {
		return ""
	}
	// Find the next section or end of string
	end := strings.Index(md[start+1:], "## ")
	if end == -1 {
		return md[start:]
	}
	return md[start : start+1+end]
}
