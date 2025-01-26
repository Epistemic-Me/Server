package integration

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "epistemic-me-core/pb"
	pbmodels "epistemic-me-core/pb/models"
)

// Custom logger that filters out noise
type testLogger struct {
	logger *log.Logger
}

func newTestLogger() *testLogger {
	return &testLogger{
		logger: log.New(os.Stdout, "", 0),
	}
}

func (l *testLogger) Printf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	if !strings.Contains(msg, "[cors]") && !strings.Contains(msg, "=== RUN") {
		l.logger.Print(msg)
	}
}

func suppressServerLogs() {
	// Suppress all server logs
	log.SetOutput(ioutil.Discard)
	// Suppress standard logger
	log.SetFlags(0)
	// Set log level to error only
	os.Setenv("GOLOG_LOG_LEVEL", "error")
}

func TestDialecticLearning(t *testing.T) {
	resetStore()
	TestCreateDialecticWithLearningObjective(t)
	TestDialecticLearningCycle(t)
}

func TestCreateDialecticWithLearningObjective(t *testing.T) {
	ctx := contextWithAPIKey(context.Background(), apiKey)
	selfModelID := setupSelfModelWithPhilosophy(t, ctx)

	createResp, err := client.CreateDialectic(ctx, connect.NewRequest(&pb.CreateDialecticRequest{
		SelfModelId:   selfModelID,
		DialecticType: pbmodels.DialecticType_DEFAULT,
		LearningObjective: &pbmodels.LearningObjective{
			Description:      "to learn my user's beliefs about sleep, diet and exercise including daily habits and the influences on their health beliefs",
			Topics:           []string{"sleep", "diet", "exercise", "health habits"},
			TargetBeliefType: pbmodels.BeliefType_STATEMENT,
			IsComplete:       false,
		},
	}))
	require.NoError(t, err)
	require.NotNil(t, createResp.Msg.Dialectic)

	dialectic := createResp.Msg.Dialectic
	assert.NotEmpty(t, dialectic.Id)
	assert.Equal(t, selfModelID, dialectic.SelfModelId)
	assert.NotNil(t, dialectic.LearningObjective)
	assert.Equal(t, []string{"sleep", "diet", "exercise", "health habits"}, dialectic.LearningObjective.Topics)

	verifyInitialInteraction(t, dialectic.UserInteractions)
	testLogf(t, "Created dialectic with learning objective, ID: %s", dialectic.Id)
}

func TestDialecticLearningCycle(t *testing.T) {
	// Suppress all server and test framework logs
	suppressServerLogs()
	logger := newTestLogger()

	// Set up test environment
	ctx := contextWithAPIKey(context.Background(), apiKey)
	selfModelID := setupSelfModelWithPhilosophy(t, ctx)

	// Create dialectic using helper function for consistency
	dialectic := createTestDialecticWithLearningObjective(t, ctx, selfModelID)
	createResp := &connect.Response[pb.CreateDialecticResponse]{Msg: &pb.CreateDialecticResponse{Dialectic: dialectic}}

	logger.Printf("\n=== Starting Learning Dialectic ===\n")
	logger.Printf("Learning Objective: %s\n\n", createResp.Msg.Dialectic.LearningObjective.Description)

	// Simulate conversation cycle
	for i := 0; i < 5; i++ {
		// Get latest question
		question := createResp.Msg.Dialectic.UserInteractions[len(createResp.Msg.Dialectic.UserInteractions)-1].Interaction.GetQuestionAnswer().Question.Question
		logger.Printf("Q%d: %s\n", i+1, question)

		// Generate test answer based on question content
		answer := generateTestAnswer(question)
		logger.Printf("A%d: %s\n", i+1, answer)

		// Update dialectic with answer
		updateResp, err := client.UpdateDialectic(ctx, connect.NewRequest(&pb.UpdateDialecticRequest{
			Id:          createResp.Msg.Dialectic.Id,
			SelfModelId: selfModelID,
			Answer: &pbmodels.UserAnswer{
				UserAnswer:         answer,
				CreatedAtMillisUtc: time.Now().UnixMilli(),
			},
		}))
		require.NoError(t, err)

		// Log extracted belief and completion status
		lastInteraction := updateResp.Msg.Dialectic.UserInteractions[len(updateResp.Msg.Dialectic.UserInteractions)-2]
		if len(lastInteraction.Interaction.GetQuestionAnswer().ExtractedBeliefs) > 0 {
			belief := lastInteraction.Interaction.GetQuestionAnswer().ExtractedBeliefs[0].Content[0].RawStr
			logger.Printf("Extracted Belief: %s\n", belief)
		}
		logger.Printf("Learning Complete: %v\n\n", updateResp.Msg.Dialectic.LearningObjective.IsComplete)

		createResp = &connect.Response[pb.CreateDialecticResponse]{Msg: &pb.CreateDialecticResponse{Dialectic: updateResp.Msg.Dialectic}}
	}

	// Verify beliefs were extracted and added to self model
	selfModel, err := client.GetSelfModel(ctx, connect.NewRequest(&pb.GetSelfModelRequest{
		SelfModelId: selfModelID,
	}))
	require.NoError(t, err)
	require.NotNil(t, selfModel)

	beliefSystem := selfModel.Msg.SelfModel.BeliefSystem
	require.NotNil(t, beliefSystem)
	require.NotEmpty(t, beliefSystem.Beliefs, "Should NOT be empty, but was %v", beliefSystem.Beliefs)
}

// Helper functions

func setupSelfModelWithPhilosophy(t *testing.T, ctx context.Context) string {
	selfModelID := "test-health-philosophy-user"
	err := CreateInitialBeliefSystemIfNotExists(selfModelID)
	require.NoError(t, err)

	// Create self-model with health philosophy
	createResp, err := client.CreateSelfModel(ctx, connect.NewRequest(&pb.CreateSelfModelRequest{
		Id:           selfModelID,
		Philosophies: []string{"life live to be healthy but comfortable"},
	}))
	require.NoError(t, err)
	require.NotNil(t, createResp.Msg.SelfModel)

	return selfModelID
}

func createTestDialecticWithLearningObjective(t *testing.T, ctx context.Context, selfModelID string) *pbmodels.Dialectic {
	createResp, err := client.CreateDialectic(ctx, connect.NewRequest(&pb.CreateDialecticRequest{
		SelfModelId:   selfModelID,
		DialecticType: pbmodels.DialecticType_DEFAULT,
		LearningObjective: &pbmodels.LearningObjective{
			Description:      "to learn my user's beliefs about sleep, diet and exercise including daily habits and the influences on their health beliefs",
			Topics:           []string{"sleep", "diet", "exercise", "health habits"},
			TargetBeliefType: pbmodels.BeliefType_STATEMENT,
			IsComplete:       false,
		},
	}))
	require.NoError(t, err)
	return createResp.Msg.Dialectic
}

func generateTestAnswer(question string) string {
	question = strings.ToLower(question)
	switch {
	case strings.Contains(question, "sleep"):
		return "I believe getting 8 hours of sleep is essential for health, and I try to maintain a consistent sleep schedule."
	case strings.Contains(question, "diet") || strings.Contains(question, "food"):
		return "I believe in eating a balanced diet with plenty of vegetables and limiting processed foods."
	case strings.Contains(question, "exercise"):
		return "I believe regular exercise is crucial, so I try to work out at least 3 times a week."
	default:
		return "I believe in maintaining healthy habits for overall wellbeing."
	}
}

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
