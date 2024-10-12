package integration

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	db "epistemic-me-backend/db"
	fixture_models "epistemic-me-backend/db/fixtures"
	pb "epistemic-me-backend/pb"
	models "epistemic-me-backend/pb/models"
	"epistemic-me-backend/pb/pbconnect"
	"epistemic-me-backend/server"
	svc_models "epistemic-me-backend/svc/models"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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

	// Import fixtures
	err = fixture_models.ImportFixtures(kvStore)
	if err != nil {
		log.Fatalf("Failed to import fixtures: %v", err)
	}

	// Create initial belief system for test user
	err = CreateInitialBeliefSystemIfNotExists("test-user-id")
	if err != nil {
		log.Fatalf("Failed to create initial belief system in TestMain: %v", err)
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

// Add this helper function
func clearStore() {
	kvStore.ClearStore()
}

func resetStore() {
	kvStore.ClearStore()
	// Re-import fixtures if necessary
	err := fixture_models.ImportFixtures(kvStore)
	if err != nil {
		log.Fatalf("Failed to import fixtures: %v", err)
	}
}

func generateUUID() string {
	return uuid.New().String()
}

func TestIntegrationInMemory(t *testing.T) {
	resetStore()
	// Test creating a belief
	TestCreateBelief(t)
	// Test creating a dialectic
	TestCreateDialectic(t)
	// Test updating a dialectic
	TestUpdateDialectic(t)
}

func TestIntegrationWithPersistentStore(t *testing.T) {
	resetStore()
	// Assuming the persistent store has been set up in TestMain
	// Test creating a belief
	TestCreateBelief(t)
	// Test creating a dialectic
	TestCreateDialectic(t)
	// Test updating a dialectic
	TestUpdateDialectic(t)
}

func TestCreateBelief(t *testing.T) {
	// Use the global client variable
	resp, err := client.CreateBelief(context.Background(), connect.NewRequest(&pb.CreateBeliefRequest{
		UserId:        "test-user-id",
		BeliefContent: "Test belief content",
	}))
	if err != nil {
		t.Fatalf("CreateBelief failed: %v", err)
	}

	assert.NotNil(t, resp.Msg)
	assert.NotEmpty(t, resp.Msg.Belief.Id)
	assert.Equal(t, "Test belief content", resp.Msg.Belief.Content[0].RawStr)
	testLogf(t, "CreateBelief response: %+v", resp.Msg)
}

func TestListBeliefs(t *testing.T) {
	clearStore()
	ctx := context.Background()
	userId := "test-user-id"

	// Create initial belief system
	err := CreateInitialBeliefSystemIfNotExists(userId)
	require.NoError(t, err)

	// Create a belief
	createReq := &pb.CreateBeliefRequest{
		UserId:        userId,
		BeliefContent: "Test belief content for ListBeliefs",
	}
	createResp, err := client.CreateBelief(ctx, connect.NewRequest(createReq))
	require.NoError(t, err)
	require.NotNil(t, createResp)

	// Wait a short time to ensure the belief is stored
	time.Sleep(100 * time.Millisecond)

	// List beliefs
	listReq := &pb.ListBeliefsRequest{
		UserId: userId,
	}
	listResp, err := client.ListBeliefs(ctx, connect.NewRequest(listReq))
	require.NoError(t, err)
	require.NotNil(t, listResp)

	beliefs := listResp.Msg.Beliefs
	require.NotEmpty(t, beliefs, "Beliefs list should not be empty")
	require.Equal(t, 1, len(beliefs), "Should have 1 belief")
	assert.Equal(t, createResp.Msg.Belief.Id, beliefs[0].Id, "Belief ID should match")
	assert.Equal(t, "Test belief content for ListBeliefs", beliefs[0].Content[0].RawStr, "Belief content should match")
}

func TestCreateDialectic(t *testing.T) {
	userId := "test-user-id"
	err := CreateInitialBeliefSystemIfNotExists(userId)
	if err != nil {
		t.Fatalf("Failed to create initial belief system: %v", err)
	}

	createResp, err := client.CreateDialectic(context.Background(), connect.NewRequest(&pb.CreateDialecticRequest{
		UserId: userId,
	}))
	if err != nil {
		t.Fatalf("CreateDialectic failed: %v", err)
	}

	assert.NotNil(t, createResp.Msg)
	assert.NotEmpty(t, createResp.Msg.DialecticId, "Dialectic ID should not be empty")
}

func TestListDialectics(t *testing.T) {
	userId := "test-user-id"
	err := CreateInitialBeliefSystemIfNotExists(userId)
	if err != nil {
		t.Fatalf("Failed to create initial belief system: %v", err)
	}

	// Create a dialectic
	_, err = client.CreateDialectic(context.Background(), connect.NewRequest(&pb.CreateDialecticRequest{
		UserId: userId,
	}))
	if err != nil {
		t.Fatalf("CreateDialectic failed: %v", err)
	}

	// List dialectics
	listResp, err := client.ListDialectics(context.Background(), connect.NewRequest(&pb.ListDialecticsRequest{
		UserId: userId,
	}))
	if err != nil {
		t.Fatalf("ListDialectics failed: %v", err)
	}

	assert.NotNil(t, listResp.Msg)
	assert.NotEmpty(t, listResp.Msg.Dialectics, "Dialectics list should not be empty")
}

func TestUpdateDialectic(t *testing.T) {
	userId := "test-user-id"
	err := CreateInitialBeliefSystemIfNotExists(userId)
	if err != nil {
		t.Fatalf("Failed to create initial belief system: %v", err)
	}

	// Create a dialectic
	createResp, err := client.CreateDialectic(context.Background(), connect.NewRequest(&pb.CreateDialecticRequest{
		UserId: userId,
	}))
	if err != nil {
		t.Fatalf("CreateDialectic failed: %v", err)
	}

	dialecticId := createResp.Msg.DialecticId

	// Update the dialectic
	updateResp, err := client.UpdateDialectic(context.Background(), connect.NewRequest(&pb.UpdateDialecticRequest{
		DialecticId: dialecticId,
		Answer: &models.UserAnswer{
			UserAnswer:         "Test answer",
			CreatedAtMillisUtc: time.Now().UnixMilli(),
		},
		UserId: userId,
	}))
	if err != nil {
		t.Fatalf("UpdateDialectic failed: %v", err)
	}

	assert.NotNil(t, updateResp.Msg)
	assert.NotNil(t, updateResp.Msg.Dialectic)
	assert.NotEmpty(t, updateResp.Msg.Dialectic.UserInteractions, "Should have interactions after update")
}

func TestGetBeliefSystemDetail(t *testing.T) {
	userId := "test-user-id"
	err := CreateInitialBeliefSystemIfNotExists(userId)
	if err != nil {
		t.Fatalf("Failed to create initial belief system: %v", err)
	}

	// Create a belief
	createResp, err := client.CreateBelief(context.Background(), connect.NewRequest(&pb.CreateBeliefRequest{
		UserId:        userId,
		BeliefContent: "Test belief for belief system",
	}))
	if err != nil {
		t.Fatalf("CreateBelief failed: %v", err)
	}
	t.Logf("Created belief with ID: %s", createResp.Msg.Belief.Id)

	// Add a small delay after creating the belief
	time.Sleep(100 * time.Millisecond)

	// Get belief system detail
	var getResp *connect.Response[pb.GetBeliefSystemDetailResponse]
	for i := 0; i < 5; i++ {
		getResp, err = client.GetBeliefSystemDetail(context.Background(), connect.NewRequest(&pb.GetBeliefSystemDetailRequest{
			UserId: userId,
		}))
		t.Logf("Attempt %d: Retrieved belief system with %d beliefs", i+1, len(getResp.Msg.BeliefSystemDetail.BeliefSystem.Beliefs))
		if err == nil && getResp.Msg.BeliefSystemDetail != nil && len(getResp.Msg.BeliefSystemDetail.BeliefSystem.Beliefs) > 0 {
			break
		}
		time.Sleep(time.Second)
	}

	if err != nil {
		t.Fatalf("GetBeliefSystemDetail failed: %v", err)
	}

	assert.NotNil(t, getResp.Msg.BeliefSystemDetail)
	assert.NotNil(t, getResp.Msg.BeliefSystemDetail.BeliefSystem)
	assert.NotEmpty(t, getResp.Msg.BeliefSystemDetail.BeliefSystem.Beliefs, "Beliefs should not be empty")

	t.Logf("Retrieved BeliefSystemDetail: %+v", getResp.Msg.BeliefSystemDetail)
	t.Logf("Number of beliefs: %d", len(getResp.Msg.BeliefSystemDetail.BeliefSystem.Beliefs))
	t.Logf("Number of observation contexts: %d", len(getResp.Msg.BeliefSystemDetail.BeliefSystem.ObservationContexts))

	// Check if the created belief is in the belief system
	foundCreatedBelief := false
	for _, belief := range getResp.Msg.BeliefSystemDetail.BeliefSystem.Beliefs {
		if belief.Id == createResp.Msg.Belief.Id {
			foundCreatedBelief = true
			break
		}
	}
	assert.True(t, foundCreatedBelief, "Created belief not found in the belief system")
}

func testLogf(t *testing.T, format string, v ...interface{}) {
	if testing.Verbose() {
		t.Logf(format, v...)
	}
}

func TestIntegrationWithFixtures(t *testing.T) {
	// Clear the store before running the test
	clearStore()

	// Import fixtures
	err := fixture_models.ImportFixtures(kvStore)
	if err != nil {
		t.Fatalf("Failed to import fixtures: %v", err)
	}

	// Read the fixture file to get the expected data
	yamlFile, err := os.ReadFile("../../db/fixtures/belief_system_fixture.yaml")
	if err != nil {
		t.Fatalf("Failed to read fixture file: %v", err)
	}

	var fixture fixture_models.BeliefSystemFixture
	err = yaml.Unmarshal(yamlFile, &fixture)
	if err != nil {
		t.Fatalf("Failed to unmarshal fixture: %v", err)
	}

	fixtureUserID := "fixture-user-id"

	// Retrieve the stored BeliefSystem
	storedBeliefSystem, err := kvStore.Retrieve(fixtureUserID, "BeliefSystemId")
	if err != nil {
		t.Fatalf("Failed to retrieve stored belief system: %v", err)
	}

	bs, ok := storedBeliefSystem.(*svc_models.BeliefSystem)
	if !ok {
		t.Fatalf("Stored belief system is not of the expected type. Got: %T", storedBeliefSystem)
	}

	// Verify the structure of the retrieved BeliefSystem
	assert.NotEmpty(t, bs.Beliefs, "BeliefSystem should have beliefs")
	assert.NotEmpty(t, bs.ObservationContexts, "BeliefSystem should have observation contexts")

	// Verify the number of beliefs and observation contexts
	assert.Equal(t, 12, len(bs.Beliefs), "Number of beliefs should match")
	assert.Equal(t, 16, len(bs.ObservationContexts), "Number of observation contexts should match")

	// Verify the content of beliefs
	for _, belief := range bs.Beliefs {
		assert.NotEmpty(t, belief.ID, "Belief ID should not be empty")
		assert.Equal(t, fixtureUserID, belief.UserID, "Belief UserID should match fixture user ID")
		assert.NotEmpty(t, belief.Content, "Belief Content should not be empty")
		assert.NotEmpty(t, belief.ObservationContextIDs, "Belief ObservationContextIDs should not be empty")
		assert.NotEmpty(t, belief.Probabilities, "Belief Probabilities should not be empty")
		assert.NotEmpty(t, belief.Result, "Belief Result should not be empty")
	}

	// Verify the content of observation contexts
	for _, context := range bs.ObservationContexts {
		assert.NotEmpty(t, context.ID, "ObservationContext ID should not be empty")
		assert.NotEmpty(t, context.Name, "ObservationContext Name should not be empty")
	}

	t.Logf("Successfully verified BeliefSystem with %d beliefs and %d observation contexts", len(bs.Beliefs), len(bs.ObservationContexts))

	// Clean up
	clearStore()
}

func CreateInitialBeliefSystemIfNotExists(userId string) error {
	bs, err := kvStore.Retrieve(userId, "BeliefSystemId")
	if err != nil || bs == nil {
		initialBS := svc_models.BeliefSystem{
			Beliefs:             []*svc_models.Belief{},
			ObservationContexts: []*svc_models.ObservationContext{},
		}

		log.Printf("Creating initial BeliefSystem: %+v", initialBS)
		err = kvStore.Store(userId, "BeliefSystemId", initialBS, 1)
		if err != nil {
			log.Printf("Error storing initial BeliefSystem: %v", err)
			return fmt.Errorf("failed to create initial belief system: %v", err)
		}
	}
	return nil
}
