package models

import (
	"fmt"
	"math"
	"strconv"

	pb "epistemic-me-core/pb/models"
)

// LearningProgress tracks belief updates and entropy reduction
type LearningProgress interface {
	// Calculate entropy for a belief context
	CalculateBeliefEntropy(context *pb.BeliefContext) float64

	// Update beliefs based on new observation
	UpdateBeliefs(observation *pb.Resource) error

	// Calculate overall learning progress
	CalculateProgress(beliefContexts []*pb.BeliefContext) float64

	// Add observation context with its interpreter
	AddContext(contextID string, context *pb.ObservationContext, interpreter StateInterpreter)

	// Get belief context by ID and user
	GetBeliefContext(contextID string, userID string) *pb.BeliefContext
}

// beliefProgressTracker implements LearningProgress
type beliefProgressTracker struct {
	// Map of user_id:context_id to BeliefContext
	beliefContexts map[string]*pb.BeliefContext
	// Map of observation context ID to its interpreter
	stateInterpreters map[string]StateInterpreter
	// Store of observation contexts
	observationContexts map[string]*pb.ObservationContext
}

// Helper function to generate belief context key
func getBeliefContextKey(userID, contextID string) string {
	if userID == "" {
		return contextID // Fallback for backward compatibility
	}
	return userID + ":" + contextID
}

// NewBeliefProgressTracker creates a new learning progress tracker
func NewBeliefProgressTracker() LearningProgress {
	return &beliefProgressTracker{
		beliefContexts:      make(map[string]*pb.BeliefContext),
		stateInterpreters:   make(map[string]StateInterpreter),
		observationContexts: make(map[string]*pb.ObservationContext),
	}
}

// AddContext adds an observation context with its interpreter
func (b *beliefProgressTracker) AddContext(contextID string, context *pb.ObservationContext, interpreter StateInterpreter) {
	b.observationContexts[contextID] = context
	b.stateInterpreters[contextID] = interpreter
}

// InitializeUserContext creates a new belief context for a specific user
func (b *beliefProgressTracker) InitializeUserContext(userID string, contextID string) *pb.BeliefContext {
	context := b.observationContexts[contextID]
	if context == nil || len(context.PossibleStates) == 0 {
		return nil
	}

	// Initialize uniform belief context
	numStates := len(context.PossibleStates)
	uniformProb := 1.0 / float64(numStates)
	probs := make(map[string]float32)
	for _, state := range context.PossibleStates {
		probs[state.Id] = float32(uniformProb)
	}

	beliefCtx := &pb.BeliefContext{
		ObservationContextId:     contextID,
		ConditionalProbabilities: probs,
	}

	// Store with composite key
	key := getBeliefContextKey(userID, contextID)
	b.beliefContexts[key] = beliefCtx
	return beliefCtx
}

// GetBeliefContext returns the belief context for a given context ID and user
func (b *beliefProgressTracker) GetBeliefContext(contextID string, userID string) *pb.BeliefContext {
	key := getBeliefContextKey(userID, contextID)
	if beliefCtx, exists := b.beliefContexts[key]; exists {
		return beliefCtx
	}

	// If not found and userID provided, initialize new context
	if userID != "" {
		return b.InitializeUserContext(userID, contextID)
	}

	return nil
}

// CalculateBeliefEntropy computes Shannon entropy for a belief context
func (b *beliefProgressTracker) CalculateBeliefEntropy(context *pb.BeliefContext) float64 {
	if context == nil || len(context.ConditionalProbabilities) == 0 {
		return 0
	}

	var entropy float64
	for _, prob := range context.ConditionalProbabilities {
		if prob > 0 {
			entropy -= float64(prob) * math.Log2(float64(prob))
		}
	}
	return entropy
}

// UpdateBeliefs updates probability distribution based on new evidence
func (b *beliefProgressTracker) UpdateBeliefs(observation *pb.Resource) error {
	if observation == nil {
		return fmt.Errorf("observation cannot be nil")
	}

	// Get measurement value and context
	measurementValue, err := getMeasurementValue(observation)
	if err != nil {
		return err
	}

	contextID := observation.Metadata["context_id"]
	userID := observation.Metadata["user_id"] // Get user ID from metadata

	context, ok := b.observationContexts[contextID]
	if !ok {
		return fmt.Errorf("context %s not found", contextID)
	}

	interpreter, ok := b.stateInterpreters[contextID]
	if !ok {
		return fmt.Errorf("no interpreter found for context %s", contextID)
	}

	// Get or initialize belief context for this user
	beliefContext := b.GetBeliefContext(contextID, userID)
	if beliefContext == nil {
		return fmt.Errorf("could not get or initialize belief context for user %s and context %s", userID, contextID)
	}

	// Calculate likelihoods and update probabilities
	newProbs := make(map[string]float32)
	var totalLikelihood float64

	// First pass: calculate unnormalized posterior probabilities
	for _, state := range context.PossibleStates {
		likelihood := interpreter.CalculateLikelihood(measurementValue, state)
		prior := float64(beliefContext.ConditionalProbabilities[state.Id])
		posterior := likelihood * prior
		newProbs[state.Id] = float32(posterior)
		totalLikelihood += posterior
	}

	// Normalize probabilities
	if totalLikelihood > 0 {
		for stateID := range newProbs {
			newProbs[stateID] = float32(float64(newProbs[stateID]) / totalLikelihood)
		}
	}

	beliefContext.ConditionalProbabilities = newProbs
	return nil
}

// CalculateProgress computes overall learning progress as entropy reduction
func (b *beliefProgressTracker) CalculateProgress(beliefContexts []*pb.BeliefContext) float64 {
	if len(beliefContexts) == 0 {
		return 0
	}

	var totalProgress float64
	var maxPossibleEntropy float64
	var numContexts float64

	for _, context := range beliefContexts {
		obsContext, ok := b.observationContexts[context.ObservationContextId]
		if !ok {
			continue
		}

		// Calculate maximum possible entropy for uniform distribution
		numStates := float64(len(obsContext.PossibleStates))
		if numStates > 0 {
			maxEntropy := -math.Log2(1.0 / numStates)
			maxPossibleEntropy += maxEntropy
			numContexts++

			// Calculate current entropy
			currentEntropy := b.CalculateBeliefEntropy(context)

			// Check if distribution is non-uniform with a more lenient threshold
			isUniform := true
			expectedUniformProb := 1.0 / numStates
			for _, prob := range context.ConditionalProbabilities {
				if math.Abs(float64(prob)-expectedUniformProb) > 0.001 { // More sensitive threshold
					isUniform = false
					break
				}
			}

			// Calculate progress based on entropy reduction
			entropyReduction := maxEntropy - currentEntropy
			if entropyReduction > 0 || !isUniform {
				// Scale progress based on both entropy reduction and non-uniformity
				progress := entropyReduction / maxEntropy
				if !isUniform {
					// Add bonus for non-uniform distributions
					progress += 0.1
				}
				totalProgress += math.Min(1.0, progress) * maxEntropy
			}
		}
	}

	// Normalize progress to [0,1] range
	if maxPossibleEntropy > 0 && numContexts > 0 {
		return math.Min(1.0, totalProgress/maxPossibleEntropy)
	}
	return 0
}

// Helper function to get measurement value from observation
func getMeasurementValue(observation *pb.Resource) (float64, error) {
	valueStr, ok := observation.Metadata["measurement_value"]
	if !ok {
		return 0, fmt.Errorf("no measurement_value found in observation")
	}

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid measurement value: %v", err)
	}

	return value, nil
}
