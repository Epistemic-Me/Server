package integration

import (
	"context"
	"testing"

	pb "epistemic-me-backend/pb"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeveloperUserIntegration(t *testing.T) {
	resetStore()
	TestCreateDeveloper(t)
	TestCreateUser(t)
}

func TestCreateDeveloper(t *testing.T) {
	ctx := context.Background()
	name := "Test Developer"
	email := "test@developer.com"

	req := &pb.CreateDeveloperRequest{
		Name:  name,
		Email: email,
	}

	resp, err := client.CreateDeveloper(ctx, connect.NewRequest(req))
	require.NoError(t, err)
	assert.NotNil(t, resp.Msg)
	assert.NotEmpty(t, resp.Msg.Developer.Id)
	assert.Equal(t, name, resp.Msg.Developer.Name)
	assert.Equal(t, email, resp.Msg.Developer.Email)
	assert.NotEmpty(t, resp.Msg.Developer.ApiKeys)
	assert.NotZero(t, resp.Msg.Developer.CreatedAt)
	assert.NotZero(t, resp.Msg.Developer.UpdatedAt)

	testLogf(t, "CreateDeveloper response: %+v", resp.Msg)

	// Store the developer ID for the User test
	developerID = resp.Msg.Developer.Id
}

func TestCreateUser(t *testing.T) {
	// Skip this test if developer creation failed
	if developerID == "" {
		t.Skip("Skipping user creation test because developer creation failed")
	}

	ctx := context.Background()
	name := "Test User"
	email := "test@user.com"

	req := &pb.CreateUserRequest{
		DeveloperId: developerID,
		Name:        name,
		Email:       email,
	}

	resp, err := client.CreateUser(ctx, connect.NewRequest(req))
	require.NoError(t, err)
	assert.NotNil(t, resp.Msg)
	assert.NotEmpty(t, resp.Msg.User.Id)
	assert.Equal(t, developerID, resp.Msg.User.DeveloperId)
	assert.Equal(t, name, resp.Msg.User.Name)
	assert.Equal(t, email, resp.Msg.User.Email)
	assert.NotZero(t, resp.Msg.User.CreatedAt)
	assert.NotZero(t, resp.Msg.User.UpdatedAt)

	testLogf(t, "CreateUser response: %+v", resp.Msg)
}

var developerID string

func init() {
	// This ensures that the TestMain in self_model_integration_test.go is run
	// and the client is properly initialized before our tests run.
}
