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
	svc_models "epistemic-me-core/svc/models"
)

func TestDialecticQuestionAnswer(t *testing.T) {
	resetStore()
	TestCreateDialectic(t)
	TestUpdateDialecticWithAnswer(t)
	TestCustomQuestionDialectic(t)
}

func TestCreateDialectic(t *testing.T) {
	ctx := contextWithAPIKey(context.Background(), apiKey)
	selfModelID := setupSelfModelWithBelief(t, ctx)

	createResp, err := client.CreateDialectic(ctx, connect.NewRequest(&pb.CreateDialecticRequest{
		SelfModelId:   selfModelID,
		DialecticType: pbmodels.DialecticType_DEFAULT,
	}))
	require.NoError(t, err)
	require.NotNil(t, createResp.Msg.Dialectic)

	dialectic := createResp.Msg.Dialectic
	assert.NotEmpty(t, dialectic.Id)
	assert.Equal(t, selfModelID, dialectic.SelfModelId)
	assert.NotNil(t, dialectic.Agent)
	assert.Equal(t, pbmodels.DialecticType_DEFAULT, dialectic.Agent.DialecticType)

	verifyInitialInteraction(t, dialectic.UserInteractions)
	testLogf(t, "Created dialectic with ID: %s", dialectic.Id)
}

func TestUpdateDialecticWithAnswer(t *testing.T) {
	ctx := contextWithAPIKey(context.Background(), apiKey)
	selfModelID := setupSelfModelWithBelief(t, ctx)
	dialectic := createTestDialectic(t, ctx, selfModelID)

	userAnswer := "This is my test answer"
	updateResp, err := client.UpdateDialectic(ctx, connect.NewRequest(&pb.UpdateDialecticRequest{
		Id:          dialectic.Id,
		SelfModelId: selfModelID,
		Answer: &pbmodels.UserAnswer{
			UserAnswer:         userAnswer,
			CreatedAtMillisUtc: time.Now().UnixMilli(),
		},
	}))
	require.NoError(t, err)

	// Get the first interaction and verify its status and answer
	interaction := updateResp.Msg.Dialectic.UserInteractions[0]
	assert.Equal(t, pbmodels.STATUS_ANSWERED, interaction.Status)
	qa := interaction.GetQuestionAnswer()
	require.NotNil(t, qa)
	require.NotNil(t, qa.Answer)
	assert.Equal(t, userAnswer, qa.Answer.UserAnswer)
}

func TestCustomQuestionDialectic(t *testing.T) {
	ctx := contextWithAPIKey(context.Background(), apiKey)
	selfModelID := setupSelfModelWithBelief(t, ctx)
	dialectic := createTestDialectic(t, ctx, selfModelID)

	customQuestion := "What is your opinion on meditation?"
	customAnswer := "Meditation helps reduce stress"
	updateResp, err := client.UpdateDialectic(ctx, connect.NewRequest(&pb.UpdateDialecticRequest{
		Id:             dialectic.Id,
		SelfModelId:    selfModelID,
		CustomQuestion: customQuestion,
		Answer: &pbmodels.UserAnswer{
			UserAnswer:         customAnswer,
			CreatedAtMillisUtc: time.Now().UnixMilli(),
		},
	}))
	require.NoError(t, err)

	// Verify the custom question was used
	dialectic = updateResp.Msg.Dialectic
	require.Len(t, dialectic.UserInteractions, 2)
	lastInteraction := dialectic.UserInteractions[1]
	qa := lastInteraction.GetQuestionAnswer()
	require.NotNil(t, qa)
	require.NotNil(t, qa.Question)
	assert.Equal(t, customQuestion, qa.Question.Question)
}

// Helper functions
func setupSelfModelWithBelief(t *testing.T, ctx context.Context) string {
	selfModelID := "test-user-id"
	err := CreateInitialBeliefSystemIfNotExists(selfModelID)
	require.NoError(t, err)

	// Create self-model and store the response
	createResp, err := client.CreateSelfModel(ctx, connect.NewRequest(&pb.CreateSelfModelRequest{
		Id:           selfModelID,
		Philosophies: []string{"default"},
	}))
	require.NoError(t, err)
	require.NotNil(t, createResp.Msg.SelfModel)

	// Create initial belief
	_, err = client.CreateBelief(ctx, connect.NewRequest(&pb.CreateBeliefRequest{
		SelfModelId:   selfModelID,
		BeliefContent: "Regular exercise and balanced diet contribute to overall health",
		BeliefType:    pbmodels.BeliefType_STATEMENT,
	}))
	require.NoError(t, err)

	// Wait for belief system to be stored
	time.Sleep(100 * time.Millisecond)

	return selfModelID
}

func createTestDialectic(t *testing.T, ctx context.Context, selfModelID string) *pbmodels.Dialectic {
	createResp, err := client.CreateDialectic(ctx, connect.NewRequest(&pb.CreateDialecticRequest{
		SelfModelId:   selfModelID,
		DialecticType: pbmodels.DialecticType_DEFAULT,
	}))
	require.NoError(t, err)
	return createResp.Msg.Dialectic
}

func verifyInitialInteraction(t *testing.T, interactions []*pbmodels.DialecticalInteraction) {
	require.NotEmpty(t, interactions)
	interaction := interactions[0]
	assert.Equal(t, pbmodels.STATUS_PENDING_ANSWER, interaction.Status)
	assert.Equal(t, pbmodels.InteractionType_QUESTION_ANSWER, interaction.Type)

	qa := interaction.GetQuestionAnswer()
	require.NotNil(t, qa)
	require.NotNil(t, qa.Question)
	assert.NotEmpty(t, qa.Question.Question)
}

func verifyUpdatedDialectic(t *testing.T, dialectic *pbmodels.Dialectic, expectedAnswer string) {
	require.NotEmpty(t, dialectic.UserInteractions)

	// Verify answered interaction
	answeredInteraction := dialectic.UserInteractions[0]
	assert.Equal(t, pbmodels.STATUS_ANSWERED, answeredInteraction.Status)
	qa := answeredInteraction.GetQuestionAnswer()
	require.NotNil(t, qa)
	require.NotNil(t, qa.Answer)
	assert.Equal(t, expectedAnswer, qa.Answer.UserAnswer)

	// Verify new interaction
	require.Len(t, dialectic.UserInteractions, 2)
	newInteraction := dialectic.UserInteractions[1]
	assert.Equal(t, pbmodels.STATUS_PENDING_ANSWER, newInteraction.Status)
	assert.Equal(t, pbmodels.InteractionType_QUESTION_ANSWER, newInteraction.Type)
	newQA := newInteraction.GetQuestionAnswer()
	require.NotNil(t, newQA)
	require.NotNil(t, newQA.Question)
	assert.NotEmpty(t, newQA.Question.Question)
}

func CreateInitialBeliefSystemIfNotExists(selfModelID string) error {
	beliefSystem := &svc_models.BeliefSystem{
		Beliefs:             []*svc_models.Belief{},
		ObservationContexts: []*svc_models.ObservationContext{},
		BeliefContexts:      []*svc_models.BeliefContext{},
	}

	// Store with both keys for compatibility
	err := kvStore.Store(selfModelID, "BeliefSystem", *beliefSystem, 1)
	if err != nil {
		return err
	}

	err = kvStore.Store(selfModelID, "BeliefSystemId", *beliefSystem, 1)
	if err != nil {
		return err
	}

	return nil
}
