package integration

import (
	"context"
	"testing"

	pb "epistemic-me-backend/pb"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
)

func TestDashboardAPIIntegration(t *testing.T) {
	userId := "test-user-id"
	err := CreateInitialBeliefSystemIfNotExists(userId) // Change this line
	if err != nil {
		t.Fatalf("Failed to create initial belief system: %v", err)
	}

	resp, err := client.ListBeliefs(context.Background(), connect.NewRequest(&pb.ListBeliefsRequest{
		UserId: userId,
	}))
	if err != nil {
		t.Fatalf("ListBeliefs failed: %v", err)
	}

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Msg)
	// The belief system is initially empty, so we expect an empty list
	assert.Empty(t, resp.Msg.Beliefs, "Expected an empty list of beliefs")
}
