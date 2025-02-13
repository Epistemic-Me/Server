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

	// Add context to the calculator
	AddContext(context *pb.ObservationContext)
}

// entropyCalculator implements LearningProgress
type entropyCalculator struct {
	// Map of context ID to current probability distribution
	stateDistributions map[string]map[string]float64
	// Store of observation contexts
	contexts []*pb.ObservationContext
}

// NewEntropyCalculator creates a new entropy calculator
func NewEntropyCalculator() LearningProgress {
	return &entropyCalculator{
		stateDistributions: make(map[string]map[string]float64),
		contexts:           make([]*pb.ObservationContext, 0),
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

		// Find the context
		var context *pb.ObservationContext
		for _, ctx := range e.contexts {
			if ctx.Id == contextID {
				context = ctx
				break
			}
		}
		if context == nil {
			return fmt.Errorf("context %s not found", contextID)
		}

		// Update probabilities based on how close the measurement is to each state's properties
		total := 0.0
		newDist := make(map[string]float64)

		// Calculate likelihoods for each state
		for stateID, oldProb := range dist {
			var state *pb.State
			for _, s := range context.PossibleStates {
				if s.Id == stateID {
					state = s
					break
				}
			}
			if state == nil {
				continue
			}

			// Calculate likelihood based on measurement type and state properties
			var likelihood float64
			switch observation.Metadata["measurement_type"] {
			case "grip_strength", "reaction_time":
				// For physical measurements, use range-based likelihood
				if minVal, ok := state.Properties["min_value"]; ok {
					if maxVal, ok := state.Properties["max_value"]; ok {
						if measurement >= float64(minVal) && measurement <= float64(maxVal) {
							likelihood = 1.0
						} else {
							distanceToRange := math.Min(
								math.Abs(measurement-float64(minVal)),
								math.Abs(measurement-float64(maxVal)),
							)
							rangeSize := float64(maxVal - minVal)
							if rangeSize > 0 {
								likelihood = 1.0 / (1.0 + distanceToRange/rangeSize)
							} else {
								likelihood = 1.0 / (1.0 + distanceToRange)
							}
						}
					}
				}
			default:
				// For other measurements, use age range-based likelihood
				if ageMin, ok := state.Properties["age_min"]; ok {
					if ageMax, ok := state.Properties["age_max"]; ok {
						if measurement >= float64(ageMin) && measurement <= float64(ageMax) {
							likelihood = 1.0
						} else {
							distanceToRange := math.Min(
								math.Abs(measurement-float64(ageMin)),
								math.Abs(measurement-float64(ageMax)),
							)
							ageRange := float64(ageMax - ageMin)
							if ageRange > 0 {
								likelihood = 1.0 / (1.0 + distanceToRange/ageRange)
							} else {
								likelihood = 1.0 / (1.0 + distanceToRange)
							}
						}
					}
				}
			}

			// If no likelihood was calculated, use a small default value
			if likelihood == 0 {
				likelihood = 0.1
			}

			newDist[stateID] = oldProb * likelihood
			total += newDist[stateID]
		}

		// Normalize probabilities
		if total > 0 {
			for stateID := range newDist {
				newDist[stateID] /= total
			}
			e.stateDistributions[contextID] = newDist
		}
	}

	return nil
}

// AddContext adds a context to the calculator
func (e *entropyCalculator) AddContext(context *pb.ObservationContext) {
	e.contexts = append(e.contexts, context)

	// Initialize state distribution with uniform probabilities
	dist := make(map[string]float64)
	numStates := float64(len(context.PossibleStates))
	for _, state := range context.PossibleStates {
		dist[state.Id] = 1.0 / numStates
	}
	e.stateDistributions[context.Id] = dist
}
