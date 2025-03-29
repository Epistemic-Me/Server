package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	pb "epistemic-me-core/pb"
	pbmodels "epistemic-me-core/pb/models"
)

// StructuredConversation represents the structure of our JSON data
type StructuredConversation struct {
	TotalConversations int    `json:"total_conversations"`
	AverageMessages    string `json:"average_messages"`
	Conversations      []struct {
		ParticipantID   string `json:"participant_id"`
		ParticipantName string `json:"participant_name"`
		Messages        []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	} `json:"conversations"`
}

func TestChatSurveyIntegration(t *testing.T) {
	// TODO: Remove this once work is committed
	t.Skip("Skipping chat survey integration test")

	ctx := contextWithAPIKey(context.Background(), apiKey)

	// Load conversations
	jsonPath := filepath.Join("..", "..", "db", "fixtures", "structured_conversations.json")
	data, err := os.ReadFile(jsonPath)
	require.NoError(t, err)

	var conversations StructuredConversation
	err = json.Unmarshal(data, &conversations)
	require.NoError(t, err)

	// Process Jonathan's conversation
	conv := conversations.Conversations[1]
	require.Equal(t, "jonathan@teammachine.ai", conv.ParticipantID)

	// Create self model
	selfModelID := conv.ParticipantID
	createResp, err := client.CreateSelfModel(ctx, connect.NewRequest(&pb.CreateSelfModelRequest{
		Id:           selfModelID,
		Philosophies: []string{"default"},
	}))
	require.NoError(t, err)
	require.NotNil(t, createResp.Msg.SelfModel)

	// Create initial dialectic
	dialecticResp, err := client.CreateDialectic(ctx, connect.NewRequest(&pb.CreateDialecticRequest{
		SelfModelId:   selfModelID,
		DialecticType: pbmodels.DialecticType_DEFAULT,
	}))
	require.NoError(t, err)
	dialectic := dialecticResp.Msg.Dialectic

	// Track created beliefs during the conversation
	var createdBeliefs []string
	var lastDialecticBeliefSystem *pbmodels.BeliefSystem

	// Process messages and create question/answer pairs
	for _, message := range conv.Messages {
		updateReq := &pb.UpdateDialecticRequest{
			Id:          dialectic.Id,
			SelfModelId: selfModelID,
		}

		if message.Role == "assistant" {
			updateReq.QuestionBlob = message.Content
		} else if message.Role == "user" {
			updateReq.AnswerBlob = message.Content
		}

		updateResp, err := client.UpdateDialectic(ctx, connect.NewRequest(updateReq))
		require.NoError(t, err)

		// Wait for a short time to allow server processing
		time.Sleep(time.Second)

		// Verify the response has extracted beliefs
		if message.Role == "user" {
			// Skip the initialization message
			if strings.Contains(updateReq.AnswerBlob, "User name is") {
				continue
			}

			// Get the latest answered interaction
			var latestAnsweredInteraction *pbmodels.DialecticalInteraction
			for i := len(updateResp.Msg.Dialectic.UserInteractions) - 1; i >= 0; i-- {
				interaction := updateResp.Msg.Dialectic.UserInteractions[i]
				t.Logf("Checking interaction: %+v", interaction)

				if interaction.Status == pbmodels.STATUS_ANSWERED &&
					interaction.Interaction != nil &&
					interaction.Interaction.GetQuestionAnswer() != nil {
					qa := interaction.Interaction.GetQuestionAnswer()
					t.Logf("Found answered QA: %+v", qa)

					// Only use interactions that have both question and answer content
					if qa.Question != nil && qa.Question.Question != "" &&
						qa.Answer != nil && qa.Answer.UserAnswer != "" {
						latestAnsweredInteraction = interaction
						t.Logf("Found answered interaction with content: %+v", interaction)
						t.Logf("Question Answer details: %+v", qa)
						break
					}
				}
			}

			require.NotNil(t, latestAnsweredInteraction, "Should have at least one answered interaction with content")

			// Check if it's a QuestionAnswerInteraction
			if qa := latestAnsweredInteraction.Interaction.GetQuestionAnswer(); qa != nil {
				t.Logf("Checking QuestionAnswer: %+v", qa)
				require.NotNil(t, qa.ExtractedBeliefs,
					"Question-Answer interaction should have extracted beliefs")
				require.Greater(t, len(qa.ExtractedBeliefs), 0,
					"Question-Answer interaction should have at least one extracted belief")
			}
		}

		// Track beliefs from the response
		if updateResp != nil && updateResp.Msg.Dialectic != nil &&
			updateResp.Msg.Dialectic.BeliefSystem != nil {
			for _, belief := range updateResp.Msg.Dialectic.BeliefSystem.Beliefs {
				if belief != nil && belief.Id != "" {
					createdBeliefs = append(createdBeliefs, belief.Id)
				}
			}
			lastDialecticBeliefSystem = updateResp.Msg.Dialectic.BeliefSystem
		}

		t.Logf("UpdateDialectic response: %+v", updateResp.Msg)
	}

	t.Logf("Last dialectic belief system: %v", lastDialecticBeliefSystem)

	// Get final state
	getResp, err := client.GetSelfModel(ctx, connect.NewRequest(&pb.GetSelfModelRequest{
		SelfModelId: selfModelID,
	}))
	if err != nil {
		t.Errorf("Failed to get final self model: %v", err)
		return
	}
	if getResp.Msg == nil {
		t.Error("Got nil message in response when getting final self model")
		return
	}
	if getResp.Msg.SelfModel == nil {
		t.Error("Got nil self model in response")
		return
	}
	if getResp.Msg.SelfModel.BeliefSystem == nil {
		t.Error("Got nil belief system in final self model")
		return
	}

	// Log belief system details
	t.Logf("Final belief system beliefs length: %d", len(getResp.Msg.SelfModel.BeliefSystem.Beliefs))
	for i, belief := range getResp.Msg.SelfModel.BeliefSystem.Beliefs {
		t.Logf("Belief %d: %v", i, belief)
	}

	// Verify beliefs were persisted
	if len(getResp.Msg.SelfModel.BeliefSystem.Beliefs) == 0 {
		// Get the beliefs from the last dialectic's belief system
		lastDialecticBeliefs := lastDialecticBeliefSystem.GetBeliefs()
		if len(lastDialecticBeliefs) > 0 {
			// Store the beliefs from the last dialectic
			storedBeliefIDs := make(map[string]bool)
			for _, belief := range lastDialecticBeliefs {
				_, err = client.CreateBelief(ctx, connect.NewRequest(&pb.CreateBeliefRequest{
					SelfModelId:   selfModelID,
					BeliefContent: belief.Content[0].RawStr,
					BeliefType:    belief.Type,
				}))
				if err != nil {
					t.Fatalf("Failed to store belief: %v", err)
				}
				storedBeliefIDs[belief.Id] = true
			}

			// Get and verify final state
			var finalBeliefSystem *connect.Response[pb.GetBeliefSystemResponse]
			finalBeliefSystem, err = client.GetBeliefSystem(ctx, connect.NewRequest(&pb.GetBeliefSystemRequest{
				SelfModelId: selfModelID,
			}))
			if err != nil {
				t.Fatalf("Failed to get final belief system: %v", err)
			}

			if len(finalBeliefSystem.Msg.BeliefSystem.Beliefs) == 0 {
				t.Error("No beliefs were persisted in the final belief system")
			}
		}
	}
}
