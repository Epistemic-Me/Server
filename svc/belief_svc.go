package svc

import (
	ai "epistemic-me-core/ai"
	db "epistemic-me-core/db"
	"epistemic-me-core/svc/models"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/google/uuid"
)

const (
	LogLevelDebug = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

var currentLogLevel = LogLevelInfo

type BeliefService struct {
	kvStore *db.KeyValueStore
	ai      *ai.AIHelper
}

// NewBeliefService initializes and returns a new BeliefService.
func NewBeliefService(kvStore *db.KeyValueStore, ai *ai.AIHelper) *BeliefService {
	return &BeliefService{
		kvStore: kvStore,
		ai:      ai,
	}
}

func (bsvc *BeliefService) CreateBelief(input *models.CreateBeliefInput) (*models.CreateBeliefOutput, error) {
	newBeliefId := "bi_" + uuid.New().String()

	belief := models.Belief{
		ID:          newBeliefId,
		SelfModelID: input.SelfModelID,
		Content:     []models.Content{{RawStr: input.BeliefContent}},
		Type:        input.BeliefType,
		Version:     1,
		Active:      true,
	}

	// Try to get existing belief system or create new one
	beliefSystem, err := bsvc.retrieveBeliefSystem(input.SelfModelID)
	if err != nil {
		if strings.Contains(err.Error(), "key not found") {
			// Create new belief system
			beliefSystem = &models.BeliefSystem{
				Beliefs:           make([]*models.Belief, 0),
				EpistemicContexts: make([]*models.EpistemicContext, 0),
			}
		} else {
			return nil, fmt.Errorf("failed to retrieve belief system: %w", err)
		}
	}

	// Store the belief first
	err = bsvc.storeBeliefValue(input.SelfModelID, &belief)
	if err != nil {
		return nil, fmt.Errorf("failed to store belief: %w", err)
	}

	// Create a copy of the belief to store in the belief system
	beliefCopy := belief // Make a copy of the belief value
	beliefSystem.Beliefs = append(beliefSystem.Beliefs, &beliefCopy)

	// Store updated belief system
	err = bsvc.kvStore.Store(input.SelfModelID, "BeliefSystem", *beliefSystem, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to store belief system: %w", err)
	}

	return &models.CreateBeliefOutput{
		Belief:       belief,
		BeliefSystem: *beliefSystem,
	}, nil
}

func (bsvc *BeliefService) UpdateBelief(input *models.UpdateBeliefInput) (*models.UpdateBeliefOutput, error) {
	existingBelief, err := bsvc.retrieveBeliefValue(input.SelfModelID, input.ID)
	if err != nil {
		logf(LogLevelError, "Error in Retrieve: %v", err)
		return nil, err
	}

	existingBelief.Content[0].RawStr = input.UpdatedBeliefContent
	existingBelief.Version++
	existingBelief.Type = models.BeliefType(input.BeliefType)

	// todo: @deen update temporal information
	if !input.DryRun {
		err = bsvc.storeBeliefValue(input.SelfModelID, existingBelief)
		if err != nil {
			log.Printf("Error in Store: %v", err)
			return nil, err
		}
	}

	// var empty_beliefs []models.Belief
	// beliefSystem, err := bsvc.getBeliefSystemFromBeliefs(empty_beliefs)
	beliefSystem, err := bsvc.GetBeliefSystemFromBeliefs([]*models.Belief{existingBelief})

	if err != nil {
		logf(LogLevelError, "Error in getBeliefSystemFromBeliefs: %v", err)
		return nil, err
	}

	return &models.UpdateBeliefOutput{
		Belief:       *existingBelief,
		BeliefSystem: *beliefSystem,
	}, nil
}

func (bsvc *BeliefService) DeleteBelief(input *models.DeleteBeliefInput) (*models.DeleteBeliefOutput, error) {
	existingBelief, err := bsvc.retrieveBeliefValue(input.SelfModelID, input.ID)
	if err != nil {
		logf(LogLevelError, "Error in Retrieve: %v", err)
		return nil, err
	}

	existingBelief.Active = false
	existingBelief.Version++
	// todo: @deen update temporal information
	if !input.DryRun {
		err = bsvc.storeBeliefValue(input.SelfModelID, existingBelief)
		if err != nil {
			log.Printf("Error in Store: %v", err)
			return nil, err
		}
	}

	beliefSystem := &models.BeliefSystem{}
	if input.ComputeBeliefSystem {
		beliefSystem, err = bsvc.GetBeliefSystemFromBeliefs([]*models.Belief{existingBelief})
		if err != nil {
			logf(LogLevelError, "Error in getBeliefSystemFromBeliefs: %v", err)
			return nil, err
		}
	}

	return &models.DeleteBeliefOutput{
		Belief:       *existingBelief,
		BeliefSystem: *beliefSystem,
	}, nil
}

func (bsvc *BeliefService) ListBeliefs(input *models.ListBeliefsInput) (*models.ListBeliefsOutput, error) {
	logf(LogLevelDebug, "ListBeliefs called with input: %+v", input)

	// Use ListByType to get all Belief objects for the user
	beliefObjects, err := bsvc.kvStore.ListByType(input.SelfModelID, reflect.TypeOf(models.Belief{}))
	if err != nil {
		return nil, fmt.Errorf("error retrieving beliefs: %v", err)
	}

	// Convert the list of interface{} to []*models.Belief
	beliefs := make([]*models.Belief, 0)
	for _, obj := range beliefObjects {
		if belief, ok := obj.(*models.Belief); ok {
			beliefs = append(beliefs, belief)
		}
	}

	// If specific belief IDs are provided, filter the beliefs
	if len(input.BeliefIDs) > 0 {
		filteredBeliefs := make([]*models.Belief, 0)
		for _, belief := range beliefs {
			for _, id := range input.BeliefIDs {
				if belief.ID == id {
					filteredBeliefs = append(filteredBeliefs, belief)
					break
				}
			}
		}
		beliefs = filteredBeliefs
	}

	// Only return active beliefs
	activeBeliefs := make([]*models.Belief, 0)
	for _, belief := range beliefs {
		if belief.Active {
			activeBeliefs = append(activeBeliefs, belief)
		}
	}

	beliefSystem, err := bsvc.GetBeliefSystemFromBeliefs(beliefs)
	if err != nil {
		logf(LogLevelError, "Error in getBeliefSystemFromBeliefs: %v", err)
		return nil, err
	}

	return &models.ListBeliefsOutput{
		Beliefs:      activeBeliefs,
		BeliefSystem: *beliefSystem,
	}, nil
}

func (bsvc *BeliefService) GetBeliefSystemFromBeliefs(beliefs []*models.Belief) (*models.BeliefSystem, error) {
	logf(LogLevelDebug, "getBeliefSystemFromBeliefs called with %d beliefs", len(beliefs))

	return &models.BeliefSystem{
		Beliefs: beliefs,
	}, nil
}

func (bsvc *BeliefService) filterBeliefsByObservationContexts(beliefs []*models.Belief, contextIDs []string) []*models.Belief {
	var filteredBeliefs []*models.Belief
	for _, belief := range beliefs {
		if bsvc.beliefMatchesContexts(belief, contextIDs) {
			filteredBeliefs = append(filteredBeliefs, belief)
		}
	}
	return filteredBeliefs
}

func (bsvc *BeliefService) beliefMatchesContexts(belief *models.Belief, contextIDs []string) bool {
	// Get the belief system for this belief's self model
	beliefSystem, err := bsvc.retrieveBeliefSystem(belief.SelfModelID)
	if err != nil {
		return false
	}

	// Check if any of the belief contexts match the given context IDs
	for _, ec := range beliefSystem.EpistemicContexts {
		predictiveProcessingContext := ec.PredictiveProcessingContext
		if predictiveProcessingContext != nil {
			for _, bc := range predictiveProcessingContext.BeliefContexts {
				if bc.BeliefID == belief.ID {
					for _, contextID := range contextIDs {
						if bc.ObservationContextID == contextID {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

// Add this method to BeliefService
func (bsvc *BeliefService) storeBeliefValue(selfModelID string, belief *models.Belief) error {
	return bsvc.kvStore.Store(selfModelID, belief.ID, *belief, int(belief.Version))
}

// Add this method to BeliefService
func (bsvc *BeliefService) retrieveBeliefValue(selfModelID, beliefID string) (*models.Belief, error) {
	value, err := bsvc.kvStore.Retrieve(selfModelID, beliefID)
	if err != nil {
		return nil, err
	}
	belief, ok := value.(*models.Belief)
	if !ok {
		return nil, fmt.Errorf("retrieved value is not a Belief")
	}
	return belief, nil
}

func logf(level int, format string, v ...interface{}) {
	if level >= currentLogLevel {
		log.Printf(format, v...)
	}
}

// Add this method to BeliefService
func (bsvc *BeliefService) retrieveBeliefSystem(selfModelID string) (*models.BeliefSystem, error) {
	value, err := bsvc.kvStore.Retrieve(selfModelID, "BeliefSystem")
	if err != nil || value == nil {
		// Create a new belief system if one doesn't exist
		beliefSystem := &models.BeliefSystem{
			Beliefs:           make([]*models.Belief, 0),
			EpistemicContexts: make([]*models.EpistemicContext, 0),
		}
		err = bsvc.kvStore.Store(selfModelID, "BeliefSystem", *beliefSystem, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to create initial belief system: %v", err)
		}
		return beliefSystem, nil
	}

	// Type assert the result to *models.BeliefSystem
	beliefSystem, ok := value.(*models.BeliefSystem)
	if !ok {
		return nil, fmt.Errorf("invalid belief system data type: %T", value)
	}

	// Use ListBeliefs to get all active beliefs
	listBeliefsOutput, err := bsvc.ListBeliefs(&models.ListBeliefsInput{
		SelfModelID: selfModelID,
	})
	if err != nil {
		return nil, fmt.Errorf("error retrieving beliefs: %v", err)
	}

	// Preserve epistemic contexts but update beliefs
	beliefSystem.Beliefs = listBeliefsOutput.Beliefs

	return beliefSystem, nil
}

// Add this method to the BeliefService
func (bsvc *BeliefService) GetBeliefSystem(selfModelID string) (*models.BeliefSystem, error) {
	beliefSystem, err := bsvc.retrieveBeliefSystem(selfModelID)
	if err != nil {
		logf(LogLevelError, "Error retrieving belief system: %v", err)
		return nil, fmt.Errorf("error retrieving belief system: %v", err)
	}

	return beliefSystem, nil
}

// Add these methods to BeliefService
func (bsvc *BeliefService) ConceptualizeBeliefSystem(beliefSystem *models.BeliefSystem) error {
	// TODO: Implementation will use AI helper to generate conceptualization
	return nil
}

func (bsvc *BeliefService) ComputeMetrics(beliefSystem *models.BeliefSystem) error {
	// TODO: Implementation will use metrics package to compute scores
	return nil
}
