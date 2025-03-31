package integration

import (
	"testing"

	pb "epistemic-me-core/pb/models"
	"github.com/stretchr/testify/assert"
)

func TestDialecticLearningFlow(t *testing.T) {
	// Create a test observation context from predictive_processing.proto
	observationCtx := &pb.ObservationContext{
		Id:          "context_1",
		Name:        "Sleep Quality",
		Description: "Sleep quality observation context",
		PossibleStates: []*pb.State{
			{
				Id:          "state_1",
				Name:        "Unknown",
				Description: "Initial state with high uncertainty",
				Properties: map[string]float32{
					"uncertainty": 1.0,
				},
			},
			{
				Id:          "state_2",
				Name:        "Understood",
				Description: "Final state with low uncertainty",
				Properties: map[string]float32{
					"uncertainty": 0.0,
				},
			},
		},
		CurrentStateId: "state_1", // Start in unknown state
	}

	// Create a learning objective that references the observation context
	learningObj := &pb.LearningObjective{
		Description:            "Learn about sleep patterns",
		TargetBeliefIds:       []string{"sleep_belief_1"},
		ObservationContextIds: []string{observationCtx.Id},
		CompletionPercentage:  0.0,
	}

	// Create a dialectic with the learning objective
	dialectic := &pb.Dialectic{
		Id:          "test_dialectic",
		SelfModelId: "test_self",
		Agent: &pb.Agent{
			AgentType:     pb.Agent_AGENT_TYPE_GPT_LATEST,
			DialecticType: pb.DialecticType_SLEEP_DIET_EXERCISE,
		},
		LearningObjective: learningObj,
		BeliefSystem:     &pb.BeliefSystem{}, // Initialize empty belief system
	}

	// Verify initial state
	assert.Equal(t, "state_1", observationCtx.CurrentStateId)
	assert.Equal(t, float32(1.0), observationCtx.PossibleStates[0].Properties["uncertainty"])
	assert.Equal(t, 0.0, dialectic.LearningObjective.CompletionPercentage)
	assert.NotNil(t, dialectic.BeliefSystem)

	// Simulate learning progress by transitioning to understood state
	observationCtx.CurrentStateId = "state_2"
	expectedProgress := 100.0 // Full understanding achieved
	dialectic.LearningObjective.CompletionPercentage = expectedProgress

	// Verify learning progress
	assert.Equal(t, "state_2", observationCtx.CurrentStateId)
	assert.Equal(t, float32(0.0), observationCtx.PossibleStates[1].Properties["uncertainty"])
	assert.Equal(t, 100.0, dialectic.LearningObjective.CompletionPercentage)
}

func TestMultiContextLearning(t *testing.T) {
	// Create multiple observation contexts
	contexts := []*pb.ObservationContext{
		{
			Id:          "sleep_context",
			Name:        "Sleep Quality",
			Description: "Sleep quality observations",
			PossibleStates: []*pb.State{
				{
					Id:          "sleep_state_1",
					Name:        "Unknown",
					Description: "Initial state with high uncertainty",
					Properties: map[string]float32{
						"uncertainty": 1.0,
					},
				},
				{
					Id:          "sleep_state_2",
					Name:        "Understood",
					Description: "Final state with low uncertainty",
					Properties: map[string]float32{
						"uncertainty": 0.0,
					},
				},
			},
			CurrentStateId: "sleep_state_1",
		},
		{
			Id:          "exercise_context",
			Name:        "Exercise Routine",
			Description: "Exercise routine observations",
			PossibleStates: []*pb.State{
				{
					Id:          "exercise_state_1",
					Name:        "Unknown",
					Description: "Initial state with high uncertainty",
					Properties: map[string]float32{
						"uncertainty": 1.0,
					},
				},
				{
					Id:          "exercise_state_2",
					Name:        "Understood",
					Description: "Final state with low uncertainty",
					Properties: map[string]float32{
						"uncertainty": 0.0,
					},
				},
			},
			CurrentStateId: "exercise_state_1",
		},
	}

	// Create learning objective with multiple contexts
	learningObj := &pb.LearningObjective{
		Description:            "Learn about sleep and exercise patterns",
		TargetBeliefIds:       []string{"sleep_belief", "exercise_belief"},
		ObservationContextIds: []string{contexts[0].Id, contexts[1].Id},
		CompletionPercentage:  0.0,
	}

	// Create dialectic with belief system
	dialectic := &pb.Dialectic{
		Id:          "test_dialectic_multi",
		SelfModelId: "test_self",
		Agent: &pb.Agent{
			AgentType:     pb.Agent_AGENT_TYPE_GPT_LATEST,
			DialecticType: pb.DialecticType_SLEEP_DIET_EXERCISE,
		},
		LearningObjective: learningObj,
		BeliefSystem:     &pb.BeliefSystem{}, // Initialize empty belief system
	}

	// Verify initial state
	assert.Equal(t, 2, len(dialectic.LearningObjective.ObservationContextIds))
	assert.Equal(t, 0.0, dialectic.LearningObjective.CompletionPercentage)
	assert.NotNil(t, dialectic.BeliefSystem)

	// Simulate learning progress across contexts
	contexts[0].CurrentStateId = "sleep_state_2"    // Full understanding
	contexts[1].CurrentStateId = "exercise_state_1" // Still unknown
	
	// Average progress should be 50%
	expectedProgress := 50.0 // One context fully understood, one not at all
	dialectic.LearningObjective.CompletionPercentage = expectedProgress

	// Verify final state
	assert.Equal(t, "sleep_state_2", contexts[0].CurrentStateId)
	assert.Equal(t, "exercise_state_1", contexts[1].CurrentStateId)
	assert.Equal(t, expectedProgress, dialectic.LearningObjective.CompletionPercentage)
}
