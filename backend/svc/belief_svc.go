package svc

import (
	ai "epistemic-me-backend/ai"
	db "epistemic-me-backend/db"
	"epistemic-me-backend/svc/models"
	"fmt"
	"log"
	"reflect"

	"github.com/google/uuid"
)

type BeliefService struct {
	kv *db.KeyValueStore
	ai *ai.AIHelper
}

// NewBeliefService initializes and returns a new BeliefService.
func NewBeliefService(kv *db.KeyValueStore, ai *ai.AIHelper) *BeliefService {
	return &BeliefService{
		kv: kv,
		ai: ai,
	}
}

func (bsvc *BeliefService) CreateBelief(input *models.CreateBeliefInput) (*models.CreateBeliefOutput, error) {
	new_uuid := uuid.New().String()

	var beliefContent []models.Content
	beliefContent = append(beliefContent, models.Content{
		RawStr: input.BeliefContent,
	})

	newBeliefId := "bi_" + uuid.New().String()

	belief := models.Belief{
		ID:      newBeliefId,
		UserID:  input.UserID,
		Content: beliefContent,
		Type:    models.Statement,
		Version: 0,
	}

	belief.ID = new_uuid

	err := bsvc.kv.Store(input.UserID, new_uuid, belief, int(belief.Version))
	if err != nil {
		return nil, err
	}

	var empty_beliefs []models.Belief
	belief_system, err := bsvc.GetBeliefSystemFromBeliefs(empty_beliefs)
	if err != nil {
		return nil, err
	}

	return &models.CreateBeliefOutput{
		Belief:       belief,
		BeliefSystem: *belief_system,
	}, nil
}

func (bsvc *BeliefService) UpdateBelief(input *models.UpdateBeliefInput) (*models.UpdateBeliefOutput, error) {

	beliefResponse, err := bsvc.kv.Retrieve(input.UserID, input.BeliefID)
	if err != nil {
		log.Printf("Error in Retrieve: %v", err)
		return nil, err
	}

	existingBelief := beliefResponse.(*models.Belief)

	if existingBelief.Version != input.CurrentVersion {
		log.Printf("Version mismatch.")
		return nil, fmt.Errorf("Version mismatch. Version of the belief you are requesting to update is out of date. Requested: %s Actual: %s", input.CurrentVersion, existingBelief.Version)
	}

	existingBelief.Content[0].RawStr = input.UpdatedBeliefContent
	existingBelief.Version += 1
	existingBelief.Type = models.BeliefType(input.BeliefType)

	// todo: @deen update temporal information

	if !input.DryRun {
		err = bsvc.kv.Store(input.UserID, existingBelief.ID, *existingBelief, int(existingBelief.Version))
		if err != nil {
			log.Printf("Error in Store: %v", err)
			return nil, err
		}
	}

	var empty_beliefs []models.Belief
	belief_system, err := bsvc.GetBeliefSystemFromBeliefs(empty_beliefs)
	if err != nil {
		log.Printf("Error in getBeliefSystemFromBeliefs: %v", err)
		return nil, err
	}

	return &models.UpdateBeliefOutput{
		Belief:       *existingBelief,
		BeliefSystem: *belief_system,
	}, nil
}

func (bsvc *BeliefService) ListBeliefs(input *models.ListBeliefsInput) (*models.ListBeliefsOutput, error) {
	// Retrieve all beliefs for the user
	beliefs, err := bsvc.kv.ListByType(input.UserID, reflect.TypeOf(models.Belief{}))
	if err != nil {
		return nil, err
	}

	// Filter beliefs by the IDs specified in the input
	var filteredBeliefs []models.Belief
	for _, belief := range beliefs {
		storedBelief := belief.(*models.Belief)
		if len(input.BeliefIDs) > 0 {
			for _, id := range input.BeliefIDs {
				if storedBelief.ID == id {
					filteredBeliefs = append(filteredBeliefs, *storedBelief)
					break
				}
			}
		} else {
			filteredBeliefs = append(filteredBeliefs, *storedBelief)
		}
	}

	belief_system, err := bsvc.GetBeliefSystemFromBeliefs(filteredBeliefs)
	if err != nil {
		return nil, err
	}

	return &models.ListBeliefsOutput{
		Beliefs:      filteredBeliefs,
		BeliefSystem: *belief_system,
	}, nil
}

// Note this is an extremely expensive belief "materialization" over existing beleifs that should
// only be performed if necessary. todo: @deen enable a parameter to be passed in ListBeliefs that
// allows the client to control when this data is passed into the response.
func (bsvc *BeliefService) GetBeliefSystemFromBeliefs(beliefs []models.Belief) (*models.BeliefSystem, error) {

	// a 2D matrix of beliefs x versions of those beliefs
	var versionedBeliefs [][]models.Belief
	// a string representation of the latest version of each belief
	var belief_strs []string

	for _, belief := range beliefs {

		// query all versions of the belief and add to version matrix
		beliefVersionsInterface, err := bsvc.kv.RetrieveAllVersions(belief.UserID, belief.ID)
		if err != nil {
			return nil, err
		}

		var beliefVersions []models.Belief
		for _, beliefInterface := range beliefVersionsInterface {
			belief := beliefInterface.(*models.Belief)
			beliefVersions = append(beliefVersions, *belief)
		}
		versionedBeliefs = append(versionedBeliefs, beliefVersions)

		// add string representation
		var beliefContent string
		for _, content := range belief.Content {
			beliefContent += content.RawStr + "."
		}
		belief_strs = append(belief_strs, beliefContent)
	}

	belief_system_str, err := bsvc.ai.GenerateBeliefSystem(belief_strs)
	if err != nil {
		return nil, err
	}

	return &models.BeliefSystem{
		RawStr: belief_system_str,
	}, nil
}
