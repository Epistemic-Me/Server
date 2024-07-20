package svc

import (
	ai "epistemic-me-backend/ai"
	db "epistemic-me-backend/db"
	"epistemic-me-backend/svc/models"
	"reflect"
)

type BeliefService struct {
	kv db.KeyValueStore
	ai ai.AIHelper
}

func (bsvc *BeliefService) CreateBelief(input *models.CreateBeliefInput) (*models.CreateBeliefOutput, error) {
	new_uuid := uuid.uuid4()

	var beliefContent []models.Content
	beliefContent = append(beliefContent, models.Content{
		RawStr: input.BeliefContent,
	})

	belief := models.Belief{
		UserID: input.UserID,
		Content: beliefContent,
		Version: 0,
	}

	belief.ID = new_uuid

	err := bsvc.kv.Store(input.UserID, new_uuid, &belief)
	if err != nil {
		return nil, err
	}

	belief_system, err := bsvc.getBeliefSystemFromBeliefs(filteredBeliefs)
	if err != nil {
		return nil, err
	}

	return &models.CreateBeliefOutput{
		Belief:       belief,
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
		for _, id := range input.BeliefIDs {
			if storedBelief.ID == id {
				filteredBeliefs = append(filteredBeliefs, *storedBelief)
				break
			}
		}
	}

	belief_system, err := bsvc.getBeliefSystemFromBeliefs(filteredBeliefs)
	if err != nil {
		return nil, err
	}

	return &models.ListBeliefsOutput{
		Beliefs:      filteredBeliefs,
		BeliefSystem: *belief_system, 
	}, nil
}

func (bsvc *BeliefService) getBeliefSystemFromBeliefs(beliefs []models.Belief) (*models.BeliefSystem, error) {
	var belief_strs []string
	for _, belief := beliefs {
		var beliefContent string
		for _, content := belief.Content {
			beliefContent += content.raw_str + "."
		}
		belief_strs = append(belief_strs, beliefContent)
	}

	belief_system_str, err := bsvc.ai.GenerateBeliefSystem(belief_strs)
	if err != nil {
		return nil, err
	}

	return &models.BeliefSystem{
		RawStr: belief_system_str,
	}
}
