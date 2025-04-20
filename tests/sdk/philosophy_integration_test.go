package integration

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "epistemic-me-core/pb"
)

func TestCreatePhilosophy(t *testing.T) {
	resetStore()
	ctx := contextWithAPIKey(context.Background(), apiKey)

	createReq := &pb.CreatePhilosophyRequest{
		Description:         "# Test Philosophy\n\n## Narrative\n[[C: Context1]] [[S: state1]] → [[S: state2]]\n",
		ExtrapolateContexts: true,
	}
	createResp, err := client.CreatePhilosophy(ctx, connect.NewRequest(createReq))
	require.NoError(t, err)
	require.NotNil(t, createResp.Msg.Philosophy)
	assert.Equal(t, createReq.Description, createResp.Msg.Philosophy.Description)
	assert.Equal(t, createReq.ExtrapolateContexts, createResp.Msg.Philosophy.ExtrapolateContexts)
}

func TestUpdatePhilosophy(t *testing.T) {
	resetStore()
	ctx := contextWithAPIKey(context.Background(), apiKey)

	// Create first
	createReq := &pb.CreatePhilosophyRequest{
		Description:         "# Test Philosophy\n\n## Narrative\n[[C: Context1]] [[S: state1]] → [[S: state2]]\n",
		ExtrapolateContexts: true,
	}
	createResp, err := client.CreatePhilosophy(ctx, connect.NewRequest(createReq))
	require.NoError(t, err)
	philosophyID := createResp.Msg.Philosophy.Id

	// Update
	updateReq := &pb.UpdatePhilosophyRequest{
		PhilosophyId:        philosophyID,
		Description:         "# Updated Philosophy\n\n## Narrative\n[[C: Context2]] [[S: state3]] → [[S: state4]]\n",
		ExtrapolateContexts: false,
	}
	updateResp, err := client.UpdatePhilosophy(ctx, connect.NewRequest(updateReq))
	require.NoError(t, err)
	require.NotNil(t, updateResp.Msg.Philosophy)
	assert.Equal(t, updateReq.Description, updateResp.Msg.Philosophy.Description)
	assert.Equal(t, updateReq.ExtrapolateContexts, updateResp.Msg.Philosophy.ExtrapolateContexts)
}

func TestAddPhilosophyToSelfModel(t *testing.T) {
	resetStore()
	ctx := contextWithAPIKey(context.Background(), apiKey)

	// Create self model
	selfModelID := "test-philosophy-user"
	createSelfModelReq := &pb.CreateSelfModelRequest{
		Id:           selfModelID,
		Philosophies: []string{},
	}
	_, err := client.CreateSelfModel(ctx, connect.NewRequest(createSelfModelReq))
	require.NoError(t, err)

	// Create philosophy
	createPhilosophyReq := &pb.CreatePhilosophyRequest{
		Description:         "# Test Philosophy",
		ExtrapolateContexts: false,
	}
	createPhilosophyResp, err := client.CreatePhilosophy(ctx, connect.NewRequest(createPhilosophyReq))
	require.NoError(t, err)
	philosophyID := createPhilosophyResp.Msg.Philosophy.Id

	// Add philosophy to self model
	addReq := &pb.AddPhilosophyRequest{
		SelfModelId:  selfModelID,
		PhilosophyId: philosophyID,
	}
	addResp, err := client.AddPhilosophy(ctx, connect.NewRequest(addReq))
	require.NoError(t, err)
	require.NotNil(t, addResp.Msg.UpdatedSelfModel)
	assert.Contains(t, addResp.Msg.UpdatedSelfModel.Philosophies, philosophyID)
}

func TestUpdateNonExistentPhilosophy(t *testing.T) {
	resetStore()
	ctx := contextWithAPIKey(context.Background(), apiKey)

	updateReq := &pb.UpdatePhilosophyRequest{
		PhilosophyId:        "non-existent-id",
		Description:         "desc",
		ExtrapolateContexts: true,
	}
	_, err := client.UpdatePhilosophy(ctx, connect.NewRequest(updateReq))
	require.Error(t, err)
}
