package integration

import (
	"testing"

	pb "epistemic-me-core/pb/models"
	"epistemic-me-core/svc/models"

	"github.com/stretchr/testify/assert"
)

func TestBiologicalAgeContext(t *testing.T) {
	// Create an observation context for grip strength
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
				Id:   "moderate",
				Name: "Moderate",
				Properties: map[string]float32{
					"min_value": 80.0,
					"max_value": 85.0,
				},
			},
			{
				Id:   "weak",
				Name: "Weak",
				Properties: map[string]float32{
					"min_value": 75.0,
					"max_value": 80.0,
				},
			},
		},
	}

	// Create belief progress tracker and add context
	tracker := models.NewBeliefProgressTracker()
	interpreter := models.NewRangeStateInterpreter("grip_strength")
	tracker.AddContext(ctx.Id, ctx, interpreter)

	// Get initial belief context from tracker
	beliefCtx := tracker.GetBeliefContext(ctx.Id, "test_user")

	// Test initial entropy (should be maximum for uniform distribution)
	initialEntropy := tracker.CalculateBeliefEntropy(beliefCtx)
	assert.InDelta(t, 1.58, initialEntropy, 0.01) // log2(3) ≈ 1.58 for 3 states

	// Add a measurement indicating strong grip strength
	measurement := &pb.Resource{
		Type: pb.ResourceType_MEASUREMENT_DATA,
		Metadata: map[string]string{
			"measurement_value": "90.0",
			"measurement_type":  "grip_strength",
			"context_id":        ctx.Id,
			"unit":              "kg",
			"user_id":           "test_user",
		},
	}

	// Update beliefs with measurement
	err := tracker.UpdateBeliefs(measurement)
	assert.NoError(t, err)

	// Get updated belief context from tracker and check entropy
	beliefCtx = tracker.GetBeliefContext(ctx.Id, "test_user")
	updatedEntropy := tracker.CalculateBeliefEntropy(beliefCtx)
	assert.Less(t, updatedEntropy, initialEntropy)
}

func TestMultipleIndicators(t *testing.T) {
	// Create contexts for different biological age indicators
	gripCtx := &pb.ObservationContext{
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
				Id:   "moderate",
				Name: "Moderate",
				Properties: map[string]float32{
					"min_value": 75.0,
					"max_value": 85.0,
				},
			},
		},
	}

	reactionCtx := &pb.ObservationContext{
		Id:   "reaction_time",
		Name: "Reaction Time Assessment",
		PossibleStates: []*pb.State{
			{
				Id:   "quick",
				Name: "Quick Response",
				Properties: map[string]float32{
					"min_value": 0.15,
					"max_value": 0.25,
				},
			},
			{
				Id:   "moderate",
				Name: "Moderate Response",
				Properties: map[string]float32{
					"min_value": 0.25,
					"max_value": 0.35,
				},
			},
		},
	}

	// Create belief progress tracker and add contexts
	tracker := models.NewBeliefProgressTracker()
	gripInterpreter := models.NewRangeStateInterpreter("grip_strength")
	reactionInterpreter := models.NewRangeStateInterpreter("reaction_time")
	tracker.AddContext(gripCtx.Id, gripCtx, gripInterpreter)
	tracker.AddContext(reactionCtx.Id, reactionCtx, reactionInterpreter)

	// Get belief contexts from tracker
	gripBeliefs := tracker.GetBeliefContext(gripCtx.Id, "test_user")
	reactionBeliefs := tracker.GetBeliefContext(reactionCtx.Id, "test_user")

	// Test initial progress across both contexts
	beliefContexts := []*pb.BeliefContext{gripBeliefs, reactionBeliefs}
	initialProgress := tracker.CalculateProgress(beliefContexts)
	assert.Equal(t, 0.0, initialProgress)

	// Add measurements
	gripMeasurement := &pb.Resource{
		Type: pb.ResourceType_MEASUREMENT_DATA,
		Metadata: map[string]string{
			"measurement_value": "90.0",
			"measurement_type":  "grip_strength",
			"context_id":        gripCtx.Id,
			"unit":              "kg",
			"user_id":           "test_user",
		},
	}

	reactionMeasurement := &pb.Resource{
		Type: pb.ResourceType_MEASUREMENT_DATA,
		Metadata: map[string]string{
			"measurement_value": "0.20",
			"measurement_type":  "reaction_time",
			"context_id":        reactionCtx.Id,
			"unit":              "seconds",
			"user_id":           "test_user",
		},
	}

	// Update beliefs with measurements
	err := tracker.UpdateBeliefs(gripMeasurement)
	assert.NoError(t, err)
	err = tracker.UpdateBeliefs(reactionMeasurement)
	assert.NoError(t, err)

	// Get updated belief contexts and test progress
	gripBeliefs = tracker.GetBeliefContext(gripCtx.Id, "test_user")
	reactionBeliefs = tracker.GetBeliefContext(reactionCtx.Id, "test_user")
	beliefContexts = []*pb.BeliefContext{gripBeliefs, reactionBeliefs}
	finalProgress := tracker.CalculateProgress(beliefContexts)
	assert.Greater(t, finalProgress, 0.0)
	assert.LessOrEqual(t, finalProgress, 1.0)
}

func TestStateTransitions(t *testing.T) {
	// Create an observation context for grip strength measurements
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
				Id:   "moderate",
				Name: "Moderate",
				Properties: map[string]float32{
					"min_value": 70.0,
					"max_value": 84.9,
				},
			},
			{
				Id:   "weak",
				Name: "Weak",
				Properties: map[string]float32{
					"min_value": 55.0,
					"max_value": 69.9,
				},
			},
		},
	}

	// Create belief progress tracker and add context
	tracker := models.NewBeliefProgressTracker()
	interpreter := models.NewRangeStateInterpreter("grip_strength")
	tracker.AddContext(ctx.Id, ctx, interpreter)

	// Test three different users
	users := []struct {
		id            string
		measurement   string
		expectedState string
	}{
		{"user1", "90.0", "strong"},   // Strong grip strength
		{"user2", "75.0", "moderate"}, // Moderate grip strength
		{"user3", "60.0", "weak"},     // Weak grip strength
	}

	// Test each user's measurements
	for _, user := range users {
		// Get initial belief context for this user
		beliefCtx := tracker.GetBeliefContext(ctx.Id, user.id)
		initialEntropy := tracker.CalculateBeliefEntropy(beliefCtx)

		// For 3 states with uniform probability (1/3 each), entropy should be log2(3) ≈ 1.58
		assert.InDelta(t, 1.58, initialEntropy, 0.01,
			"Initial entropy should be maximum (uniform distribution) for user %s", user.id)

		// Verify initial uniform probabilities
		for _, state := range ctx.PossibleStates {
			assert.InDelta(t, 0.333, beliefCtx.ConditionalProbabilities[state.Id], 0.001,
				"Initial probability should be uniform (1/3) for user %s", user.id)
		}

		// Create measurement for this user
		measurement := &pb.Resource{
			Type: pb.ResourceType_MEASUREMENT_DATA,
			Metadata: map[string]string{
				"measurement_value": user.measurement,
				"measurement_type":  "grip_strength",
				"context_id":        ctx.Id,
				"unit":              "kg",
				"user_id":           user.id,
			},
		}

		// Update beliefs with measurement
		err := tracker.UpdateBeliefs(measurement)
		assert.NoError(t, err)

		// Get updated belief context and check entropy
		beliefCtx = tracker.GetBeliefContext(ctx.Id, user.id)
		updatedEntropy := tracker.CalculateBeliefEntropy(beliefCtx)

		// Entropy should decrease significantly
		assert.Less(t, updatedEntropy, initialEntropy/2,
			"Measurement should significantly reduce entropy for user %s", user.id)

		// Expected state should have highest probability
		assert.Greater(t, beliefCtx.ConditionalProbabilities[user.expectedState], 0.9,
			"%s state should have very high probability for user %s", user.expectedState, user.id)

		// Log the entropy change for this user
		t.Logf("User %s entropy: initial=%.4f -> after %s measurement=%.4f",
			user.id, initialEntropy, user.expectedState, updatedEntropy)
	}

	// Verify that each user's belief context remains distinct
	for _, user := range users {
		beliefCtx := tracker.GetBeliefContext(ctx.Id, user.id)
		assert.Greater(t, beliefCtx.ConditionalProbabilities[user.expectedState], 0.9,
			"User %s should still have high probability for %s state", user.id, user.expectedState)
	}
}
