package integration

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "epistemic-me-core/pb"
	pbmodels "epistemic-me-core/pb/models"
)

func TestSurveyIntegration(t *testing.T) {
	resetStore()
	TestCreateSurveyWithContexts(t)
	TestCompleteSurveySection(t)
}

func TestCreateSurveyWithContexts(t *testing.T) {
	ctx := contextWithAPIKey(context.Background(), apiKey)

	// Create self model with survey contexts
	selfModelID := setupSelfModelWithSurveyContexts(t, ctx)

	// Create dialectic for survey section
	createResp, err := client.CreateDialectic(ctx, connect.NewRequest(&pb.CreateDialecticRequest{
		SelfModelId:   selfModelID,
		DialecticType: pbmodels.DialecticType_DEFAULT,
	}))
	require.NoError(t, err)
	require.NotNil(t, createResp.Msg.Dialectic)

	dialectic := createResp.Msg.Dialectic
	assert.NotEmpty(t, dialectic.Id)
	assert.Equal(t, selfModelID, dialectic.SelfModelId)

	testLogf(t, "Created survey dialectic with ID: %s", dialectic.Id)
}

func TestCompleteSurveySection(t *testing.T) {
	ctx := contextWithAPIKey(context.Background(), apiKey)
	selfModelID := setupSelfModelWithSurveyContexts(t, ctx)

	// Create dialectic for "before_sleep" section
	dialectic := createTestDialectic(t, ctx, selfModelID)

	// Define survey section structure
	beforeSleepQuestions := []struct {
		question string
		answer   string
		category string
		prompt   string
	}{
		{
			prompt:   "Think about the decisions you make in the evening as you prepare for bed...",
			question: "I consciously decide what time to go to bed based on how tired I feel.",
			answer:   "4", // Agree
			category: "conscious_sleep_decisions",
		},
		{
			prompt:   "Think about the decisions you make in the evening as you prepare for bed...",
			question: "My evening routine is influenced by subconscious cues like restlessness or calmness in my body.",
			answer:   "5", // Strongly agree
			category: "subconscious_sleep_signals",
		},
	}

	// Process each question in the section
	for _, q := range beforeSleepQuestions {
		// First, update dialectic with context prompt
		_, err := client.UpdateDialectic(ctx, connect.NewRequest(&pb.UpdateDialecticRequest{
			Id:             dialectic.Id,
			SelfModelId:    selfModelID,
			CustomQuestion: q.prompt,
		}))
		require.NoError(t, err)

		// Then create question interaction
		_, err = client.UpdateDialectic(ctx, connect.NewRequest(&pb.UpdateDialecticRequest{
			Id:             dialectic.Id,
			SelfModelId:    selfModelID,
			CustomQuestion: q.question,
			Answer: &pbmodels.UserAnswer{
				UserAnswer:         q.answer,
				CreatedAtMillisUtc: time.Now().UnixMilli(),
			},
		}))
		require.NoError(t, err)

		// Verify belief was created with full context
		belief := verifyBeliefCreated(t, ctx, selfModelID, q.question)
		require.NotNil(t, belief)
		// assert.Equal(t, q.category, belief.Metadata["category"])
		// assert.Equal(t, "before_sleep", belief.Metadata["context"])
		// assert.Equal(t, q.prompt, belief.Metadata["prompt"])
	}

	// Verify final belief system state
	getResp, err := client.GetSelfModel(ctx, connect.NewRequest(&pb.GetSelfModelRequest{
		SelfModelId: selfModelID,
	}))
	require.NoError(t, err)

	beliefSystem := getResp.Msg.SelfModel.BeliefSystem
	assert.Equal(t, len(beforeSleepQuestions), len(beliefSystem.Beliefs))
}

// Helper functions

func setupSelfModelWithSurveyContexts(t *testing.T, ctx context.Context) string {
	selfModelID := "test-survey-user"

	// Create self model with default philosophy
	createResp, err := client.CreateSelfModel(ctx, connect.NewRequest(&pb.CreateSelfModelRequest{
		Id:           selfModelID,
		Philosophies: []string{"default"},
	}))
	require.NoError(t, err)
	require.NotNil(t, createResp.Msg.SelfModel)

	// Create initial belief system
	err = CreateInitialBeliefSystemIfNotExists(selfModelID)
	require.NoError(t, err)

	return selfModelID
}

func verifyBeliefCreated(t *testing.T, ctx context.Context, selfModelID string, statement string) *pbmodels.Belief {
	getResp, err := client.GetSelfModel(ctx, connect.NewRequest(&pb.GetSelfModelRequest{
		SelfModelId: selfModelID,
	}))
	require.NoError(t, err)

	for _, belief := range getResp.Msg.SelfModel.BeliefSystem.Beliefs {
		if belief.Content[0].GetRawStr() == statement {
			return belief
		}
	}
	return nil
}
