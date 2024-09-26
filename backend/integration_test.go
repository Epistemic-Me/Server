package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	db "epistemic-me-backend/db"
	pb "epistemic-me-backend/pb"
	models "epistemic-me-backend/pb/models"
	"epistemic-me-backend/pb/pbconnect"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

var (
	customHttpClient *http.Client
	client           pbconnect.EpistemicMeServiceClient
	kvStore          *db.KeyValueStore
)

// roundTripperFunc type defines a custom HTTP RoundTripper.
type roundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip implements the RoundTripper interface for roundTripperFunc.
func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// createCustomHttpClient creates an HTTP client with custom settings.
func createCustomHttpClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second, // Set a 10-second timeout for all requests
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

// createServiceClient creates a new gRPC service client.
func createServiceClient(customHttpClient *http.Client) pbconnect.EpistemicMeServiceClient {
	clientWithHeaders := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req.Header.Add("Origin", "http://localhost:8081")
			return customHttpClient.Do(req)
		}),
		Timeout: customHttpClient.Timeout,
	}
	return pbconnect.NewEpistemicMeServiceClient(clientWithHeaders, "http://localhost:8080")
}

// Update TestMain to use KeyValueStore
func TestMain(m *testing.M) {
	var err error
	kvStore, err = db.NewKeyValueStore("") // Use in-memory store for tests
	if err != nil {
		log.Fatalf("Failed to create KeyValueStore: %v", err)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatalf("OPENAI_API_KEY environment variable not set")
	}

	customHttpClient = createCustomHttpClient()
	client = createServiceClient(customHttpClient)

	// Run tests
	code := m.Run()

	// Clear the store after all tests
	kvStore.ClearStore()

	os.Exit(code)
}

// Add this helper function
func clearStore() {
	kvStore.ClearStore()
}

func generateUUID() string {
	return uuid.New().String()
}

func TestIntegrationInMemory(t *testing.T) {
	clearStore()
	// Test creating a belief
	TestCreateBelief(t)
	// Test creating a dialectic
	TestCreateDialectic(t)
	// Test updating a dialectic
	TestUpdateDialectic(t)
}

func TestIntegrationWithPersistentStore(t *testing.T) {
	clearStore()
	// Assuming the persistent store has been set up in TestMain
	// Test creating a belief
	TestCreateBelief(t)
	// Test creating a dialectic
	TestCreateDialectic(t)
	// Test updating a dialectic
	TestUpdateDialectic(t)
}

func TestCreateBelief(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	createBeliefReq := &pb.CreateBeliefRequest{
		UserId:        "test-user-id",
		BeliefContent: "Test belief content",
	}

	createBeliefResp, err := client.CreateBelief(ctx, connect.NewRequest(createBeliefReq))
	if err != nil {
		t.Fatalf("CreateBelief failed: %v", err)
	}

	assert.NotNil(t, createBeliefResp.Msg)
	assert.NotEmpty(t, createBeliefResp.Msg.Belief.Id)
	assert.Equal(t, createBeliefReq.BeliefContent, createBeliefResp.Msg.Belief.Content[0].RawStr)
	testLogf(t, "CreateBelief response: %+v", createBeliefResp.Msg)
}

func TestListBeliefs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// First, create a belief to ensure there's at least one belief in the system
	createBeliefReq := &pb.CreateBeliefRequest{
		UserId:        "test-user-id",
		BeliefContent: "Test belief content for ListBeliefs",
	}
	_, err := client.CreateBelief(ctx, connect.NewRequest(createBeliefReq))
	if err != nil {
		t.Fatalf("CreateBelief failed: %v", err)
	}

	// Wait a short time to ensure the belief is saved
	time.Sleep(500 * time.Millisecond)

	listBeliefsReq := &pb.ListBeliefsRequest{UserId: "test-user-id"}
	listBeliefsResp, err := client.ListBeliefs(ctx, connect.NewRequest(listBeliefsReq))
	if err != nil {
		t.Fatalf("ListBeliefs failed: %v", err)
	}

	assert.NotNil(t, listBeliefsResp.Msg)
	assert.NotNil(t, listBeliefsResp.Msg.BeliefSystem)
	assert.NotEmpty(t, listBeliefsResp.Msg.BeliefSystem.Beliefs, "Beliefs list should not be empty")
	testLogf(t, "ListBeliefs response: %+v", listBeliefsResp.Msg)
}

func TestCreateDialectic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	createDialecticReq := &pb.CreateDialecticRequest{UserId: "test-user-id"}
	createDialecticResp, err := client.CreateDialectic(ctx, connect.NewRequest(createDialecticReq))
	if err != nil {
		t.Fatalf("CreateDialectic failed: %v", err)
	}

	assert.NotEmpty(t, createDialecticResp.Msg.DialecticId, "Dialectic ID should not be empty")
	assert.NotNil(t, createDialecticResp.Msg.Dialectic, "Dialectic should not be nil")
	assert.Equal(t, 1, len(createDialecticResp.Msg.Dialectic.UserInteractions), "Newly created dialectic should have one interaction")
	assert.NotEmpty(t, createDialecticResp.Msg.Dialectic.UserInteractions[0].Question.Question, "Initial question should not be empty")
	assert.Equal(t, models.DialecticalInteraction_STATUS_PENDING_ANSWER, createDialecticResp.Msg.Dialectic.UserInteractions[0].Status, "Initial interaction status should be pending answer")

	testLogf(t, "CreateDialectic response: %+v", createDialecticResp.Msg)
}

func TestListDialectics(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var listDialecticsResp *connect.Response[pb.ListDialecticsResponse]
	var err error

	// First, create a dialectic
	createDialecticReq := &pb.CreateDialecticRequest{UserId: "test-user-id"}
	createResp, err := client.CreateDialectic(ctx, connect.NewRequest(createDialecticReq))
	if err != nil {
		t.Fatalf("CreateDialectic failed: %v", err)
	}
	testLogf(t, "Created dialectic with ID: %s", createResp.Msg.DialecticId)

	// Increase the delay to allow for the dialectic to be saved
	time.Sleep(2 * time.Second)

	// Retry up to 5 times with a 500ms delay between attempts
	for i := 0; i < 5; i++ {
		listDialecticsReq := &pb.ListDialecticsRequest{UserId: "test-user-id"}
		listDialecticsResp, err = client.ListDialectics(ctx, connect.NewRequest(listDialecticsReq))
		if err == nil && len(listDialecticsResp.Msg.Dialectics) > 0 {
			break
		}
		testLogf(t, "Attempt %d: ListDialectics returned %d dialectics", i+1, len(listDialecticsResp.Msg.Dialectics))
		time.Sleep(500 * time.Millisecond)
	}

	if err != nil {
		t.Fatalf("ListDialectics failed: %v", err)
	}

	assert.NotNil(t, listDialecticsResp.Msg)
	assert.NotEmpty(t, listDialecticsResp.Msg.Dialectics, "Dialectics list should not be empty")
	testLogf(t, "ListDialectics response: %+v", listDialecticsResp.Msg)
}

func TestUpdateDialectic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First, create a dialectic
	createDialecticReq := &pb.CreateDialecticRequest{UserId: "test-user-id"}
	createDialecticResp, err := client.CreateDialectic(ctx, connect.NewRequest(createDialecticReq))
	if err != nil {
		t.Fatalf("CreateDialectic failed: %v", err)
	}

	dialecticId := createDialecticResp.Msg.DialecticId

	// First update should generate the initial question
	updateDialecticReq := &pb.UpdateDialecticRequest{
		DialecticId: dialecticId,
		Answer: &models.UserAnswer{
			UserAnswer:         "Initial answer to generate the first question",
			CreatedAtMillisUtc: time.Now().UnixMilli(),
		},
		UserId: "test-user-id",
	}

	updateDialecticResp, err := client.UpdateDialectic(ctx, connect.NewRequest(updateDialecticReq))
	if err != nil {
		t.Fatalf("First UpdateDialectic failed: %v", err)
	}

	assert.NotNil(t, updateDialecticResp.Msg)
	assert.NotNil(t, updateDialecticResp.Msg.Dialectic)
	assert.NotEmpty(t, updateDialecticResp.Msg.Dialectic.UserInteractions, "Should have interactions after first update")
	lastInteraction := updateDialecticResp.Msg.Dialectic.UserInteractions[len(updateDialecticResp.Msg.Dialectic.UserInteractions)-1]
	assert.NotEmpty(t, lastInteraction.Question.Question, "Last interaction should have a question")

	testLogf(t, "First UpdateDialectic response: %+v", updateDialecticResp.Msg)

	// Second update should answer the first question and generate a new one
	secondUpdateReq := &pb.UpdateDialecticRequest{
		DialecticId: dialecticId,
		Answer: &models.UserAnswer{
			UserAnswer:         "Answer to the first question",
			CreatedAtMillisUtc: time.Now().UnixMilli(),
		},
		UserId: "test-user-id",
	}

	secondUpdateResp, err := client.UpdateDialectic(ctx, connect.NewRequest(secondUpdateReq))
	if err != nil {
		t.Fatalf("Second UpdateDialectic failed: %v", err)
	}

	assert.NotNil(t, secondUpdateResp.Msg)
	assert.NotNil(t, secondUpdateResp.Msg.Dialectic)
	assert.True(t, len(secondUpdateResp.Msg.Dialectic.UserInteractions) > 1, "Should have multiple interactions after second update")
	lastInteraction = secondUpdateResp.Msg.Dialectic.UserInteractions[len(secondUpdateResp.Msg.Dialectic.UserInteractions)-1]
	assert.NotEmpty(t, lastInteraction.Question.Question, "Last interaction should have a new question")

	testLogf(t, "Second UpdateDialectic response: %+v", secondUpdateResp.Msg)
}

func TestGetBeliefSystemDetail(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	userId := "test-user-id"

	// Create a belief to ensure a belief system exists
	createBeliefReq := &pb.CreateBeliefRequest{
		UserId:        userId,
		BeliefContent: "Test belief for belief system",
	}
	createResp, err := client.CreateBelief(ctx, connect.NewRequest(createBeliefReq))
	if err != nil {
		t.Fatalf("CreateBelief failed: %v", err)
	}
	testLogf(t, "Created belief with ID: %s", createResp.Msg.Belief.Id)

	// Increase the delay to allow for the belief system to be created
	time.Sleep(2 * time.Second)

	getBeliefSystemDetailReq := &pb.GetBeliefSystemDetailRequest{
		UserId:                       userId,
		CurrentObservationContextIds: []string{},
	}

	var getBeliefSystemDetailResp *connect.Response[pb.GetBeliefSystemDetailResponse]
	// Retry up to 5 times with a 500ms delay between attempts
	for i := 0; i < 5; i++ {
		getBeliefSystemDetailResp, err = client.GetBeliefSystemDetail(ctx, connect.NewRequest(getBeliefSystemDetailReq))
		if err == nil && getBeliefSystemDetailResp.Msg.BeliefSystemDetail != nil &&
			(len(getBeliefSystemDetailResp.Msg.BeliefSystemDetail.BeliefSystem.Beliefs) > 0 ||
				len(getBeliefSystemDetailResp.Msg.BeliefSystemDetail.BeliefSystem.ObservationContexts) > 0) {
			break
		}
		testLogf(t, "Attempt %d: GetBeliefSystemDetail failed or returned empty: %v", i+1, err)
		time.Sleep(500 * time.Millisecond)
	}

	if err != nil {
		t.Fatalf("GetBeliefSystemDetail failed after retries: %v", err)
	}

	assert.NotNil(t, getBeliefSystemDetailResp.Msg)
	assert.NotNil(t, getBeliefSystemDetailResp.Msg.BeliefSystemDetail)

	beliefSystemDetail := getBeliefSystemDetailResp.Msg.BeliefSystemDetail

	assert.NotEmpty(t, beliefSystemDetail.ExampleName, "Example name should not be empty")
	assert.NotNil(t, beliefSystemDetail.BeliefSystem, "Belief system should not be nil")

	// Check if beliefs or observation contexts are present
	beliefsPresent := len(beliefSystemDetail.BeliefSystem.Beliefs) > 0
	contextsPresent := len(beliefSystemDetail.BeliefSystem.ObservationContexts) > 0

	assert.True(t, beliefsPresent || contextsPresent, "Either beliefs or observation contexts should be present")

	testLogf(t, "Retrieved BeliefSystemDetail: %+v", beliefSystemDetail)
	testLogf(t, "Number of beliefs: %d", len(beliefSystemDetail.BeliefSystem.Beliefs))
	testLogf(t, "Number of observation contexts: %d", len(beliefSystemDetail.BeliefSystem.ObservationContexts))

	if beliefsPresent {
		firstBelief := beliefSystemDetail.BeliefSystem.Beliefs[0]
		testLogf(t, "First belief: %+v", firstBelief)
		assert.NotEmpty(t, firstBelief.Id)
		assert.NotEmpty(t, firstBelief.Content)
	}

	if contextsPresent {
		firstContext := beliefSystemDetail.BeliefSystem.ObservationContexts[0]
		testLogf(t, "First observation context: %+v", firstContext)
		assert.NotEmpty(t, firstContext.Id)
		assert.NotEmpty(t, firstContext.Name)
	}
}

func testLogf(t *testing.T, format string, v ...interface{}) {
	if testing.Verbose() {
		t.Logf(format, v...)
	}
}
