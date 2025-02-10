package integration

import (
	"context"
	"testing"

	pb "epistemic-me-core/pb"
	models "epistemic-me-core/pb/models"
	svc_models "epistemic-me-core/svc/models"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
)

func TestBeliefSystemAPIIntegration(t *testing.T) {
	// Remove unused context
	require.NotEmpty(t, testUserID, "testUserID should be set")

	// Reset the store to a clean state
	err := resetStore()
	require.NoError(t, err, "Failed to reset store")

	// Verify that fixtures were imported correctly
	fixtureBeliefSystem, err := kvStore.Retrieve(testUserID, "BeliefSystem")
	require.NoError(t, err, "Failed to retrieve fixture belief system")
	require.NotNil(t, fixtureBeliefSystem, "Fixture belief system should not be nil")

	// Log the type of the retrieved object
	t.Logf("Retrieved object type: %T", fixtureBeliefSystem)

	// Try to cast to different possible types
	switch bs := fixtureBeliefSystem.(type) {
	case *models.BeliefSystem:
		t.Logf("Successfully cast to *models.BeliefSystem with %d beliefs", len(bs.Beliefs))
		if bs.EpistemicContexts != nil {
			t.Logf("Number of epistemic contexts: %d", len(bs.EpistemicContexts.EpistemicContexts))
		}
	case *svc_models.BeliefSystem:
		t.Logf("Successfully cast to *svc_models.BeliefSystem with %d beliefs and %d epistemic contexts",
			len(bs.Beliefs), len(bs.EpistemicContexts))
	default:
		t.Fatalf("Retrieved object is not a recognized BeliefSystem type: %T", fixtureBeliefSystem)
	}

	// Use the global client instead of creating a new one
	resp, err := client.GetBeliefSystem(context.Background(), connect.NewRequest(&pb.GetBeliefSystemRequest{
		SelfModelId: testUserID,
	}))
	require.NoError(t, err, "GetBeliefSystem failed")

	beliefSystem := resp.Msg.BeliefSystem
	require.NotNil(t, beliefSystem, "BeliefSystem should not be nil")
	require.NotEmpty(t, beliefSystem.Beliefs, "Beliefs should not be empty")
	require.NotNil(t, beliefSystem.EpistemicContexts, "EpistemicContexts should not be nil")

	t.Logf("Retrieved BeliefSystem: %+v", beliefSystem)
	t.Logf("Number of beliefs: %d", len(beliefSystem.Beliefs))
	t.Logf("Number of epistemic contexts: %d", len(beliefSystem.EpistemicContexts.EpistemicContexts))

	// Verify the structure of epistemic contexts
	for i, ec := range beliefSystem.EpistemicContexts.EpistemicContexts {
		t.Logf("Examining epistemic context %d", i)
		if ppc := ec.GetPredictiveProcessingContext(); ppc != nil {
			t.Logf("  PredictiveProcessingContext found")
			t.Logf("  Number of observation contexts: %d", len(ppc.ObservationContexts))
			t.Logf("  Number of belief contexts: %d", len(ppc.BeliefContexts))

			// Verify observation contexts
			for j, oc := range ppc.ObservationContexts {
				require.NotEmpty(t, oc.Id, "ObservationContext ID should not be empty")
				require.NotEmpty(t, oc.Name, "ObservationContext Name should not be empty")
				t.Logf("    ObservationContext %d: %s (%s)", j, oc.Name, oc.Id)
			}

			// Verify belief contexts
			for j, bc := range ppc.BeliefContexts {
				require.NotEmpty(t, bc.BeliefId, "BeliefContext BeliefID should not be empty")
				require.NotEmpty(t, bc.ObservationContextId, "BeliefContext ObservationContextID should not be empty")
				t.Logf("    BeliefContext %d: Belief %s in Context %s", j, bc.BeliefId, bc.ObservationContextId)
			}
		}
	}
}
