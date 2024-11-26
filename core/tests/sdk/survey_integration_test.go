package integration

import (
	"context"
	"strconv"
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
	TestCreateSurveyDialectic(t)
	TestProcessSurveyAnswers(t)
}

func TestCreateSurveyDialectic(t *testing.T) {
	ctx := contextWithAPIKey(context.Background(), apiKey)

	// Create self model with survey contexts
	selfModelID := setupSelfModelWithSurveyContexts(t, ctx)

	// Create dialectic for survey
	createResp, err := client.CreateDialectic(ctx, connect.NewRequest(&pb.CreateDialecticRequest{
		SelfModelId:   selfModelID,
		DialecticType: pbmodels.DialecticType_DEFAULT,
	}))
	require.NoError(t, err)
	require.NotNil(t, createResp.Msg.Dialectic)

	dialectic := createResp.Msg.Dialectic
	assert.NotEmpty(t, dialectic.Id)
	assert.Equal(t, selfModelID, dialectic.SelfModelId)
	assert.Equal(t, pbmodels.DialecticType_DEFAULT, dialectic.Agent.DialecticType)

	testLogf(t, "Created survey dialectic with ID: %s", dialectic.Id)
}

func TestProcessSurveyAnswers(t *testing.T) {
	ctx := contextWithAPIKey(context.Background(), apiKey)
	selfModelID := setupSelfModelWithSurveyContexts(t, ctx)
	dialectic := createSurveyDialectic(t, ctx, selfModelID)

	// Process a series of survey answers
	answers := []struct {
		question string
		answer   int32
		context  string
		category string
	}{
		{
			question: "I consciously decide what time to go to bed based on how tired I feel.",
			answer:   4, // Agree
			context:  "before_sleep",
			category: "conscious_sleep_decisions",
		},
		{
			question: "My evening routine is influenced by subconscious cues like restlessness or calmness in my body.",
			answer:   5, // Strongly agree
			context:  "before_sleep",
			category: "subconscious_sleep_signals",
		},
	}

	for _, a := range answers {
		// Submit answer
		updateResp, err := client.UpdateDialectic(ctx, connect.NewRequest(&pb.UpdateDialecticRequest{
			Id:          dialectic.Id,
			SelfModelId: selfModelID,
			Answer: &pbmodels.UserAnswer{
				UserAnswer:         strconv.Itoa(int(a.answer)),
				CreatedAtMillisUtc: time.Now().UnixMilli(),
			},
		}))
		require.NoError(t, err)

		// Verify answer was recorded
		lastInteraction := updateResp.Msg.Dialectic.UserInteractions[len(updateResp.Msg.Dialectic.UserInteractions)-1]
		assert.Equal(t, pbmodels.STATUS_ANSWERED, lastInteraction.Status)
		qa := lastInteraction.GetQuestionAnswer()
		require.NotNil(t, qa)
		require.NotNil(t, qa.Answer)
		assert.Equal(t, strconv.Itoa(int(a.answer)), qa.Answer.UserAnswer)

		// Verify belief was created
		belief := verifyBeliefCreated(t, ctx, selfModelID, a.question, a.category)
		assert.NotNil(t, belief)
	}
}

// Helper functions

func setupSelfModelWithSurveyContexts(t *testing.T, ctx context.Context) string {
	selfModelID := "test-survey-user"

	// Create self model
	createResp, err := client.CreateSelfModel(ctx, connect.NewRequest(&pb.CreateSelfModelRequest{
		Id:           selfModelID,
		Philosophies: []string{"default"},
	}))
	require.NoError(t, err)
	require.NotNil(t, createResp.Msg.SelfModel)

	return selfModelID
}

func createSurveyDialectic(t *testing.T, ctx context.Context, selfModelID string) *pbmodels.Dialectic {
	createResp, err := client.CreateDialectic(ctx, connect.NewRequest(&pb.CreateDialecticRequest{
		SelfModelId:   selfModelID,
		DialecticType: pbmodels.DialecticType_DEFAULT,
	}))
	require.NoError(t, err)
	return createResp.Msg.Dialectic
}

func verifyBeliefCreated(t *testing.T, ctx context.Context, selfModelID string, statement string, category string) *pbmodels.Belief {
	// Get self model to check beliefs
	getResp, err := client.GetSelfModel(ctx, connect.NewRequest(&pb.GetSelfModelRequest{
		SelfModelId: selfModelID,
	}))
	require.NoError(t, err)

	// Find matching belief
	for _, belief := range getResp.Msg.SelfModel.BeliefSystem.Beliefs {
		contents := belief.Content
		for _, content := range contents {
			if content.RawStr == statement {
				return belief
			}
		}
	}
	return nil
}
