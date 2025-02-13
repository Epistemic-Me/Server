package integration

import (
	"testing"

	pb "epistemic-me-core/pb/models"
	"github.com/stretchr/testify/assert"
)

func TestLearningObjectiveStructure(t *testing.T) {
	// Test case 1: Create a valid learning objective
	obj := &pb.LearningObjective{
		Description:         "Learn about user's sleep patterns",
		TargetBeliefIds:    []string{"belief_1", "belief_2"},
		ObservationContextIds: []string{"context_1", "context_2"},
		CompletionPercentage: 0.0,
	}

	// Verify all fields are set correctly
	assert.Equal(t, "Learn about user's sleep patterns", obj.Description)
	assert.Equal(t, 2, len(obj.TargetBeliefIds))
	assert.Equal(t, 2, len(obj.ObservationContextIds))
	assert.Equal(t, 0.0, obj.CompletionPercentage)

	// Test case 2: Verify field types
	assert.IsType(t, "", obj.Description)
	assert.IsType(t, []string{}, obj.TargetBeliefIds)
	assert.IsType(t, []string{}, obj.ObservationContextIds)
	assert.IsType(t, float64(0), obj.CompletionPercentage)
}

func TestDialecticWithLearningObjective(t *testing.T) {
	// Create a dialectic with learning objective
	dialectic := &pb.Dialectic{
		Id:          "test_dialectic",
		SelfModelId: "test_self",
		Agent: &pb.Agent{
			AgentType:     pb.Agent_AGENT_TYPE_GPT_LATEST,
			DialecticType: pb.DialecticType_SLEEP_DIET_EXERCISE,
		},
		LearningObjective: &pb.LearningObjective{
			Description:         "Learn about user's sleep patterns",
			TargetBeliefIds:    []string{"belief_1"},
			ObservationContextIds: []string{"context_1"},
			CompletionPercentage: 0.0,
		},
	}

	// Verify dialectic structure
	assert.NotNil(t, dialectic.LearningObjective)
	assert.Equal(t, "test_dialectic", dialectic.Id)
	assert.Equal(t, pb.DialecticType_SLEEP_DIET_EXERCISE, dialectic.Agent.DialecticType)

	// Verify learning objective is properly linked
	assert.Equal(t, "Learn about user's sleep patterns", dialectic.LearningObjective.Description)
	assert.Equal(t, 1, len(dialectic.LearningObjective.TargetBeliefIds))
	assert.Equal(t, 1, len(dialectic.LearningObjective.ObservationContextIds))
}
