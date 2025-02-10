package fixture_models

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"epistemic-me-core/db"
	"epistemic-me-core/svc/models"

	"gopkg.in/yaml.v3"
)

// ImportFixtures imports the belief system fixtures into the given KeyValueStore
func ImportFixtures(kvStore *db.KeyValueStore, userID string) error {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("failed to get current file path")
	}
	currentDir := filepath.Dir(filename)
	yamlFilePath := filepath.Join(currentDir, "belief_system_fixture.yaml")

	yamlFile, err := os.ReadFile(yamlFilePath)
	if err != nil {
		return fmt.Errorf("error reading YAML file: %v", err)
	}

	var fixture BeliefSystemFixture
	err = yaml.Unmarshal(yamlFile, &fixture)
	if err != nil {
		return fmt.Errorf("error parsing YAML: %v", err)
	}

	beliefSystem := &models.BeliefSystem{
		Beliefs:           make([]*models.Belief, 0),
		EpistemicContexts: make([]*models.EpistemicContext, 0),
	}

	// Map to store contexts by name for easy lookup
	contextMap := make(map[string]*models.ObservationContext)

	// First, create all observation contexts
	for _, example := range fixture.BeliefSystem.Examples {
		// Create a new PredictiveProcessingContext for each example
		ppc := &models.PredictiveProcessingContext{
			ObservationContexts: make([]*models.ObservationContext, 0),
			BeliefContexts:      make([]*models.BeliefContext, 0),
		}

		for _, oc := range example.ObservationContext {
			context := &models.ObservationContext{
				ID:             fmt.Sprintf("context-%s", oc.ContextName),
				Name:           oc.ContextName,
				ParentID:       oc.NestedWithin,
				PossibleStates: make([]string, 0),
			}
			contextMap[oc.ContextName] = context
			ppc.ObservationContexts = append(ppc.ObservationContexts, context)
		}

		// Create beliefs and associate them with contexts
		for _, b := range example.Beliefs {
			belief := &models.Belief{
				ID:          fmt.Sprintf("belief-%s", b.BeliefName),
				SelfModelID: userID,
				Version:     1,
				Type:        models.Falsifiable,
				Content: []models.Content{
					{RawStr: b.Description},
				},
				Active: true,
			}
			beliefSystem.Beliefs = append(beliefSystem.Beliefs, belief)

			// Create belief context
			if context, ok := contextMap[b.Context]; ok {
				// Add states to context's possible states
				context.PossibleStates = append(
					context.PossibleStates,
					b.PredictedOutcome,
					b.CounterfactualOutcome,
				)

				// Create belief context
				beliefContext := &models.BeliefContext{
					BeliefID:             belief.ID,
					ObservationContextID: context.ID,
					ConfidenceRatings: []models.ConfidenceRating{
						{ConfidenceScore: 0.8, Default: true},
					},
					ConditionalProbs: map[string]float32{
						b.PredictedOutcome:      0.8,
						b.CounterfactualOutcome: 0.2,
					},
					EpistemicEmotion: models.Confirmation,
					EmotionIntensity: 0.5,
				}
				ppc.BeliefContexts = append(ppc.BeliefContexts, beliefContext)
			}
		}

		// Create an EpistemicContext for each example's PredictiveProcessingContext
		epistemicContext := &models.EpistemicContext{
			AssociatedBeleifs:           nil,
			PredictiveProcessingContext: ppc,
		}
		beliefSystem.EpistemicContexts = append(beliefSystem.EpistemicContexts, epistemicContext)
	}

	// First store each individual belief
	for _, belief := range beliefSystem.Beliefs {
		err = kvStore.Store(userID, belief.ID, *belief, int(belief.Version))
		if err != nil {
			return fmt.Errorf("error storing belief %s: %v", belief.ID, err)
		}
	}

	// Store the belief system with both keys for compatibility
	err = kvStore.Store(userID, "BeliefSystem", *beliefSystem, 1)
	if err != nil {
		return fmt.Errorf("error storing belief system: %v", err)
	}
	err = kvStore.Store(userID, "BeliefSystemId", *beliefSystem, 1)
	if err != nil {
		return fmt.Errorf("error storing belief system with ID key: %v", err)
	}

	fmt.Printf("Fixture belief system has been successfully imported with %d beliefs and %d epistemic contexts for user %s\n",
		len(beliefSystem.Beliefs), len(beliefSystem.EpistemicContexts), userID)

	return nil
}
