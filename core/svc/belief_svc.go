package svc

import (
	ai "epistemic-me-core/ai"
	db "epistemic-me-core/db"
	"epistemic-me-core/svc/models"
	"fmt"
	"log"
	"reflect"

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
		Type:        models.Statement,
		Version:     1,
	}

	err := bsvc.storeBeliefValue(input.SelfModelID, &belief)
	if err != nil {
		return nil, err
	}

	// var empty_beliefs []models.Belief
	// beliefSystem, err := bsvc.GetBeliefSystemFromBeliefs(empty_beliefs)
	beliefSystem, err := bsvc.getBeliefSystemFromBeliefs([]*models.Belief{&belief})

	if err != nil {
		return nil, err
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
	beliefSystem, err := bsvc.getBeliefSystemFromBeliefs([]*models.Belief{existingBelief})

	if err != nil {
		logf(LogLevelError, "Error in getBeliefSystemFromBeliefs: %v", err)
		return nil, err
	}

	return &models.UpdateBeliefOutput{
		Belief:       *existingBelief,
		BeliefSystem: *beliefSystem,
	}, nil
}

func (bsvc *BeliefService) ListBeliefs(input *models.ListBeliefsInput) (*models.ListBeliefsOutput, error) {
	logf(LogLevelDebug, "ListBeliefs called with input: %+v", input)

	beliefSystem, err := bsvc.retrieveBeliefSystem(input.SelfModelID)
	if err != nil {
		logf(LogLevelError, "Error in ListBeliefs: %v", err)
		return nil, fmt.Errorf("error retrieving beliefs: %v", err)
	}

	// TODO: Filter beliefs by the IDs specified in the input

	if err != nil {
		return nil, err
	}

	return &models.ListBeliefsOutput{
		BeliefSystem: *beliefSystem,
	}, nil
}

func (bsvc *BeliefService) getBeliefSystemFromBeliefs(beliefs []*models.Belief) (*models.BeliefSystem, error) {
	logf(LogLevelDebug, "getBeliefSystemFromBeliefs called with %d beliefs", len(beliefs))

	// TODO: Implement logic to populate observation contexts
	observationContexts := []*models.ObservationContext{}

	return &models.BeliefSystem{
		Beliefs:             beliefs,
		ObservationContexts: observationContexts,
	}, nil
}

func (bsvc *BeliefService) filterBeliefsByObservationContexts(beliefs []*models.Belief, contextIDs []string) []*models.Belief {
	var filteredBeliefs []*models.Belief
	for _, belief := range beliefs {
		if bsvc.beliefMatchesContexts(*belief, contextIDs) {
			filteredBeliefs = append(filteredBeliefs, belief)
		}
	}
	return filteredBeliefs
}

func (bsvc *BeliefService) beliefMatchesContexts(belief models.Belief, contextIDs []string) bool {
	for _, contextID := range contextIDs {
		for _, beliefContextID := range belief.ObservationContextIDs {
			if contextID == beliefContextID {
				return true
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
	belief, ok := value.(models.Belief)
	if !ok {
		return nil, fmt.Errorf("retrieved value is not a Belief")
	}
	return &belief, nil
}

func (bsvc *BeliefService) getAllBeliefs(selfModelID string) ([]*models.Belief, error) {
	beliefs := []*models.Belief{}

	// Use ListByType to get all Belief objects for the user
	beliefObjects, err := bsvc.kvStore.ListByType(selfModelID, reflect.TypeOf(models.Belief{}))
	if err != nil {
		return nil, fmt.Errorf("error retrieving beliefs: %v", err)
	}

	// Convert the list of interface{} to []*models.Belief
	for _, obj := range beliefObjects {
		if belief, ok := obj.(*models.Belief); ok {
			beliefs = append(beliefs, belief)
		}
	}

	return beliefs, nil
}

func logf(level int, format string, v ...interface{}) {
	if level >= currentLogLevel {
		log.Printf(format, v...)
	}
}

// Add this method to BeliefService
func (bsvc *BeliefService) retrieveBeliefSystem(selfModelID string) (*models.BeliefSystem, error) {
	value, err := bsvc.kvStore.Retrieve(selfModelID, "BeliefSystemId")
	if err != nil {
		logf(LogLevelError, "Error retrieving belief system: %v", err)
		return nil, fmt.Errorf("error retrieving belief system: %v", err)
	}

	beliefSystem, ok := value.(*models.BeliefSystem)
	if !ok {
		logf(LogLevelError, "Retrieved value is not a BeliefSystem. Type: %T", value)
		return nil, fmt.Errorf("invalid belief system data type")
	}

	return beliefSystem, nil
}

// Add this method to the BeliefService
func (bsvc *BeliefService) GetBeliefSystem(selfModelID string) (*models.BeliefSystem, error) {
	beliefSystem, err := bsvc.retrieveBeliefSystem(selfModelID)
	if err != nil {
		logf(LogLevelError, "Error retrieving belief system: %v", err)
		return nil, fmt.Errorf("error retrieving belief system: %v", err)
	}

	if beliefSystem == nil {
		// Return empty belief system if none exists
		return &models.BeliefSystem{
			Beliefs:             []*models.Belief{},
			ObservationContexts: []*models.ObservationContext{},
		}, nil
	}

	return beliefSystem, nil
}
