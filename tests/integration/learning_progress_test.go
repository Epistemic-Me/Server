package integration

import (
	"testing"

	pb "epistemic-me-core/pb/models"
	"epistemic-me-core/svc/models"

	"github.com/stretchr/testify/assert"
)

func TestBeliefEntropyCalculation(t *testing.T) {
	// Create a test observation context with range-based states
	ctx := &pb.ObservationContext{
		Id:   "test_context",
		Name: "Test Context",
		PossibleStates: []*pb.State{
			{
				Id:   "state1",
				Name: "State 1",
				Properties: map[string]float32{
					"min_value": 7.0,
					"max_value": 9.0,
				},
			},
			{
				Id:   "state2",
				Name: "State 2",
				Properties: map[string]float32{
					"min_value": 9.0,
					"max_value": 11.0,
				},
			},
			{
				Id:   "state3",
				Name: "State 3",
				Properties: map[string]float32{
					"min_value": 11.0,
					"max_value": 13.0,
				},
			},
		},
	}

	// Create progress tracker with range interpreter
	tracker := models.NewBeliefProgressTracker()
	interpreter := models.NewRangeStateInterpreter("gaussian")
	tracker.AddContext("test_context", ctx, interpreter)

	// Get the belief context from the tracker
	beliefCtx := tracker.GetBeliefContext("test_context")
	initialEntropy := tracker.CalculateBeliefEntropy(beliefCtx)
	assert.InDelta(t, 1.58, initialEntropy, 0.01) // log2(3) ≈ 1.58 for 3 states

	// Create a test observation that matches state2's range
	observation := &pb.Resource{
		Type: pb.ResourceType_MEASUREMENT_DATA,
		Metadata: map[string]string{
			"measurement_value": "10.0",
			"context_id":        "test_context",
		},
	}

	// Update beliefs with observation
	err := tracker.UpdateBeliefs(observation)
	assert.NoError(t, err)

	// Get the updated belief context and test entropy
	beliefCtx = tracker.GetBeliefContext("test_context")
	updatedEntropy := tracker.CalculateBeliefEntropy(beliefCtx)
	assert.Less(t, updatedEntropy, initialEntropy)
}

func TestBeliefLearningProgress(t *testing.T) {
	// Create test observation contexts
	ctx1 := &pb.ObservationContext{
		Id:   "ctx1",
		Name: "Context 1",
		PossibleStates: []*pb.State{
			{
				Id:   "state1",
				Name: "State 1",
				Properties: map[string]float32{
					"min_value": 4.0,
					"max_value": 6.0,
				},
			},
			{
				Id:   "state2",
				Name: "State 2",
				Properties: map[string]float32{
					"min_value": 9.0,
					"max_value": 11.0,
				},
			},
		},
	}

	ctx2 := &pb.ObservationContext{
		Id:   "ctx2",
		Name: "Context 2",
		PossibleStates: []*pb.State{
			{
				Id:   "state3",
				Name: "State 3",
				Properties: map[string]float32{
					"min_value": 14.0,
					"max_value": 16.0,
				},
			},
			{
				Id:   "state4",
				Name: "State 4",
				Properties: map[string]float32{
					"min_value": 19.0,
					"max_value": 21.0,
				},
			},
		},
	}

	// Create progress tracker with range interpreter
	tracker := models.NewBeliefProgressTracker()
	interpreter := models.NewRangeStateInterpreter("gaussian")
	tracker.AddContext("ctx1", ctx1, interpreter)
	tracker.AddContext("ctx2", ctx2, interpreter)

	// Get belief contexts from tracker
	beliefCtx1 := tracker.GetBeliefContext("ctx1")
	beliefCtx2 := tracker.GetBeliefContext("ctx2")

	// Test initial progress
	initialProgress := tracker.CalculateProgress([]*pb.BeliefContext{beliefCtx1, beliefCtx2})
	assert.Equal(t, 0.0, initialProgress)

	// Create observations
	obs1 := &pb.Resource{
		Type: pb.ResourceType_MEASUREMENT_DATA,
		Metadata: map[string]string{
			"measurement_value": "5.0",
			"context_id":        "ctx1",
		},
	}

	obs2 := &pb.Resource{
		Type: pb.ResourceType_MEASUREMENT_DATA,
		Metadata: map[string]string{
			"measurement_value": "15.0",
			"context_id":        "ctx2",
		},
	}

	// Update beliefs with observations
	err := tracker.UpdateBeliefs(obs1)
	assert.NoError(t, err)
	err = tracker.UpdateBeliefs(obs2)
	assert.NoError(t, err)

	// Get updated belief contexts and test progress
	beliefCtx1 = tracker.GetBeliefContext("ctx1")
	beliefCtx2 = tracker.GetBeliefContext("ctx2")
	finalProgress := tracker.CalculateProgress([]*pb.BeliefContext{beliefCtx1, beliefCtx2})
	assert.Greater(t, finalProgress, 0.0)
	assert.LessOrEqual(t, finalProgress, 1.0)
}

func TestSimpleEntropyCalculation(t *testing.T) {
	// Create a simple observation context with just two states
	ctx := &pb.ObservationContext{
		Id:   "grip_strength",
		Name: "Grip Strength Assessment",
		PossibleStates: []*pb.State{
			{
				Id:   "strong",
				Name: "Strong",
				Properties: map[string]float32{
					"min_value": 85.0,
					"max_value": 120.0,
				},
			},
			{
				Id:   "weak",
				Name: "Weak",
				Properties: map[string]float32{
					"min_value": 50.0,
					"max_value": 84.9,
				},
			},
		},
	}

	// Create belief progress tracker and add context
	tracker := models.NewBeliefProgressTracker()
	interpreter := models.NewRangeStateInterpreter("grip_strength")
	tracker.AddContext(ctx.Id, ctx, interpreter)

	// Get initial belief context and verify uniform distribution
	beliefCtx := tracker.GetBeliefContext(ctx.Id)
	initialEntropy := tracker.CalculateBeliefEntropy(beliefCtx)

	// For 2 states with uniform probability (1/2 each), entropy should be log2(2) = 1
	assert.InDelta(t, 1.0, initialEntropy, 0.01, "Initial entropy should be log2(2) = 1 for uniform distribution")

	// Print initial probabilities
	t.Logf("Initial probabilities: strong=%v, weak=%v",
		beliefCtx.ConditionalProbabilities["strong"],
		beliefCtx.ConditionalProbabilities["weak"])

	// Test measurement indicating strong grip strength (90kg)
	measurement := &pb.Resource{
		Type: pb.ResourceType_MEASUREMENT_DATA,
		Metadata: map[string]string{
			"measurement_value": "90.0",
			"measurement_type":  "grip_strength",
			"context_id":        ctx.Id,
			"unit":              "kg",
		},
	}

	// Update beliefs with measurement
	err := tracker.UpdateBeliefs(measurement)
	assert.NoError(t, err)

	// Get updated belief context and check entropy
	beliefCtx = tracker.GetBeliefContext(ctx.Id)
	updatedEntropy := tracker.CalculateBeliefEntropy(beliefCtx)

	// Print updated probabilities
	t.Logf("Updated probabilities after 90kg measurement: strong=%v, weak=%v",
		beliefCtx.ConditionalProbabilities["strong"],
		beliefCtx.ConditionalProbabilities["weak"])
	t.Logf("Updated entropy: %v", updatedEntropy)

	// Entropy should be close to 0 since we should be very confident in the "strong" state
	assert.Less(t, updatedEntropy, 0.1, "Entropy should be close to 0 after clear strong measurement")
	assert.Greater(t, float64(beliefCtx.ConditionalProbabilities["strong"]), 0.9,
		"Strong state should have very high probability after 90kg measurement")
}
