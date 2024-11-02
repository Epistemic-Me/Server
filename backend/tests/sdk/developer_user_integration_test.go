package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	pb "epistemic-me-backend/pb"
	"epistemic-me-backend/pb/pbconnect"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var developerID string

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

	developerID = resp.Msg.Developer.Id
	apiKey = resp.Msg.Developer.ApiKeys[0]

	// Update client with new API key
	client = pbconnect.NewEpistemicMeServiceClient(
		http.DefaultClient,
		fmt.Sprintf("http://localhost:%s", port),
		connect.WithInterceptors(withAPIKey(apiKey)),
	)
}

func TestCreateUser(t *testing.T) {
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
}

func init() {
	// This ensures that the TestMain in self_model_integration_test.go is run
	// and the client is properly initialized before our tests run.
}
