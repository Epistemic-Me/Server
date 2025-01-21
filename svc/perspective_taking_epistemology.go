package svc

import (
	ai "epistemic-me-core/ai"
	"epistemic-me-core/svc/models"
	"errors"
)

type PerspectiveTakingEpistemology struct {
	bsvc                       *BeliefService
	ai                         *ai.AIHelper
	enablePredictiveProcessing bool
}

func NewPerspectiveTakingEpistemology(beliefService *BeliefService, ai *ai.AIHelper) *PerspectiveTakingEpistemology {
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

	newBeliefsStrings, err := de.ai.ExtractBeliefsFromResource(event.Resource)
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

	return bs, nil
}

func (pte *PerspectiveTakingEpistemology) Respond(bs *models.BeliefSystem, request models.EpistemicRequest) (perspective *string, error error) {

	content := request.Content

	question, ok := content["question"].(string)
	if !ok {
		return nil, errors.New("Error: no question provided")
	}

	answer, ok := content["answer"].(string)
	if !ok {
		return nil, errors.New("Error: no answer provided")
	}

	listBeliefsResponse, err := pte.bsvc.ListBeliefs(&models.ListBeliefsInput{
		SelfModelID: request.SelfModelID,
	})
	if err != nil {
		return nil, err
	}

	beliefs := listBeliefsResponse.BeliefSystem.Beliefs
	if err != nil {
		return nil, err
	}

	beliefStrings := make([]string, 0)
	for _, belief := range beliefs {
		beliefStrings = append(beliefStrings, belief.Content[0].RawStr)
	}

	beliefSystem, err := pte.ai.GenerateBeliefSystem(beliefStrings)
	if err != nil {
		return nil, err
	}

	// After extracting the latest belief system, provide a perspective on the question and
	// answer currently requesting a given perspective
	response, err := pte.ai.ProvidePerspectiveOnQuestionAndAnswer(question, answer, beliefSystem)
	if err != nil {
		return nil, err
	}

	return &response, err
}
