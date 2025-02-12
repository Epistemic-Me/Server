package integration

import (
	"testing"

	pb "epistemic-me-core/pb/models"
	"epistemic-me-core/svc/models"

	"github.com/stretchr/testify/assert"
)

func TestEntropyCalculation(t *testing.T) {
	// Create a test observation context
	ctx := &pb.ObservationContext{
		Id:   "test_context",
		Name: "Test Context",
		PossibleStates: []*pb.State{
			{Id: "state1", Name: "State 1"},
			{Id: "state2", Name: "State 2"},
			{Id: "state3", Name: "State 3"},
		},
	}

	// Create entropy calculator
	calc := models.NewEntropyCalculator()

	// Test initial entropy (should be maximum for uniform distribution)
	initialEntropy := calc.CalculateContextEntropy(ctx)
	assert.InDelta(t, 1.58, initialEntropy, 0.01) // log2(3) ≈ 1.58 for 3 states

	// Create a test observation
	observation := &pb.Resource{
		Type: pb.ResourceType_MEASUREMENT_DATA,
		Metadata: map[string]string{
			"measurement_value": "10.0",
			"state_state1":      "8.0",
			"state_state2":      "10.0",
			"state_state3":      "12.0",
		},
	}

	// Update entropy with observation
	err := calc.UpdateEntropy(ctx.Id, observation)
	assert.NoError(t, err)

	// Test updated entropy (should be lower after observation)
	updatedEntropy := calc.CalculateContextEntropy(ctx)
	assert.Less(t, updatedEntropy, initialEntropy)
}

func TestLearningProgress(t *testing.T) {
	// Create test observation contexts
	ctx1 := &pb.ObservationContext{
		Id:   "ctx1",
		Name: "Context 1",
		PossibleStates: []*pb.State{
			{Id: "state1", Name: "State 1"},
			{Id: "state2", Name: "State 2"},
		},
	}

	ctx2 := &pb.ObservationContext{
		Id:   "ctx2",
		Name: "Context 2",
		PossibleStates: []*pb.State{
			{Id: "state3", Name: "State 3"},
			{Id: "state4", Name: "State 4"},
		},
	}

	// Create entropy calculator
	calc := models.NewEntropyCalculator()

	// Test initial progress
	initialProgress := calc.CalculateProgress([]*pb.ObservationContext{ctx1, ctx2})
	assert.Equal(t, 0.0, initialProgress)

	// Create observations
	obs1 := &pb.Resource{
		Type: pb.ResourceType_MEASUREMENT_DATA,
		Metadata: map[string]string{
			"measurement_value": "5.0",
			"state_state1":      "5.0",
			"state_state2":      "10.0",
		},
	}

	obs2 := &pb.Resource{
		Type: pb.ResourceType_MEASUREMENT_DATA,
		Metadata: map[string]string{
			"measurement_value": "15.0",
			"state_state3":      "15.0",
			"state_state4":      "20.0",
		},
	}

	// Update entropy with observations
	err := calc.UpdateEntropy(ctx1.Id, obs1)
	assert.NoError(t, err)
	err = calc.UpdateEntropy(ctx2.Id, obs2)
	assert.NoError(t, err)

	// Test final progress (should be > 0)
	finalProgress := calc.CalculateProgress([]*pb.ObservationContext{ctx1, ctx2})
	assert.Greater(t, finalProgress, 0.0)
	assert.LessOrEqual(t, finalProgress, 1.0)
}
