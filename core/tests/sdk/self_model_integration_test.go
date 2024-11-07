package integration

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"epistemic-me-core/db"
	pb "epistemic-me-core/pb"
	"epistemic-me-core/pb/pbconnect"
	"epistemic-me-core/server"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

var (
	kvStore *db.KeyValueStore
	client  pbconnect.EpistemicMeServiceClient
	port    string
	apiKey  string
	srv     *http.Server
	wg      *sync.WaitGroup
)

// Add this interceptor function
func withAPIKey(apiKey string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			// Set the API key in the request headers
			req.Header().Set("x-api-key", apiKey)
			req.Header().Set("Origin", "http://localhost:8081")

			return next(ctx, req)
		}
	}
}

func TestMain(m *testing.M) {
	// Setup
	tempDir, err := os.MkdirTemp("", "test_kv_store")
	if err != nil {
		log.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	kvStorePath := filepath.Join(tempDir, "test_kv_store.json")
	kvStore, err = db.NewKeyValueStore(kvStorePath)
	if err != nil {
		log.Fatalf("Failed to create KeyValueStore: %v", err)
	}

	// Start the server using RunServer from server.go with dynamic port
	srv, wg, port = server.RunServer(kvStore, "") // RunServer returns 3 values

	// First create a temporary client without API key
	tempClient := pbconnect.NewEpistemicMeServiceClient(
		http.DefaultClient,
		fmt.Sprintf("http://localhost:%s", port), // Use the port here too
	)

	// Create a developer and get the API key
	ctx := context.Background()
	createReq := connect.NewRequest(&pb.CreateDeveloperRequest{
		Name:  "Test Developer",
		Email: "test@example.com",
	})

	resp, err := tempClient.CreateDeveloper(ctx, createReq)

	// Get the API key from the response
	apiKey = resp.Msg.Developer.ApiKeys[0]

	// Now create the real client with the API key interceptor
	client = pbconnect.NewEpistemicMeServiceClient(
		http.DefaultClient,
		fmt.Sprintf("http://localhost:%s", port),
		connect.WithInterceptors(withAPIKey(apiKey)),
	)

	// Run tests
	code := m.Run()

	// Shutdown the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Error shutting down server: %v", err)
	}
	wg.Wait()

	// Cleanup
	kvStore.ClearStore()

	os.Exit(code)
}

func createTestDeveloper() string {
	ctx := context.Background()
	req := &pb.CreateDeveloperRequest{
		Name:  "Test Developer",
		Email: "test@example.com",
	}
	resp, err := client.CreateDeveloper(ctx, connect.NewRequest(req))
	if err != nil {
		log.Fatalf("Failed to create test developer: %v", err)
	}
	return resp.Msg.Developer.ApiKeys[0]
}

func resetStore() {
	if kvStore != nil {
		kvStore.ClearStore()
	} else {
		log.Println("Warning: kvStore is nil in resetStore")
	}
}

func TestSelfModelIntegration(t *testing.T) {
	resetStore()
	TestCreateSelfModel(t)
	TestGetSelfModel(t)
	TestGetBeliefSystemOfSelfModel(t)
	TestListDialecticsOfSelfModel(t)
	TestAPIKeyValidation(t)
}

// Update contextWithAPIKey to be consistent
func contextWithAPIKey(ctx context.Context, apiKey string) context.Context {
	return ctx // No need to modify context since we're using interceptor
}

func TestCreateSelfModel(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New().String()

	// Add API key to outgoing context metadata
	ctx = metadata.AppendToOutgoingContext(ctx, "X-API-Key", apiKey)

	req := connect.NewRequest(&pb.CreateSelfModelRequest{
		Id:           userID,
		Philosophies: []string{"default"},
	})

	resp, err := client.CreateSelfModel(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp.Msg)
	assert.Equal(t, userID, resp.Msg.SelfModel.Id)
	assert.Equal(t, []string{"default"}, resp.Msg.SelfModel.Philosophies)
	testLogf(t, "CreateSelfModel response: %+v", resp.Msg)
}

func TestGetSelfModel(t *testing.T) {
	ctx := contextWithAPIKey(context.Background(), apiKey)
	userID := uuid.New().String()

	// First, create a self model
	_, err := client.CreateSelfModel(
		ctx,
		connect.NewRequest(&pb.CreateSelfModelRequest{
			Id:           userID,
			Philosophies: []string{"default"},
		}),
	)
	require.NoError(t, err)

	// Wait a short time to ensure the self model is stored
	time.Sleep(100 * time.Millisecond)

	// Now, get the self model
	getReq := &pb.GetSelfModelRequest{
		SelfModelId: userID,
	}
	getResp, err := client.GetSelfModel(
		ctx,
		connect.NewRequest(getReq),
	)
	require.NoError(t, err)
	assert.NotNil(t, getResp.Msg.SelfModel)
	assert.Equal(t, userID, getResp.Msg.SelfModel.Id)
	assert.Equal(t, []string{"default"}, getResp.Msg.SelfModel.Philosophies)
	testLogf(t, "GetSelfModel response: %+v", getResp.Msg)
}

func TestGetBeliefSystemOfSelfModel(t *testing.T) {
	ctx := contextWithAPIKey(context.Background(), apiKey)
	userID := uuid.New().String()

	// First, create a self model
	_, err := client.CreateSelfModel(
		ctx,
		connect.NewRequest(&pb.CreateSelfModelRequest{
			Id:           userID,
			Philosophies: []string{"default"},
		}),
	)
	require.NoError(t, err)

	// Wait a short time to ensure the self model is stored
	time.Sleep(100 * time.Millisecond)

	// Get the belief system of the self model
	getReq := &pb.GetSelfModelRequest{
		SelfModelId: userID,
	}
	getResp, err := client.GetSelfModel(
		ctx,
		connect.NewRequest(getReq),
	)
	require.NoError(t, err)
	assert.NotNil(t, getResp.Msg.SelfModel)
	assert.NotNil(t, getResp.Msg.SelfModel.BeliefSystem)

	beliefSystem := getResp.Msg.SelfModel.BeliefSystem
	assert.Empty(t, beliefSystem.Beliefs, "Newly created self model should have no beliefs")
	assert.Empty(t, beliefSystem.ObservationContexts, "Newly created self model should have no observation contexts")

	testLogf(t, "GetBeliefSystemOfSelfModel response: %+v", beliefSystem)
}

func TestListDialecticsOfSelfModel(t *testing.T) {
	ctx := contextWithAPIKey(context.Background(), apiKey)

	// First, create a self model
	createSelfModelReq := &pb.CreateSelfModelRequest{
		Id:           uuid.New().String(),
		Philosophies: []string{"default"},
	}
	createSelfModelResp, err := client.CreateSelfModel(
		ctx,
		connect.NewRequest(createSelfModelReq),
	)
	require.NoError(t, err)
	require.NotNil(t, createSelfModelResp.Msg.SelfModel)
	selfModelID := createSelfModelResp.Msg.SelfModel.Id

	// Wait a short time to ensure the self model is stored
	time.Sleep(100 * time.Millisecond)

	// Create a dialectic for the self model
	createDialecticReq := &pb.CreateDialecticRequest{
		SelfModelId: selfModelID, // Use the self-model ID as the user ID
	}
	createDialecticResp, err := client.CreateDialectic(
		ctx,
		connect.NewRequest(createDialecticReq),
	)
	require.NoError(t, err)
	require.NotEmpty(t, createDialecticResp.Msg.Dialectic.Id)

	// Wait a short time to ensure the dialectic is stored
	time.Sleep(100 * time.Millisecond)

	// Get the self model to list its dialectics
	getSelfModelReq := &pb.GetSelfModelRequest{
		SelfModelId: selfModelID,
	}
	getSelfModelResp, err := client.GetSelfModel(
		ctx,
		connect.NewRequest(getSelfModelReq),
	)
	require.NoError(t, err)
	require.NotNil(t, getSelfModelResp.Msg.SelfModel)

	// Check if the dialectic is in the SelfModel's dialectics
	assert.NotEmpty(t, getSelfModelResp.Msg.SelfModel.Dialectics, "Self model should have at least one dialectic")
	assert.Equal(t, 1, len(getSelfModelResp.Msg.SelfModel.Dialectics), "Self model should have exactly one dialectic")
	assert.Equal(t, createDialecticResp.Msg.Dialectic.Id, getSelfModelResp.Msg.SelfModel.Dialectics[0].Id, "Dialectic ID should match")

	testLogf(t, "ListDialecticsOfSelfModel response: %+v", getSelfModelResp.Msg.SelfModel.Dialectics)
}

// Update TestAPIKeyValidation to use the existing port variable
func TestAPIKeyValidation(t *testing.T) {
	// Verify port is set
	require.NotEmpty(t, port, "Port should be set by TestMain")

	// Test with valid API key first
	resp, err := createSelfModel(t)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.Msg.SelfModel.Id)
	serverURL := fmt.Sprintf("http://localhost:%s", port)

	// Create a new client with an invalid API key
	invalidClient := pbconnect.NewEpistemicMeServiceClient(
		http.DefaultClient,
		serverURL,
		connect.WithInterceptors(withAPIKey("invalid-api-key")),
	)

	// Test with invalid API key
	invalidReq := connect.NewRequest(&pb.CreateSelfModelRequest{
		Id:           uuid.New().String(),
		Philosophies: []string{"default"},
	})

	_, err = invalidClient.CreateSelfModel(context.Background(), invalidReq)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid API key format")
}

// Helper function to log test output when in verbose mode
func testLogf(t *testing.T, format string, v ...interface{}) {
	if testing.Verbose() {
		t.Logf(format, v...)
	}
}

// Add this helper function
func createSelfModel(t *testing.T) (*connect.Response[pb.CreateSelfModelResponse], error) {
	ctx := context.Background()
	userID := uuid.New().String()

	req := connect.NewRequest(&pb.CreateSelfModelRequest{
		Id:           userID,
		Philosophies: []string{"default"},
	})

	return client.CreateSelfModel(ctx, req)
}
