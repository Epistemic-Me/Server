package integration

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"epistemic-me-backend/db"
	pb "epistemic-me-backend/pb"
	"epistemic-me-backend/pb/pbconnect"
	"epistemic-me-backend/server"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	kvStore *db.KeyValueStore
	client  pbconnect.EpistemicMeServiceClient
	port    string
)

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
	srv, wg, port := server.RunServer(kvStore, "")

	// Create a client for the EpistemicMeService
	client = pbconnect.NewEpistemicMeServiceClient(http.DefaultClient, "http://localhost:"+port)

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

func resetStore() {
	if kvStore != nil {
		kvStore.ClearStore()
	} else {
		log.Println("Warning: kvStore is nil in resetStore")
	}
}

func TestSelfAgentIntegration(t *testing.T) {
	resetStore()
	TestCreateSelfAgent(t)
	TestGetSelfAgent(t)
	TestCreatePhilosophy(t)
	TestAddPhilosophy(t)
	TestGetBeliefSystemOfSelfAgent(t)
	TestListDialecticsOfSelfAgent(t)
}

func TestCreateSelfAgent(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New().String()
	req := &pb.CreateSelfAgentRequest{
		Id:           userID,
		Philosophies: []string{"default"},
	}
	resp, err := client.CreateSelfAgent(ctx, connect.NewRequest(req))
	require.NoError(t, err)
	assert.NotNil(t, resp.Msg)
	assert.Equal(t, userID, resp.Msg.SelfAgent.Id)
	assert.Equal(t, []string{"default"}, resp.Msg.SelfAgent.Philosophies)
	testLogf(t, "CreateSelfAgent response: %+v", resp.Msg)
}

func TestGetSelfAgent(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New().String()

	// First, create a self agent
	_, err := client.CreateSelfAgent(ctx, connect.NewRequest(&pb.CreateSelfAgentRequest{
		Id:           userID,
		Philosophies: []string{"default"},
	}))
	require.NoError(t, err)

	// Wait a short time to ensure the self agent is stored
	time.Sleep(100 * time.Millisecond)

	// Now, get the self agent
	getReq := &pb.GetSelfAgentRequest{
		SelfAgentId: userID,
	}
	getResp, err := client.GetSelfAgent(ctx, connect.NewRequest(getReq))
	require.NoError(t, err)
	assert.NotNil(t, getResp.Msg.SelfAgent)
	assert.Equal(t, userID, getResp.Msg.SelfAgent.Id)
	assert.Equal(t, []string{"default"}, getResp.Msg.SelfAgent.Philosophies)
	testLogf(t, "GetSelfAgent response: %+v", getResp.Msg)
}

func TestCreatePhilosophy(t *testing.T) {
	ctx := context.Background()
	req := &pb.CreatePhilosophyRequest{
		Description:         "Test philosophy",
		ExtrapolateContexts: true,
	}
	resp, err := client.CreatePhilosophy(ctx, connect.NewRequest(req))
	require.NoError(t, err)
	assert.NotNil(t, resp.Msg.Philosophy)
	assert.NotEmpty(t, resp.Msg.Philosophy.Id)
	assert.Equal(t, "Test philosophy", resp.Msg.Philosophy.Description)
	assert.True(t, resp.Msg.Philosophy.ExtrapolateContexts)
	testLogf(t, "CreatePhilosophy response: %+v", resp.Msg)
}

func TestAddPhilosophy(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New().String()

	// First, create a self agent
	_, err := client.CreateSelfAgent(ctx, connect.NewRequest(&pb.CreateSelfAgentRequest{
		Id:           userID,
		Philosophies: []string{"default"},
	}))
	require.NoError(t, err)

	// Create a new philosophy
	createPhilosophyResp, err := client.CreatePhilosophy(ctx, connect.NewRequest(&pb.CreatePhilosophyRequest{
		Description:         "New philosophy",
		ExtrapolateContexts: false,
	}))
	require.NoError(t, err)
	philosophyID := createPhilosophyResp.Msg.Philosophy.Id

	// Wait a short time to ensure the philosophy is stored
	time.Sleep(100 * time.Millisecond)

	// Add the new philosophy to the self agent
	addReq := &pb.AddPhilosophyRequest{
		SelfAgentId:  userID,
		PhilosophyId: philosophyID,
	}
	addResp, err := client.AddPhilosophy(ctx, connect.NewRequest(addReq))
	require.NoError(t, err)
	assert.NotNil(t, addResp.Msg.UpdatedSelfAgent)
	assert.Contains(t, addResp.Msg.UpdatedSelfAgent.Philosophies, philosophyID)
	testLogf(t, "AddPhilosophy response: %+v", addResp.Msg)
}

func TestGetBeliefSystemOfSelfAgent(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New().String()

	// First, create a self agent
	_, err := client.CreateSelfAgent(ctx, connect.NewRequest(&pb.CreateSelfAgentRequest{
		Id:           userID,
		Philosophies: []string{"default"},
	}))
	require.NoError(t, err)

	// Wait a short time to ensure the self agent is stored
	time.Sleep(100 * time.Millisecond)

	// Get the belief system of the self agent
	getReq := &pb.GetSelfAgentRequest{
		SelfAgentId: userID,
	}
	getResp, err := client.GetSelfAgent(ctx, connect.NewRequest(getReq))
	require.NoError(t, err)
	assert.NotNil(t, getResp.Msg.SelfAgent)
	assert.NotNil(t, getResp.Msg.SelfAgent.BeliefSystem)

	beliefSystem := getResp.Msg.SelfAgent.BeliefSystem
	assert.Empty(t, beliefSystem.Beliefs, "Newly created self agent should have no beliefs")
	assert.Empty(t, beliefSystem.ObservationContexts, "Newly created self agent should have no observation contexts")

	testLogf(t, "GetBeliefSystemOfSelfAgent response: %+v", beliefSystem)
}

func TestListDialecticsOfSelfAgent(t *testing.T) {
	ctx := context.Background()

	// First, create a self agent
	createSelfAgentReq := &pb.CreateSelfAgentRequest{
		Id:           uuid.New().String(),
		Philosophies: []string{"default"},
	}
	createSelfAgentResp, err := client.CreateSelfAgent(ctx, connect.NewRequest(createSelfAgentReq))
	require.NoError(t, err)
	require.NotNil(t, createSelfAgentResp.Msg.SelfAgent)
	selfAgentID := createSelfAgentResp.Msg.SelfAgent.Id

	// Wait a short time to ensure the self agent is stored
	time.Sleep(100 * time.Millisecond)

	// Create a dialectic for the self agent
	createDialecticReq := &pb.CreateDialecticRequest{
		UserId: selfAgentID, // Use the self-agent ID as the user ID
	}
	createDialecticResp, err := client.CreateDialectic(ctx, connect.NewRequest(createDialecticReq))
	require.NoError(t, err)
	require.NotEmpty(t, createDialecticResp.Msg.DialecticId)

	// Wait a short time to ensure the dialectic is stored
	time.Sleep(100 * time.Millisecond)

	// Get the self agent to list its dialectics
	getSelfAgentReq := &pb.GetSelfAgentRequest{
		SelfAgentId: selfAgentID,
	}
	getSelfAgentResp, err := client.GetSelfAgent(ctx, connect.NewRequest(getSelfAgentReq))
	require.NoError(t, err)
	require.NotNil(t, getSelfAgentResp.Msg.SelfAgent)

	// Check if the dialectic is in the SelfAgent's dialectics
	assert.NotEmpty(t, getSelfAgentResp.Msg.SelfAgent.Dialectics, "Self agent should have at least one dialectic")
	assert.Equal(t, 1, len(getSelfAgentResp.Msg.SelfAgent.Dialectics), "Self agent should have exactly one dialectic")
	assert.Equal(t, createDialecticResp.Msg.DialecticId, getSelfAgentResp.Msg.SelfAgent.Dialectics[0].Id, "Dialectic ID should match")

	testLogf(t, "ListDialecticsOfSelfAgent response: %+v", getSelfAgentResp.Msg.SelfAgent.Dialectics)
}

// Helper function to log test output when in verbose mode
func testLogf(t *testing.T, format string, v ...interface{}) {
	if testing.Verbose() {
		t.Logf(format, v...)
	}
}
