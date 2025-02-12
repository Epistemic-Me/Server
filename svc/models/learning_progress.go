package models

import (
	"fmt"
	"math"
	"strconv"

	pb "epistemic-me-core/pb/models"
)

// LearningProgress handles entropy calculation and progress tracking for learning objectives
type LearningProgress interface {
	// Calculate entropy for a single observation context
	CalculateContextEntropy(context *pb.ObservationContext) float64

	// Calculate overall learning progress across contexts
	CalculateProgress(contexts []*pb.ObservationContext) float64

	// Update entropy based on new observation
	UpdateEntropy(contextID string, observation *pb.Resource) error
}

// entropyCalculator implements LearningProgress
type entropyCalculator struct {
	// Map of context ID to current probability distribution
	stateDistributions map[string]map[string]float64
}

// NewEntropyCalculator creates a new entropy calculator
func NewEntropyCalculator() LearningProgress {
	return &entropyCalculator{
		stateDistributions: make(map[string]map[string]float64),
	}
}

// CalculateContextEntropy computes Shannon entropy for a single observation context
func (e *entropyCalculator) CalculateContextEntropy(context *pb.ObservationContext) float64 {
	if context == nil || len(context.PossibleStates) == 0 {
		return 0
	}

	// Get probability distribution for this context
	dist, ok := e.stateDistributions[context.Id]
	if !ok {
		// Initialize with uniform distribution if not exists
		dist = make(map[string]float64)
		numStates := float64(len(context.PossibleStates))
		for _, state := range context.PossibleStates {
			dist[state.Id] = 1.0 / numStates
		}
		e.stateDistributions[context.Id] = dist
	}

	// Calculate Shannon entropy: -sum(p * log2(p))
	entropy := 0.0
	for _, p := range dist {
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}
	return entropy
}

// CalculateProgress computes overall learning progress as entropy reduction
func (e *entropyCalculator) CalculateProgress(contexts []*pb.ObservationContext) float64 {
	if len(contexts) == 0 {
		return 0
	}

	totalEntropy := 0.0
	maxPossibleEntropy := 0.0

	for _, ctx := range contexts {
		currentEntropy := e.CalculateContextEntropy(ctx)
		totalEntropy += currentEntropy

		// Max entropy would be uniform distribution
		maxStates := float64(len(ctx.PossibleStates))
		if maxStates > 0 {
			maxPossibleEntropy += math.Log2(maxStates)
		}
	}

	if maxPossibleEntropy == 0 {
		return 0
	}

	// Progress is reduction in entropy relative to maximum possible
	return 1.0 - (totalEntropy / maxPossibleEntropy)
}

// UpdateEntropy updates probability distribution based on new evidence
func (e *entropyCalculator) UpdateEntropy(contextID string, observation *pb.Resource) error {
	// Get current distribution
	dist, ok := e.stateDistributions[contextID]
	if !ok {
		return fmt.Errorf("no distribution found for context %s", contextID)
	}

	// Update probabilities based on observation metadata
	if val, ok := observation.Metadata["measurement_value"]; ok {
		measurement, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return fmt.Errorf("invalid measurement value: %v", err)
		}

		// Update probabilities based on how close the measurement is to each state's range
		total := 0.0
		newDist := make(map[string]float64)

		for stateID, oldProb := range dist {
			// Calculate likelihood based on measurement and state properties
			// This should be customized based on the context's state properties
			var stateValue float64
			if v, ok := observation.Metadata["state_"+stateID]; ok {
				stateValue, _ = strconv.ParseFloat(v, 64)
			} else {
				stateValue = 0 // Default value if not found
			}

			likelihood := 1.0 / (1.0 + math.Abs(measurement-stateValue))
			newDist[stateID] = oldProb * likelihood
			total += newDist[stateID]
		}

		// Normalize
		if total > 0 {
			for stateID := range newDist {
				newDist[stateID] /= total
			}
			e.stateDistributions[contextID] = newDist
		}
	}

	return nil
}
