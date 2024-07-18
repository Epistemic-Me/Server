package svc

import (
	db "epistemic-me-backend/db"
	"epistemic-me-backend/svc/models"
	models "epistemic-me-backend/svc/models"
	"reflect"
)

type BeliefService struct {
	kv db.KeyValueStore
}

func (bsvc *BeliefService) CreateBelief(input *models.CreateBeliefInput) (*models.CreateBeliefOutput, error) {
	new_uuid := uuid.uuid4()

	belief := input.Belief
	belief.ID = new_uuid

	err := bsvc.kv.Store(input.UserID, new_uuid, &belief)
	if err != nil {
		return nil, err
	}

	return &models.CreateBeliefOutput{
		Belief:       belief,
		BeliefSystem: nil, // TODO: (@deen) add materialization of belief system to belief endpoints
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

	// Return the filtered beliefs in the output
	return &models.ListBeliefsOutput{
		Beliefs:      filteredBeliefs,
		BeliefSystem: nil, // TODO: (@deen) add materialization of belief system to belief endpoints
	}, nil
}
