package svc

import (
	ai "epistemic-me-core/ai"
	"epistemic-me-core/svc/models"
)

type PerspectiveTakingEpistemology struct {
	bsvc                       *BeliefService
	ai                         *ai.AIHelper
	enablePredictiveProcessing bool
}

func NewPerspectiveTakingcEpistemology(beliefService *BeliefService, ai *ai.AIHelper) *PerspectiveTakingEpistemology {
	return &PerspectiveTakingEpistemology{
		bsvc:                       beliefService,
		ai:                         ai,
		enablePredictiveProcessing: true,
	}
}

func (de *PerspectiveTakingEpistemology) Process(event *models.PerspectiveTakingEpistemicEvent, dryRun bool, selfModelID string) (*models.BeliefSystem, error) {

	bs, err := de.bsvc.GetBeliefSystem(selfModelID)
	if err != nil {
		return nil, err
	}

	newBeliefsStrings, err := de.ai.ExtractBeleifsFromResource(event.Resource)
	if err != nil {
		return nil, err
	}

	newBeliefs := make([]*models.Belief, 0)
	for _, newBeliefString := range newBeliefsStrings {
		beliefOutput, err := de.bsvc.CreateBelief(&models.CreateBeliefInput{
			SelfModelID:   selfModelID,
			BeliefContent: newBeliefString,
		})
		if err != nil {
			return nil, err
		}
		newBeliefs = append(newBeliefs, &beliefOutput.Belief)
	}

	_, beliefIdsToRemove, err := de.ai.DetermineBeliefValidity(bs.Beliefs, newBeliefs)
	if err != nil {
		return nil, err
	}

	for index, beliefIDToRemove := range beliefIdsToRemove {
		lastItem := index == len(beliefIDToRemove)-1

		deletionOutput, err := de.bsvc.DeleteBelief(&models.DeleteBeliefInput{
			ID:                  beliefIDToRemove,
			SelfModelID:         selfModelID,
			DryRun:              false,
			ComputeBeliefSystem: lastItem,
		})
		if err != nil {
			return nil, err
		}

		if lastItem {
			bs = &deletionOutput.BeliefSystem
		}
	}

	// todo: @deen add epistemic contexts for new beleifs to the beleif system and persist

	return bs, nil
}
