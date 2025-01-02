package svc

import (
	ai "epistemic-me-core/ai"
	"epistemic-me-core/svc/models"
	"fmt"
	"log"
	"strings"
	"time"
	// Make sure to import your models package
)

// Epistemology defines an interface for handling and evaluating beliefs through structured epistemological processes.
// Higher level processes like dialectics can orchestrate epistemologies to produce new and/or updated beliefs.
type EpistemologyService interface {

	// ProcessInformation accepts new data from an external environment and updates a belief system.
	// The event parameter can be of any type.
	Process(bs *models.BeliefSystem, event interface{}, dryRun bool) (*models.BeliefSystem, error)

	// Initiate Request, from a given event creates a hook to request more information
	RequestInformationFromEvent(bs *models.BeliefSystem, event interface{}) (request interface{}, error error)
}

type PredictiveEpistemologyService interface {

	// Given a sample request for information, predict what the user will respond with
	PredictFutureEvent(bs *models.BeliefSystem, request *interface{}) (event interface{}, error error)
}

// PredictiveProcessing defines methods to handle and validate belief systems and beliefs
// based on the principles of predictive processing epistemology.
type DialecticalEpistemology struct {
	bsvc                       *BeliefService
	ai                         *ai.AIHelper
	enablePredictiveProcessing bool
}

func NewDialecticEpistemology(beliefService *BeliefService, ai *ai.AIHelper) *DialecticalEpistemology {
	return &DialecticalEpistemology{
		bsvc:                       beliefService,
		ai:                         ai,
		enablePredictiveProcessing: true,
	}
}

func (de *DialecticalEpistemology) Process(bs *models.BeliefSystem, event *models.DialecticEvent, dryRun bool, selfModelID string) (*models.BeliefSystem, error) {
	var updatedBeliefs []models.Belief

	answeredInteraction, err := getAnsweredInteraction(event.PreviousInteractions)
	if err != nil {
		return nil, err
	}

	interactionEvent, err := getDialecticalInteractionAsEvent(*answeredInteraction)
	if err != nil {
		return nil, err
	}

	// for each of the user beliefs, check to see if event has relevance and update accordingly
	for _, existingBelief := range bs.Beliefs {

		shouldUpdate, interpretedBeliefStr, err := de.ai.UpdateBeliefWithInteractionEvent(*interactionEvent, existingBelief.GetContentAsString())
		if err != nil {
			log.Printf("Error in UpdateBeliefWithInteractionEvent: %v", err)
			return nil, err
		}

		if shouldUpdate {
			// store the interpeted belief as a user belief so it will be included in the belief system
			updatedBeliefOutput, err := de.bsvc.UpdateBelief(&models.UpdateBeliefInput{
				SelfModelID:          selfModelID,
				ID:                   existingBelief.ID,
				CurrentVersion:       existingBelief.Version,
				UpdatedBeliefContent: interpretedBeliefStr,
				BeliefType:           models.Statement,
				DryRun:               dryRun,
			})

			if err != nil {
				log.Printf("Error in UpdateBelief: %v", err)
				return nil, err
			}

			updatedBeliefs = append(updatedBeliefs, updatedBeliefOutput.Belief)
		}
	}

	// if we've updated no existing beleifs, create a new one
	// todo: @deen this may need to become more sophisticated in the future
	if len(updatedBeliefs) == 0 {
		interpretedBeliefStr, err := de.ai.GetInteractionEventAsBelief(*interactionEvent)
		if err != nil {
			return nil, err
		}

		// store the interpeted belief as a user belief so it will be included in the belief system
		createBeliefOutput, err := de.bsvc.CreateBelief(&models.CreateBeliefInput{
			SelfModelID:   selfModelID,
			BeliefContent: interpretedBeliefStr,
			DryRun:        dryRun,
		})

		if err != nil {
			log.Printf("Error in CreateBelief: %v", err)
			return nil, err
		}

		updatedBeliefs = append(updatedBeliefs, createBeliefOutput.Belief)
	}

	beliefPointers := make([]*models.Belief, len(updatedBeliefs))
	for i := range updatedBeliefs {
		beliefPointers[i] = &updatedBeliefs[i]
	}

	beliefSystem, err := de.bsvc.GetBeliefSystemFromBeliefs(beliefPointers)
	if err != nil {
		return nil, err
	}

	if de.enablePredictiveProcessing {
		beliefSystem.EpistemicContexts = []*models.EpistemicContext{
			{
				PredictiveProcessingContext: &models.PredictiveProcessingContext{
					ObservationContexts: []*models.ObservationContext{},
					BeliefContexts:      []*models.BeliefContext{},
				},
			},
		}
	}

	return beliefSystem, nil
}

func (de *DialecticalEpistemology) Respond(bs *models.BeliefSystem, event *models.DialecticEvent) (*models.DialecticRequest, error) {

	var customQuestion *string
	pendingInteraction, err := getPendingInteraction(event.PreviousInteractions)
	if err != nil {
		return nil, err
	}

	if pendingInteraction != nil {
		customQuestion = &pendingInteraction.Question.Question
	}

	nextInteraction, err := de.generatePendingDialecticalInteraction(event.PreviousInteractions, bs, customQuestion)
	if err != nil {
		return nil, err
	}

	return &models.DialecticRequest{
		NewInteraction: nextInteraction,
	}, nil
}

func (de *DialecticalEpistemology) PredictFutureEvent(bs *models.BeliefSystem, request *models.DialecticRequest) (event *models.DialecticEvent, error error) {
	return nil, nil
}

func getPendingInteraction(UserInteractions []models.DialecticalInteraction) (*models.DialecticalInteraction, error) {
	// Get the latest interaction
	if len(UserInteractions) == 0 {
		log.Printf("No interactions found in the dialectic")
		return nil, fmt.Errorf("no interactions found in the dialectic")
	}
	latestInteraction := UserInteractions[len(UserInteractions)-1]
	// Check if the latest interaction is pending
	if latestInteraction.Status != models.StatusPendingAnswer {
		log.Printf("Latest interaction is not pending")
		return nil, fmt.Errorf("latest interaction is not pending")
	}
	return &latestInteraction, nil
}

func getAnsweredInteraction(UserInteractions []models.DialecticalInteraction) (*models.DialecticalInteraction, error) {
	// Get the latest interaction
	if len(UserInteractions) == 0 {
		log.Printf("No interactions found in the dialectic")
		return nil, fmt.Errorf("no interactions found in the dialectic")
	}
	latestInteraction := UserInteractions[len(UserInteractions)-1]
	// Check if the latest interaction is pending
	if latestInteraction.Status != models.StatusAnswered {
		log.Printf("Latest interaction is not pending")
		return nil, fmt.Errorf("latest interaction is not pending")
	}
	return &latestInteraction, nil
}

func getDialecticalInteractionAsEvent(interaction models.DialecticalInteraction) (*ai.InteractionEvent, error) {
	log.Printf("getDialecticalInteractionAsEvent called with interaction status: %v", interaction.Status)
	if interaction.Status != models.StatusAnswered {
		log.Printf("Interaction is not answered yet")
		return nil, fmt.Errorf("interaction is not answered yet")
	}
	return &ai.InteractionEvent{
		Question: interaction.Question.Question,
		Answer:   interaction.UserAnswer.UserAnswer,
	}, nil
}

func (de *DialecticalEpistemology) generatePendingDialecticalInteraction(previousInteractions []models.DialecticalInteraction, userBeliefSystem *models.BeliefSystem, customQuestion *string) (*models.DialecticalInteraction, error) {
	var events []ai.InteractionEvent
	for _, interaction := range previousInteractions {
		if interaction.Status == models.StatusAnswered {
			interactionEvent, err := getDialecticalInteractionAsEvent(interaction)
			if err != nil {
				log.Printf("Error in getDialecticalInteractionAsEvent: %v", err)
				return nil, err
			}
			events = append(events, *interactionEvent)
		}
	}

	beliefStrings := make([]string, len(userBeliefSystem.Beliefs))
	for i, belief := range userBeliefSystem.Beliefs {
		beliefStrings[i] = belief.GetContentAsString()
	}

	var question string
	var err error

	if customQuestion != nil {
		question = *customQuestion
	} else {
		question, err = de.ai.GenerateQuestion(strings.Join(beliefStrings, " "), events)
		if err != nil {
			log.Printf("Error in GenerateQuestion: %v", err)
			return nil, err
		}
	}

	// Create the interaction with QuestionAnswer initialized
	interaction := &models.DialecticalInteraction{
		Status: models.StatusPendingAnswer,
		Type:   models.InteractionTypeQuestionAnswer,
		Question: models.Question{
			Question:           question,
			CreatedAtMillisUTC: time.Now().UnixMilli(),
		},
		Interaction: &models.QuestionAnswerInteraction{
			Question: models.Question{
				Question:           question,
				CreatedAtMillisUTC: time.Now().UnixMilli(),
			},
		},
	}

	return interaction, nil
}
