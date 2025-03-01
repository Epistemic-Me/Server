package models

import (
	"fmt"
	"math"
	"sort"

	pb "epistemic-me-core/pb/models"
)

// AgeEstimate represents a biological age estimate with confidence
type AgeEstimate struct {
	MinAge     float64
	MaxAge     float64
	Confidence float64
	Sources    []string // Which measurements contributed to this estimate
}

// MeasurementProcessor handles processing of biological age measurements
type MeasurementProcessor interface {
	// Process a batch of measurements and update observation contexts
	ProcessMeasurements(measurements []*pb.Resource) error

	// Get combined age estimate from all contexts
	GetAgeEstimate() (*AgeEstimate, error)

	// Get measurement suggestions based on uncertainty
	GetSuggestions() []string

	// Add a new measurement context
	AddContext(measurementType string, context *pb.ObservationContext)
}

// measurementProcessor implements MeasurementProcessor
type measurementProcessor struct {
	entropyCalc  LearningProgress
	contexts     map[string]*pb.ObservationContext // Map of measurement type to context
	interpreters map[string]StateInterpreter       // Map of measurement type to state interpreter
	beliefs      map[string]*pb.BeliefContext      // Map of context ID to belief context
}

// NewMeasurementProcessor creates a new measurement processor
func NewMeasurementProcessor() MeasurementProcessor {
	return &measurementProcessor{
		entropyCalc:  NewBeliefProgressTracker(),
		contexts:     make(map[string]*pb.ObservationContext),
		interpreters: make(map[string]StateInterpreter),
		beliefs:      make(map[string]*pb.BeliefContext),
	}
}

// GetAgeEstimate combines estimates from all contexts
func (m *measurementProcessor) GetAgeEstimate() (*AgeEstimate, error) {
	if len(m.contexts) == 0 {
		return nil, fmt.Errorf("need measurements to make age estimate")
	}

	// Initialize age ranges
	var minAge float64 = 0
	var maxAge float64 = 150
	var totalConfidence float64 = 0
	sources := make([]string, 0)
	numNonUniformContexts := 0

	// For each context, check if we have measurements and if they contribute to age estimation
	for measurementType, context := range m.contexts {
		if len(context.PossibleStates) == 0 {
			continue
		}

		// Get the belief context for this measurement type
		beliefCtx := m.beliefs[measurementType]
		if beliefCtx == nil || len(beliefCtx.ConditionalProbabilities) == 0 {
			continue
		}

		// Check if the distribution is non-uniform by comparing to uniform probability
		numStates := len(context.PossibleStates)
		uniformProb := 1.0 / float64(numStates)
		maxDiff := 0.0

		for _, prob := range beliefCtx.ConditionalProbabilities {
			diff := math.Abs(float64(prob) - uniformProb)
			if diff > maxDiff {
				maxDiff = diff
			}
		}

		if maxDiff <= 0.001 { // Small threshold for non-uniformity
			continue
		}

		numNonUniformContexts++
		sources = append(sources, measurementType)

		// Find the most likely state
		var maxProb float64 = 0
		var mostLikelyState *pb.State
		for _, state := range context.PossibleStates {
			if prob := float64(beliefCtx.ConditionalProbabilities[state.Id]); prob > maxProb {
				maxProb = prob
				mostLikelyState = state
			}
		}

		if mostLikelyState == nil {
			continue
		}

		// Update age range based on the most likely state
		stateMinAge := float64(mostLikelyState.Properties["age_min"])
		stateMaxAge := float64(mostLikelyState.Properties["age_max"])

		if minAge == 0 || stateMinAge > minAge {
			minAge = stateMinAge
		}
		if maxAge == 150 || stateMaxAge < maxAge {
			maxAge = stateMaxAge
		}

		totalConfidence += maxProb
	}

	// If we have no contributing contexts, return error
	if numNonUniformContexts == 0 {
		return nil, fmt.Errorf("no contexts with non-uniform distributions found")
	}

	// Calculate average confidence
	confidence := totalConfidence / float64(numNonUniformContexts)

	return &AgeEstimate{
		MinAge:     minAge,
		MaxAge:     maxAge,
		Confidence: confidence,
		Sources:    sources,
	}, nil
}

// AddContext adds a new measurement context
func (m *measurementProcessor) AddContext(measurementType string, context *pb.ObservationContext) {
	// Store the context
	m.contexts[measurementType] = context

	// Create state interpreter
	if len(context.PossibleStates) > 0 {
		interpreter := NewRangeStateInterpreter(measurementType)
		m.interpreters[measurementType] = interpreter

		// Initialize uniform belief context
		numStates := len(context.PossibleStates)
		uniformProb := 1.0 / float64(numStates)
		probs := make(map[string]float32)
		for _, state := range context.PossibleStates {
			probs[state.Id] = float32(uniformProb)
		}
		beliefCtx := &pb.BeliefContext{
			ObservationContextId:     context.Id,
			ConditionalProbabilities: probs,
		}
		// Store belief context with both measurement type and context ID
		m.beliefs[measurementType] = beliefCtx
		m.beliefs[context.Id] = beliefCtx
	}
}

// GetSuggestions returns measurement suggestions based on uncertainty
func (m *measurementProcessor) GetSuggestions() []string {
	var suggestions []string

	// Check each context for measurements
	for measurementType, context := range m.contexts {
		beliefs := m.beliefs[context.Id]
		if beliefs == nil || len(beliefs.ConditionalProbabilities) == 0 {
			suggestions = append(suggestions, fmt.Sprintf("Need initial %s measurement", measurementType))
			continue
		}

		// Check if distribution is still uniform (no measurements)
		isUniform := true
		firstProb := float32(0)
		for _, prob := range beliefs.ConditionalProbabilities {
			if firstProb == 0 {
				firstProb = prob
			} else if prob != firstProb {
				isUniform = false
				break
			}
		}

		if isUniform {
			suggestions = append(suggestions, fmt.Sprintf("Need initial %s measurement", measurementType))
		} else {
			// Check uncertainty level
			entropy := m.entropyCalc.CalculateBeliefEntropy(beliefs)
			if entropy > 1.0 { // High uncertainty threshold
				suggestions = append(suggestions, fmt.Sprintf("Additional %s measurement recommended", measurementType))
			}
		}
	}

	return suggestions
}

// ProcessMeasurements processes a batch of measurements
func (m *measurementProcessor) ProcessMeasurements(measurements []*pb.Resource) error {
	// Group measurements by type
	measurementsByType := make(map[string][]*pb.Resource)
	for _, measurement := range measurements {
		if measurement.Type != pb.ResourceType_MEASUREMENT_DATA {
			continue
		}
		if measurementType, ok := measurement.Metadata["measurement_type"]; ok {
			measurementsByType[measurementType] = append(measurementsByType[measurementType], measurement)
		}
	}

	// Process each measurement type
	for measurementType, typeMeasurements := range measurementsByType {
		if _, ok := m.contexts[measurementType]; !ok {
			return fmt.Errorf("no context found for measurement type: %s", measurementType)
		}

		// Process measurements in chronological order if timestamps are available
		sort.Slice(typeMeasurements, func(i, j int) bool {
			iTime, iOk := typeMeasurements[i].Metadata["timestamp"]
			jTime, jOk := typeMeasurements[j].Metadata["timestamp"]
			if !iOk || !jOk {
				return false
			}
			return iTime < jTime
		})

		// Update beliefs for each measurement
		for _, measurement := range typeMeasurements {
			// Add measurement type to metadata for belief lookup
			measurement.Metadata["context_id"] = measurementType
			if err := m.entropyCalc.UpdateBeliefs(measurement); err != nil {
				return fmt.Errorf("error updating beliefs for %s: %v", measurementType, err)
			}
		}
	}

	return nil
}
