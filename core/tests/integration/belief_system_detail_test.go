package integration

import (
	"context"
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
	"github.com/stretchr/testify/assert"
)

func TestBeliefSystemAPIIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	selfModelId := "fixture-self-model-id"

	// Initialize the KeyValueStore
	tempDir, err := os.MkdirTemp("", "test_kv_store_belief_system_detail")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	kvStorePath := filepath.Join(tempDir, "test_kv_store_belief_system_detail.json")
	kvStore, err := db.NewKeyValueStore(kvStorePath)
	if err != nil {
		t.Fatalf("Failed to create KeyValueStore: %v", err)
	}
	defer kvStore.ClearStore()

	// Import fixtures
	err = fixture_models.ImportFixtures(kvStore)
	if err != nil {
		t.Fatalf("Failed to import fixtures: %v", err)
	}

	// Wait for fixtures to be fully imported
	time.Sleep(2 * time.Second)

	// Verify that fixtures were imported correctly
	fixtureBeliefSystem, err := kvStore.Retrieve(selfModelId, "BeliefSystemId")
	if err != nil {
		t.Fatalf("Failed to retrieve fixture belief system: %v", err)
	}

	// Log the type of the retrieved object
	t.Logf("Retrieved object type: %T", fixtureBeliefSystem)

	// Try to cast to different possible types
	switch bs := fixtureBeliefSystem.(type) {
	case *models.BeliefSystem:
		t.Logf("Successfully cast to *models.BeliefSystem with %d beliefs and %d observation contexts", len(bs.Beliefs), len(bs.ObservationContexts))
	case *svc_models.BeliefSystem:
		t.Logf("Successfully cast to *svc_models.BeliefSystem with %d beliefs and %d observation contexts", len(bs.Beliefs), len(bs.ObservationContexts))
	default:
		t.Fatalf("Retrieved object is not a recognized BeliefSystem type: %T", fixtureBeliefSystem)
	}

	// Start the server with the test KeyValueStore
	srv, wg, port := server.RunServer(kvStore, "")
	defer func() {
		if err := srv.Shutdown(ctx); err != nil {
			t.Fatalf("Failed to shut down server: %v", err)
		}
		wg.Wait()
	}()

	// Create a client for this specific test
	testClient := pbconnect.NewEpistemicMeServiceClient(http.DefaultClient, "http://localhost:"+port)

	resp, err := testClient.GetBeliefSystem(context.Background(), connect.NewRequest(&pb.GetBeliefSystemRequest{
		SelfModelId: selfModelId,
	}))
	if err != nil {
		t.Fatalf("GetBeliefSystem failed: %v", err)
	}

	beliefSystem := resp.Msg.BeliefSystem
	assert.NotNil(t, beliefSystem)
	assert.NotEmpty(t, beliefSystem.Beliefs, "Beliefs should not be empty")
	t.Logf("Retrieved BeliefSystem: %+v", beliefSystem)
	t.Logf("Number of beliefs: %d", len(beliefSystem.Beliefs))
	t.Logf("Number of observation contexts: %d", len(beliefSystem.ObservationContexts))
}
