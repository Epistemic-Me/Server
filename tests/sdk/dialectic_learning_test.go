package integration

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ai "epistemic-me-core/ai"
	pb "epistemic-me-core/pb"
	pbmodels "epistemic-me-core/pb/models"
	svcmodels "epistemic-me-core/svc/models"
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
			Description:          "to learn my user's beliefs about sleep, diet and exercise including daily habits and the influences on their health beliefs",
			Topics:               []string{"sleep", "diet", "exercise", "health habits"},
			TargetBeliefType:     pbmodels.BeliefType_STATEMENT,
			CompletionPercentage: 0,
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
	// Check for OpenAI API key
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("Skipping test: OPENAI_API_KEY not set")
	}

	// Create a test logger that writes to both t.Log and stdout
	logger := log.New(io.MultiWriter(os.Stdout, testWriter{t}), "", 0)

	// Set up test environment
	ctx := contextWithAPIKey(context.Background(), apiKey)
	selfModelID := setupSelfModelWithPhilosophy(t, ctx)

	// Create dialectic using helper function for consistency
	dialectic := createTestDialecticWithLearningObjective(t, ctx, selfModelID)
	createResp := &connect.Response[pb.CreateDialecticResponse]{Msg: &pb.CreateDialecticResponse{Dialectic: dialectic}}

	logger.Printf("\n=== Starting Learning Dialectic ===")
	logger.Printf("Learning Objective: %s\n\n", createResp.Msg.Dialectic.LearningObjective.Description)

	// Track previous completion percentage
	var prevCompletion float32 = 0

	// Test a few rounds of interaction
	for i := 0; i < 5; i++ {
		// Get latest question
		if len(createResp.Msg.Dialectic.UserInteractions) == 0 {
			logger.Printf("No more questions - learning objective complete\n")
			break
		}
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

		// Log extracted beliefs and completion status
		if len(updateResp.Msg.Dialectic.UserInteractions) > 0 {
			var lastAnsweredInteraction *pbmodels.DialecticalInteraction
			// Find the last answered interaction
			for i := len(updateResp.Msg.Dialectic.UserInteractions) - 1; i >= 0; i-- {
				if updateResp.Msg.Dialectic.UserInteractions[i].Status == pbmodels.STATUS_ANSWERED {
					lastAnsweredInteraction = updateResp.Msg.Dialectic.UserInteractions[i]
					break
				}
			}

			if lastAnsweredInteraction != nil && len(lastAnsweredInteraction.Interaction.GetQuestionAnswer().ExtractedBeliefs) > 0 {
				logger.Printf("Extracted Beliefs:")
				for i, belief := range lastAnsweredInteraction.Interaction.GetQuestionAnswer().ExtractedBeliefs {
					logger.Printf("  %d. %s", i+1, belief.Content[0].RawStr)
				}
			} else {
				logger.Printf("No beliefs extracted from this interaction.")
			}
		}
		logger.Printf("Learning Complete: %v\n\n", updateResp.Msg.Dialectic.LearningObjective.CompletionPercentage)

		// Log completion analysis details
		if updateResp.Msg.Dialectic.LearningObjective.CompletionPercentage < prevCompletion {
			logger.Printf("\n!!! Warning: Completion percentage decreased from %.1f%% to %.1f%% !!!\n",
				prevCompletion,
				updateResp.Msg.Dialectic.LearningObjective.CompletionPercentage)
		}
		prevCompletion = updateResp.Msg.Dialectic.LearningObjective.CompletionPercentage

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

// Helper type to write to testing.T
type testWriter struct {
	t *testing.T
}

func (tw testWriter) Write(p []byte) (n int, err error) {
	tw.t.Log(string(p))
	return len(p), nil
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
			Description:          "to learn my user's beliefs about sleep, diet and exercise including daily habits and the influences on their health beliefs",
			Topics:               []string{"sleep", "diet", "exercise", "health habits"},
			TargetBeliefType:     pbmodels.BeliefType_STATEMENT,
			CompletionPercentage: 0,
		},
	}))
	require.NoError(t, err)
	return createResp.Msg.Dialectic
}

func generateTestAnswer(question string) string {
	// Get the self model's beliefs to generate a contextual answer
	ctx := contextWithAPIKey(context.Background(), apiKey)
	selfModel, err := client.GetSelfModel(ctx, connect.NewRequest(&pb.GetSelfModelRequest{
		SelfModelId: "test-health-philosophy-user",
	}))
	if err != nil {
		// Fallback to default answers if we can't get the belief system
		return generateDefaultAnswer(question)
	}

	// Convert proto belief system to service model
	beliefSystem := &svcmodels.BeliefSystem{
		Beliefs: make([]*svcmodels.Belief, len(selfModel.Msg.SelfModel.BeliefSystem.Beliefs)),
	}
	for i, belief := range selfModel.Msg.SelfModel.BeliefSystem.Beliefs {
		beliefSystem.Beliefs[i] = &svcmodels.Belief{
			Content: []svcmodels.Content{{RawStr: belief.Content[0].RawStr}},
		}
	}

	// Create AI helper with the test API key
	helper := ai.NewAIHelper(os.Getenv("OPENAI_API_KEY"))

	// Use AI helper to generate an answer based on the belief system
	answer, err := helper.GenerateAnswerFromBeliefSystem(
		question,
		beliefSystem,
		selfModel.Msg.SelfModel.Philosophies,
	)
	if err != nil {
		// Fallback to default answers if AI generation fails
		return generateDefaultAnswer(question)
	}

	return answer
}

func generateDefaultAnswer(question string) string {
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
