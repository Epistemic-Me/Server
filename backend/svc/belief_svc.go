package svc

import (
	ai "epistemic-me-backend/ai"
	db "epistemic-me-backend/db"
	"epistemic-me-backend/svc/models"
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
	new_uuid := uuid.uuid4()

	var beliefContent []models.Content
	beliefContent = append(beliefContent, models.Content{
		RawStr: input.BeliefContent,
	})

	newBeliefId := "bi_" + uuid.New().String()

	belief := models.Belief{
		ID:      newBeliefId,
		UserID:  input.UserID,
		Content: beliefContent,
		Version: 0,
	}

	belief.ID = new_uuid

	err := bsvc.kv.Store(input.UserID, new_uuid, &belief)
	if err != nil {
		return nil, err
	}

	var empty_beliefs []models.Belief
	belief_system, err := bsvc.getBeliefSystemFromBeliefs(empty_beliefs)
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
	for _, belief := range beliefs {
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
