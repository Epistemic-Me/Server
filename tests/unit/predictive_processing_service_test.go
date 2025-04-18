package unit

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"epistemic-me-core/svc"
	"epistemic-me-core/svc/models"
)

func TestEnsurePredictiveProcessingContext(t *testing.T) {
	// Create the service
	pps := svc.NewPredictiveProcessingService()

	// Test case 1: Empty belief system
	bs := &models.BeliefSystem{
		Beliefs:           []*models.Belief{},
		EpistemicContexts: []*models.EpistemicContext{},
	}

	ppc := pps.EnsurePredictiveProcessingContext(bs)
	require.NotNil(t, ppc, "PredictiveProcessingContext should not be nil")
	require.Len(t, bs.EpistemicContexts, 1, "Should have created one EpistemicContext")
	assert.NotNil(t, bs.EpistemicContexts[0].PredictiveProcessingContext, "PPC should be created")
	assert.Empty(t, ppc.ObservationContexts, "ObservationContexts should be empty")
	assert.Empty(t, ppc.BeliefContexts, "BeliefContexts should be empty")

	// Test case 2: Belief system with EpistemicContext but no PPC
	bs = &models.BeliefSystem{
		Beliefs: []*models.Belief{},
		EpistemicContexts: []*models.EpistemicContext{
			{
				PredictiveProcessingContext: nil,
			},
		},
	}

	ppc = pps.EnsurePredictiveProcessingContext(bs)
	require.NotNil(t, ppc, "PredictiveProcessingContext should not be nil")
	assert.Len(t, bs.EpistemicContexts, 1, "Should still have one EpistemicContext")
	assert.NotNil(t, bs.EpistemicContexts[0].PredictiveProcessingContext, "PPC should be created")

	// Test case 3: Belief system with existing PPC
	existingPPC := &models.PredictiveProcessingContext{
		ObservationContexts: []*models.ObservationContext{
			{
				ID:   "test-oc",
				Name: "Test Observation",
			},
		},
		BeliefContexts: []*models.BeliefContext{
			{
				BeliefID:             "test-belief",
				ObservationContextID: "test-oc",
			},
		},
	}

	bs = &models.BeliefSystem{
		Beliefs: []*models.Belief{},
		EpistemicContexts: []*models.EpistemicContext{
			{
				PredictiveProcessingContext: existingPPC,
			},
		},
	}

	ppc = pps.EnsurePredictiveProcessingContext(bs)
	require.NotNil(t, ppc, "PredictiveProcessingContext should not be nil")
	assert.Equal(t, existingPPC, ppc, "Should return the existing PPC")
	assert.Len(t, ppc.ObservationContexts, 1, "Should have the existing observation contexts")
	assert.Len(t, ppc.BeliefContexts, 1, "Should have the existing belief contexts")
}

func TestCreateObservationContext(t *testing.T) {
	// Create the service
	pps := svc.NewPredictiveProcessingService()

	// Create a PPC to add the observation to
	ppc := &models.PredictiveProcessingContext{
		ObservationContexts: []*models.ObservationContext{},
		BeliefContexts:      []*models.BeliefContext{},
	}

	// Test creating an observation context
	question := "How often do you exercise?"
	answer := "I exercise three times per week"
	oc := pps.CreateObservationContext(ppc, question, answer)

	// Verify the observation context
	require.NotNil(t, oc, "ObservationContext should not be nil")
	assert.NotEmpty(t, oc.ID, "ObservationContext ID should not be empty")
	assert.NotEmpty(t, oc.Name, "ObservationContext Name should not be empty")
	assert.Contains(t, oc.Name, "Response", "Name should indicate it's a response")

	// Verify the observation was added to the PPC
	assert.Len(t, ppc.ObservationContexts, 1, "ObservationContext should be added to PPC")
	assert.Equal(t, oc, ppc.ObservationContexts[0], "The added ObservationContext should match")
}

func TestCreateBeliefContext(t *testing.T) {
	// Create the service
	pps := svc.NewPredictiveProcessingService()

	// Create a PPC to add the belief context to
	ppc := &models.PredictiveProcessingContext{
		ObservationContexts: []*models.ObservationContext{},
		BeliefContexts:      []*models.BeliefContext{},
	}

	// Test creating a belief context
	beliefID := uuid.New().String()
	observationID := uuid.New().String()
	confidenceScore := 0.8
	bc := pps.CreateBeliefContext(ppc, beliefID, observationID, confidenceScore)

	// Verify the belief context
	require.NotNil(t, bc, "BeliefContext should not be nil")
	assert.Equal(t, beliefID, bc.BeliefID, "BeliefID should match")
	assert.Equal(t, observationID, bc.ObservationContextID, "ObservationContextID should match")

	// Verify confidence ratings
	require.Len(t, bc.ConfidenceRatings, 1, "Should have one confidence rating")
	assert.Equal(t, confidenceScore, bc.ConfidenceRatings[0].ConfidenceScore, "Confidence score should match")
	assert.True(t, bc.ConfidenceRatings[0].Default, "Should be marked as default")

	// Verify the belief context was added to the PPC
	assert.Len(t, ppc.BeliefContexts, 1, "BeliefContext should be added to PPC")
	assert.Equal(t, bc, ppc.BeliefContexts[0], "The added BeliefContext should match")
}

func TestAddObservationFromInteraction(t *testing.T) {
	// Create the service
	pps := svc.NewPredictiveProcessingService()

	// Create beliefs and belief system
	belief1 := &models.Belief{
		ID: uuid.New().String(),
		Content: []models.Content{
			{RawStr: "I exercise regularly"},
		},
		Type: models.Statement,
	}

	belief2 := &models.Belief{
		ID: uuid.New().String(),
		Content: []models.Content{
			{RawStr: "I prefer morning workouts"},
		},
		Type: models.Statement,
	}

	bs := &models.BeliefSystem{
		Beliefs:           []*models.Belief{},
		EpistemicContexts: []*models.EpistemicContext{},
	}

	// Test adding an observation with extracted beliefs
	question := "What is your exercise routine?"
	answer := "I exercise three times per week in the morning"
	extractedBeliefs := []*models.Belief{belief1, belief2}

	// Add the observation and link beliefs
	pps.AddObservationFromInteraction(bs, question, answer, extractedBeliefs)

	// Verify the PPC was created and populated
	require.Len(t, bs.EpistemicContexts, 1, "Should have created one EpistemicContext")
	require.NotNil(t, bs.EpistemicContexts[0].PredictiveProcessingContext, "PPC should be created")

	ppc := bs.EpistemicContexts[0].PredictiveProcessingContext

	// Verify an observation context was created
	require.Len(t, ppc.ObservationContexts, 1, "Should have created one ObservationContext")
	assert.Contains(t, ppc.ObservationContexts[0].Name, "Response", "Name should indicate it's a response")

	// Verify belief contexts were created for both beliefs
	assert.Len(t, ppc.BeliefContexts, 2, "Should have created two BeliefContexts")

	// Check that both beliefs are linked to the observation
	ocID := ppc.ObservationContexts[0].ID
	beliefIDs := []string{belief1.ID, belief2.ID}

	for _, bc := range ppc.BeliefContexts {
		assert.Equal(t, ocID, bc.ObservationContextID, "BeliefContext should reference the ObservationContext")
		assert.Contains(t, beliefIDs, bc.BeliefID, "BeliefContext should reference one of the beliefs")
	}
}

func TestGetObservationsByBelief(t *testing.T) {
	// Create the service
	pps := svc.NewPredictiveProcessingService()

	// Create a belief ID
	beliefID := uuid.New().String()

	// Create two observation contexts
	oc1 := &models.ObservationContext{
		ID:   uuid.New().String(),
		Name: "Observation 1",
	}

	oc2 := &models.ObservationContext{
		ID:   uuid.New().String(),
		Name: "Observation 2",
	}

	// Create belief contexts that link the belief to the observations
	bc1 := &models.BeliefContext{
		BeliefID:             beliefID,
		ObservationContextID: oc1.ID,
	}

	bc2 := &models.BeliefContext{
		BeliefID:             beliefID,
		ObservationContextID: oc2.ID,
	}

	// Create a PPC with these contexts
	ppc := &models.PredictiveProcessingContext{
		ObservationContexts: []*models.ObservationContext{oc1, oc2},
		BeliefContexts:      []*models.BeliefContext{bc1, bc2},
	}

	// Get observations for the belief
	observations := pps.GetObservationsByBelief(ppc, beliefID)

	// Verify the correct observations were returned
	require.Len(t, observations, 2, "Should return two observations")
	assert.Contains(t, observations, oc1, "Should contain first observation")
	assert.Contains(t, observations, oc2, "Should contain second observation")

	// Test with a belief that has no observations
	otherBeliefID := uuid.New().String()
	observations = pps.GetObservationsByBelief(ppc, otherBeliefID)
	assert.Empty(t, observations, "Should return no observations for unknown belief")
}

func TestCalculateBeliefMetrics(t *testing.T) {
	// Create the service
	pps := svc.NewPredictiveProcessingService()

	// Create some beliefs
	belief1 := &models.Belief{
		ID: uuid.New().String(),
		Content: []models.Content{
			{RawStr: "Belief 1"},
		},
		Type: models.Statement,
	}

	belief2 := &models.Belief{
		ID: uuid.New().String(),
		Content: []models.Content{
			{RawStr: "Belief 2"},
		},
		Type: models.Statement,
	}

	belief3 := &models.Belief{
		ID: uuid.New().String(),
		Content: []models.Content{
			{RawStr: "Belief 3"},
		},
		Type: models.Statement,
	}

	// Create an observation context
	oc := &models.ObservationContext{
		ID:   uuid.New().String(),
		Name: "Test Observation",
	}

	// Create belief contexts that link some beliefs to the observation
	bc1 := &models.BeliefContext{
		BeliefID:             belief1.ID,
		ObservationContextID: oc.ID,
	}

	bc2 := &models.BeliefContext{
		BeliefID:             belief2.ID,
		ObservationContextID: oc.ID,
	}

	// Belief 3 has no observation context (not falsifiable)

	// Create a belief system with these components
	bs := &models.BeliefSystem{
		Beliefs: []*models.Belief{belief1, belief2, belief3},
		EpistemicContexts: []*models.EpistemicContext{
			{
				PredictiveProcessingContext: &models.PredictiveProcessingContext{
					ObservationContexts: []*models.ObservationContext{oc},
					BeliefContexts:      []*models.BeliefContext{bc1, bc2},
				},
			},
		},
	}

	// Calculate metrics
	metrics := pps.CalculateBeliefMetrics(bs)

	// Verify the metrics
	assert.Equal(t, int32(3), metrics.TotalBeliefs, "Should count all beliefs")
	assert.Equal(t, int32(2), metrics.TotalFalsifiableBeliefs, "Should count beliefs with observation contexts as falsifiable")
	assert.Equal(t, int32(3), metrics.TotalBeliefStatements, "Should count all beliefs as statements")

	// Clarification score should be proportion of falsifiable beliefs
	expectedScore := float64(2) / float64(3)
	assert.Equal(t, expectedScore, metrics.ClarificationScore, "Clarification score should be proportion of falsifiable beliefs")
}
