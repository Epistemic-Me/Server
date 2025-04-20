package unit

import (
	"context"
	"testing"

	"epistemic-me-core/db"
	"epistemic-me-core/svc"
	"epistemic-me-core/svc/models"

	"github.com/stretchr/testify/require"
)

func TestCreateAndUpdatePhilosophy(t *testing.T) {
	kv, err := db.NewKeyValueStore("")
	require.NoError(t, err)
	svc := svc.NewSelfModelService(kv, nil, nil)
	ctx := context.Background()

	// Create
	createInput := &models.CreatePhilosophyInput{
		Description:         "# Metabolic Health Philosophy\n\n## Experiential Narrative\n[[C: Circadian Rhythm]] [[S: asleep]] → [[S: awake]]\n",
		ExtrapolateContexts: true,
	}
	createOut, err := svc.CreatePhilosophy(ctx, createInput)
	require.NoError(t, err)
	require.NotNil(t, createOut.Philosophy)
	philosophyID := createOut.Philosophy.ID
	require.Equal(t, createInput.Description, createOut.Philosophy.Description)
	require.Equal(t, createInput.ExtrapolateContexts, createOut.Philosophy.ExtrapolateContexts)

	// Retrieve and check
	stored, err := kv.Retrieve(philosophyID, "Philosophy")
	require.NoError(t, err)
	storedPhilosophy, ok := stored.(*models.Philosophy)
	require.True(t, ok)
	require.Equal(t, createInput.Description, storedPhilosophy.Description)

	// Update
	updateInput := &models.UpdatePhilosophyInput{
		PhilosophyID:        philosophyID,
		Description:         "# Updated Philosophy\n\n## Experiential Narrative\n[[C: Sleep Architecture]] [[S: light-n1]] → [[S: slow-wave]]\n",
		ExtrapolateContexts: false,
	}
	updateOut, err := svc.UpdatePhilosophy(ctx, updateInput)
	require.NoError(t, err)
	require.NotNil(t, updateOut.Philosophy)
	require.Equal(t, updateInput.Description, updateOut.Philosophy.Description)
	require.Equal(t, updateInput.ExtrapolateContexts, updateOut.Philosophy.ExtrapolateContexts)

	// Retrieve and check update
	stored2, err := kv.Retrieve(philosophyID, "Philosophy")
	require.NoError(t, err)
	storedPhilosophy2, ok := stored2.(*models.Philosophy)
	require.True(t, ok)
	require.Equal(t, updateInput.Description, storedPhilosophy2.Description)

	// Error: update non-existent
	badUpdate := &models.UpdatePhilosophyInput{
		PhilosophyID:        "non-existent-id",
		Description:         "desc",
		ExtrapolateContexts: true,
	}
	_, err = svc.UpdatePhilosophy(ctx, badUpdate)
	require.Error(t, err)

	// Extrapolation logic (not in service)
	contexts := models.ExtrapolateObservationContexts(updateInput.Description)
	require.NotEmpty(t, contexts)
	found := false
	for _, ctx := range contexts {
		if ctx.Name == "Sleep Architecture" && contains(ctx.PossibleStates, "light-n1") && contains(ctx.PossibleStates, "slow-wave") {
			found = true
		}
	}
	require.True(t, found, "Should extract Sleep Architecture context with correct states")
}

func TestPhilosophyExtrapolationCaching(t *testing.T) {
	kv, err := db.NewKeyValueStore("")
	require.NoError(t, err)
	svc := svc.NewSelfModelService(kv, nil, nil)
	ctx := context.Background()

	desc := "Test [[C: Circadian Rhythm]] [[S: asleep]]"
	createInput := &models.CreatePhilosophyInput{
		Description:         desc,
		ExtrapolateContexts: true,
	}
	createOut, err := svc.CreatePhilosophy(ctx, createInput)
	require.NoError(t, err)
	philosophyID := createOut.Philosophy.ID

	// First call: should parse and cache
	svc.CacheMu().RLock()
	cached, ok := svc.Cache()[philosophyID]
	svc.CacheMu().RUnlock()
	require.True(t, ok)
	require.Len(t, cached, 2)

	// Update: should invalidate and update cache
	updateInput := &models.UpdatePhilosophyInput{
		PhilosophyID:        philosophyID,
		Description:         "Test [[C: New Context]]",
		ExtrapolateContexts: true,
	}
	_, err = svc.UpdatePhilosophy(ctx, updateInput)
	require.NoError(t, err)
	svc.CacheMu().RLock()
	cached, ok = svc.Cache()[philosophyID]
	svc.CacheMu().RUnlock()
	require.True(t, ok)
	require.Len(t, cached, 1)
	require.Equal(t, "New Context", cached[0].Name)
}

func TestPhilosophyExtrapolation_NoMarkers(t *testing.T) {
	kv, err := db.NewKeyValueStore("")
	require.NoError(t, err)
	svc := svc.NewSelfModelService(kv, nil, nil)
	ctx := context.Background()

	createInput := &models.CreatePhilosophyInput{
		Description:         "This description has no markers.",
		ExtrapolateContexts: true,
	}
	createOut, err := svc.CreatePhilosophy(ctx, createInput)
	require.NoError(t, err)
	require.Empty(t, createOut.ExtrapolatedObservationContexts)
}

func TestPhilosophyExtrapolation_DuplicateMarkers(t *testing.T) {
	kv, err := db.NewKeyValueStore("")
	require.NoError(t, err)
	svc := svc.NewSelfModelService(kv, nil, nil)
	ctx := context.Background()

	createInput := &models.CreatePhilosophyInput{
		Description:         "Test [[C: Circadian Rhythm]] [[C: Circadian Rhythm]] [[S: asleep]]",
		ExtrapolateContexts: true,
	}
	createOut, err := svc.CreatePhilosophy(ctx, createInput)
	require.NoError(t, err)
	require.Len(t, createOut.ExtrapolatedObservationContexts, 2)
	names := []string{createOut.ExtrapolatedObservationContexts[0].Name, createOut.ExtrapolatedObservationContexts[1].Name}
	require.Contains(t, names, "Circadian Rhythm")
	require.Contains(t, names, "asleep")
}

func TestPhilosophyExtrapolation_Disabled(t *testing.T) {
	kv, err := db.NewKeyValueStore("")
	require.NoError(t, err)
	svc := svc.NewSelfModelService(kv, nil, nil)
	ctx := context.Background()

	createInput := &models.CreatePhilosophyInput{
		Description:         "Test [[C: Circadian Rhythm]] [[S: asleep]]",
		ExtrapolateContexts: false,
	}
	createOut, err := svc.CreatePhilosophy(ctx, createInput)
	require.NoError(t, err)
	require.Empty(t, createOut.ExtrapolatedObservationContexts)
}

func TestPhilosophyExtrapolation_UpdateDisablesCaching(t *testing.T) {
	kv, err := db.NewKeyValueStore("")
	require.NoError(t, err)
	svc := svc.NewSelfModelService(kv, nil, nil)
	ctx := context.Background()

	createInput := &models.CreatePhilosophyInput{
		Description:         "Test [[C: Circadian Rhythm]]",
		ExtrapolateContexts: true,
	}
	createOut, err := svc.CreatePhilosophy(ctx, createInput)
	require.NoError(t, err)
	philosophyID := createOut.Philosophy.ID

	// Confirm cache is set
	svc.CacheMu().RLock()
	_, ok := svc.Cache()[philosophyID]
	svc.CacheMu().RUnlock()
	require.True(t, ok)

	// Update with ExtrapolateContexts: false
	updateInput := &models.UpdatePhilosophyInput{
		PhilosophyID:        philosophyID,
		Description:         "Test [[C: Circadian Rhythm]]",
		ExtrapolateContexts: false,
	}
	_, err = svc.UpdatePhilosophy(ctx, updateInput)
	require.NoError(t, err)

	// Cache should be cleared
	svc.CacheMu().RLock()
	_, ok = svc.Cache()[philosophyID]
	svc.CacheMu().RUnlock()
	require.False(t, ok)
}
