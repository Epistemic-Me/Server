package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	pb "epistemic-me-core/pb"
)

type MessageGroup struct {
	Questions []string
	Answer    string
}

func testLogQA(t *testing.T, format string, v ...interface{}) {
	if testing.Verbose() {
		t.Logf(format, v...)
	}
}

func logMessageGroup(t *testing.T, group MessageGroup, index int) {
	t.Logf("\nMessage Group %d:", index+1)
	t.Log("Questions:")
	for i, q := range group.Questions {
		t.Logf("  %d. %s", i+1, q)
	}
	t.Log("Answer:")
	t.Logf("  %s", group.Answer)
	t.Log("---")
}

func TestPreprocessQuestionAnswerIntegration(t *testing.T) {
	// Setup server and client
	resetStore()
	ctx := contextWithAPIKey(context.Background(), apiKey)

	// Load test data
	jsonPath := filepath.Join("..", "..", "db", "fixtures", "structured_conversations.json")
	data, err := os.ReadFile(jsonPath)
	require.NoError(t, err)

	var conversations StructuredConversation
	err = json.Unmarshal(data, &conversations)
	require.NoError(t, err)

	// Test specific conversation
	conv := conversations.Conversations[1]
	require.Equal(t, "jonathan@teammachine.ai", conv.ParticipantID)

	// Send the entire conversation to the preprocessing endpoint
	resp, err := client.PreprocessQuestionAnswer(ctx, connect.NewRequest(&pb.PreprocessQuestionAnswerRequest{
		QuestionBlobs: []string{conv.Messages[1].Content}, // Assistant message with questions
		AnswerBlobs:   []string{conv.Messages[2].Content}, // User message with answers
	}))
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify the server matched questions with answers correctly
	pairs := resp.Msg.QaPairs
	require.NotEmpty(t, pairs, "Should have extracted at least one QA pair")

	t.Run("processes sleep and diet questions", func(t *testing.T) {
		// Test sleep questions (Messages 1-2)
		resp, err := client.PreprocessQuestionAnswer(ctx, connect.NewRequest(&pb.PreprocessQuestionAnswerRequest{
			QuestionBlobs: []string{conv.Messages[1].Content}, // Sleep questions
			AnswerBlobs:   []string{conv.Messages[2].Content}, // Sleep answers
		}))
		require.NoError(t, err)
		require.NotNil(t, resp)

		t.Log("\nProcessing Sleep Questions:")
		t.Logf("Question Blob: %s", conv.Messages[1].Content)
		t.Logf("Answer Blob: %s", conv.Messages[2].Content)

		pairs := resp.Msg.QaPairs
		require.NotEmpty(t, pairs)

		// Test diet questions (Messages 3-4)
		resp2, err := client.PreprocessQuestionAnswer(ctx, connect.NewRequest(&pb.PreprocessQuestionAnswerRequest{
			QuestionBlobs: []string{conv.Messages[3].Content}, // Diet questions
			AnswerBlobs:   []string{conv.Messages[4].Content}, // Diet answers
		}))
		require.NoError(t, err)

		t.Log("\nProcessing Diet Questions:")
		t.Logf("Question Blob: %s", conv.Messages[3].Content)
		t.Logf("Answer Blob: %s", conv.Messages[4].Content)

		pairs2 := resp2.Msg.QaPairs
		require.NotEmpty(t, pairs2)

		// Log all processed pairs
		t.Log("\nProcessed QA Pairs:")
		for i, pair := range pairs {
			t.Logf("Sleep Pair %d:", i+1)
			t.Logf("Q: %s", pair.Question)
			t.Logf("A: %s", pair.Answer)
		}
		for i, pair := range pairs2 {
			t.Logf("Diet Pair %d:", i+1)
			t.Logf("Q: %s", pair.Question)
			t.Logf("A: %s", pair.Answer)
		}

		// Verify all diet questions are matched correctly
		t.Log("\nVerifying Diet Question Matches:")
		for _, pair := range pairs2 {
			t.Logf("Checking question: %s", pair.Question)

			switch pair.Question {
			case "Can you describe a typical day's meals for you?":
				require.Contains(t, pair.Answer, "shake for breakfast",
					"Meals question should match breakfast description")

			case "How do you make your food choices?":
				require.Contains(t, pair.Answer, "based on nutrition",
					"Food choices question should match nutrition criteria")
				require.Contains(t, pair.Answer, "hunger and enjoyment",
					"Food choices question should match all criteria")

			case "What specific beliefs influence your dietary choices?":
				require.Contains(t, pair.Answer, "manage metabolic health",
					"Dietary beliefs question should match health beliefs")

			case "What do you believe are the benefits of your current diet?":
				require.Contains(t, pair.Answer, "gives me energy",
					"Diet benefits question should match energy benefits")
				require.Contains(t, pair.Answer, "healthy as measured by my blood tests",
					"Diet benefits question should match health benefits")
			}
		}
	})
}

func isInitMessage(content string) bool {
	return len(content) > 0 && content[:9] == "User name"
}
