package svc

import (
	"fmt"

	"github.com/google/uuid"

	"epistemic-me-core/svc/models"
)

// PredictiveProcessingService handles operations related to the PredictiveProcessingContext
type PredictiveProcessingService struct {
	// No dependencies for now, but could add the KVStore or other services as needed
}

// NewPredictiveProcessingService creates a new instance of PredictiveProcessingService
func NewPredictiveProcessingService() *PredictiveProcessingService {
	return &PredictiveProcessingService{}
}

// EnsurePredictiveProcessingContext makes sure a PredictiveProcessingContext exists
// in the provided BeliefSystem and returns it
func (pps *PredictiveProcessingService) EnsurePredictiveProcessingContext(bs *models.BeliefSystem) *models.PredictiveProcessingContext {
	// Initialize EpistemicContexts if empty
	if len(bs.EpistemicContexts) == 0 {
		bs.EpistemicContexts = []*models.EpistemicContext{
			{
				PredictiveProcessingContext: &models.PredictiveProcessingContext{
					ObservationContexts: []*models.ObservationContext{},
					BeliefContexts:      []*models.BeliefContext{},
				},
			},
		}
		return bs.EpistemicContexts[0].PredictiveProcessingContext
	}

	// Check if PredictiveProcessingContext exists in the first context
	if bs.EpistemicContexts[0].PredictiveProcessingContext == nil {
		bs.EpistemicContexts[0].PredictiveProcessingContext = &models.PredictiveProcessingContext{
			ObservationContexts: []*models.ObservationContext{},
			BeliefContexts:      []*models.BeliefContext{},
		}
	}

	return bs.EpistemicContexts[0].PredictiveProcessingContext
}

// CreateObservationContext creates a new ObservationContext from a question-answer interaction
func (pps *PredictiveProcessingService) CreateObservationContext(
	ppc *models.PredictiveProcessingContext,
	question string,
	answer string,
) *models.ObservationContext {
	ocID := uuid.New().String()

	// Create an observation context for this interaction
	oc := &models.ObservationContext{
		ID:             ocID,
		Name:           fmt.Sprintf("Response to question"),
		ParentID:       "",
		PossibleStates: []string{"Positive", "Negative", "Neutral"},
	}

	// Add to the list of observation contexts
	ppc.ObservationContexts = append(ppc.ObservationContexts, oc)

	return oc
}

// CreateBeliefContext links a belief to an observation context
func (pps *PredictiveProcessingService) CreateBeliefContext(
	ppc *models.PredictiveProcessingContext,
	beliefID string,
	observationContextID string,
	confidenceScore float64,
) *models.BeliefContext {
	// Create a belief context that links the belief to the observation
	bc := &models.BeliefContext{
		BeliefID:             beliefID,
		ObservationContextID: observationContextID,
		ConfidenceRatings: []models.ConfidenceRating{
			{
				ConfidenceScore: confidenceScore,
				Default:         true,
			},
		},
		ConditionalProbs:        map[string]float32{},
		DialecticInteractionIDs: []string{},
		EpistemicEmotion:        models.Confirmation,
		EmotionIntensity:        0.5,
	}

	// Add to the list of belief contexts
	ppc.BeliefContexts = append(ppc.BeliefContexts, bc)

	return bc
}

// AddObservationFromInteraction creates a new observation from a question-answer interaction
// and links it to any beliefs that were extracted from the answer
func (pps *PredictiveProcessingService) AddObservationFromInteraction(
	bs *models.BeliefSystem,
	question string,
	answer string,
	extractedBeliefs []*models.Belief,
) {
	// Ensure we have a PredictiveProcessingContext
	ppc := pps.EnsurePredictiveProcessingContext(bs)

	// Create an observation context for this interaction
	oc := pps.CreateObservationContext(ppc, question, answer)

	// Create belief contexts for each extracted belief
	for _, belief := range extractedBeliefs {
		pps.CreateBeliefContext(ppc, belief.ID, oc.ID, 0.8) // Default confidence score
	}
}

// GetObservationsByBelief returns all observation contexts associated with a belief
func (pps *PredictiveProcessingService) GetObservationsByBelief(
	ppc *models.PredictiveProcessingContext,
	beliefID string,
) []*models.ObservationContext {
	var observations []*models.ObservationContext

	// Find all belief contexts for this belief
	for _, bc := range ppc.BeliefContexts {
		if bc.BeliefID == beliefID {
			// Find the observation context for this belief context
			for _, oc := range ppc.ObservationContexts {
				if oc.ID == bc.ObservationContextID {
					observations = append(observations, oc)
					break
				}
			}
		}
	}

	return observations
}

// GetBeliefsByObservation returns all beliefs associated with an observation context
func (pps *PredictiveProcessingService) GetBeliefsByObservation(
	bs *models.BeliefSystem,
	ppc *models.PredictiveProcessingContext,
	observationContextID string,
) []*models.Belief {
	var beliefs []*models.Belief

	// Find all belief contexts for this observation
	for _, bc := range ppc.BeliefContexts {
		if bc.ObservationContextID == observationContextID {
			// Find the belief for this belief context
			for _, belief := range bs.Beliefs {
				if belief.ID == bc.BeliefID {
					beliefs = append(beliefs, belief)
					break
				}
			}
		}
	}

	return beliefs
}

// CalculateBeliefMetrics computes metrics about the belief system based on the
// PredictiveProcessingContext data
func (pps *PredictiveProcessingService) CalculateBeliefMetrics(
	bs *models.BeliefSystem,
) *models.BeliefSystemMetrics {
	if len(bs.EpistemicContexts) == 0 || bs.EpistemicContexts[0].PredictiveProcessingContext == nil {
		return &models.BeliefSystemMetrics{
			TotalBeliefs:            int32(len(bs.Beliefs)),
			TotalFalsifiableBeliefs: 0,
			TotalCausalBeliefs:      0,
			TotalBeliefStatements:   int32(len(bs.Beliefs)),
		}
	}

	ppc := bs.EpistemicContexts[0].PredictiveProcessingContext

	// Count beliefs that have observation contexts (potentially falsifiable)
	beliefIDsWithObservations := make(map[string]struct{})
	for _, bc := range ppc.BeliefContexts {
		beliefIDsWithObservations[bc.BeliefID] = struct{}{}
	}

	// For now, we're using simple heuristics
	// In a real implementation, more sophisticated analysis would be used
	metrics := &models.BeliefSystemMetrics{
		TotalBeliefs:            int32(len(bs.Beliefs)),
		TotalFalsifiableBeliefs: int32(len(beliefIDsWithObservations)),
		TotalCausalBeliefs:      0, // Would require additional analysis
		TotalBeliefStatements:   int32(len(bs.Beliefs)),
		ClarificationScore:      0.0, // Would require additional analysis
	}

	// Calculate clarification score (simplified version)
	if len(bs.Beliefs) > 0 {
		metrics.ClarificationScore = float64(len(beliefIDsWithObservations)) / float64(len(bs.Beliefs))
	}

	return metrics
}
