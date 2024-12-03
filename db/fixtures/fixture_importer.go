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
func ImportFixtures(kvStore *db.KeyValueStore) error {
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
		return fmt.Errorf("error unmarshaling YAML: %v", err)
	}

	fixtureUserID := "fixture-self-model-id"
	fixtureSelfModel := models.SelfModel{
		ID: fixtureUserID,
	}

	err = kvStore.Store(fixtureUserID, "SelfModelId", fixtureSelfModel, 1)
	if err != nil {
		return fmt.Errorf("error storing fixture self model: %v", err)
	}

	beliefSystem := models.BeliefSystem{
		Beliefs:             []*models.Belief{},
		ObservationContexts: []*models.ObservationContext{},
		BeliefContexts:      []*models.BeliefContext{},
	}

	for _, example := range fixture.BeliefSystem.Examples {
		contextMap := make(map[string]*models.ObservationContext)
		for _, oc := range example.ObservationContext {
			context := &models.ObservationContext{
				ID:             fmt.Sprintf("%s_%s", example.Name, oc.ContextName),
				Name:           oc.ContextName,
				ParentID:       oc.NestedWithin,
				PossibleStates: []string{}, // Will be populated from beliefs
			}
			contextMap[oc.ContextName] = context
			beliefSystem.ObservationContexts = append(beliefSystem.ObservationContexts, context)
		}

		for _, b := range example.Beliefs {
			belief := &models.Belief{
				ID:          fmt.Sprintf("%s_%s", example.Name, b.BeliefName),
				SelfModelID: fixtureSelfModel.ID,
				Version:     1,
				Type:        models.Falsifiable,
				Content:     []models.Content{{RawStr: b.Description}},
			}
			beliefSystem.Beliefs = append(beliefSystem.Beliefs, belief)

			// Create BeliefContext for each belief
			beliefContext := &models.BeliefContext{
				BeliefID:             belief.ID,
				ObservationContextID: contextMap[b.Context].ID,
				ConfidenceRatings:    []models.ConfidenceRating{{ConfidenceScore: 0.8, Default: true}},
				ConditionalProbs: map[string]float32{
					b.PredictedOutcome:      0.8,
					b.CounterfactualOutcome: 0.2,
				},
				EpistemicEmotion: models.Confirmation,
				EmotionIntensity: 0.5,
			}
			beliefSystem.BeliefContexts = append(beliefSystem.BeliefContexts, beliefContext)

			// Add states to context's possible states
			contextMap[b.Context].PossibleStates = append(
				contextMap[b.Context].PossibleStates,
				b.PredictedOutcome,
				b.CounterfactualOutcome,
			)
		}
	}

	err = kvStore.Store(fixtureUserID, "BeliefSystemId", beliefSystem, 1)
	if err != nil {
		return fmt.Errorf("error storing fixture belief system: %v", err)
	}

	fmt.Printf("Fixture belief system has been successfully imported with %d beliefs and %d observation contexts\n",
		len(beliefSystem.Beliefs), len(beliefSystem.ObservationContexts))

	return nil
}
