package svc

import (
	ai "epistemic-me-core/ai"
	"epistemic-me-core/svc/models"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	// Make sure to import your models package
)

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

func (de *DialecticalEpistemology) Process(event *models.DialecticEvent, dryRun bool, selfModelID string) (*models.BeliefSystem, error) {
	var updatedBeliefs []models.Belief

	answeredInteraction, err := getAnsweredInteraction(event.PreviousInteractions)
	if answeredInteraction == nil {
		return &models.BeliefSystem{}, nil
	}
	if err != nil {
		return nil, err
	}

	interactionEvent, err := getDialecticalInteractionAsEvent(*answeredInteraction)
	if err != nil {
		return nil, err
	}

	bs, err := de.bsvc.GetBeliefSystem(selfModelID)
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
		interpretedBeliefStrings, err := de.ai.GetInteractionEventAsBelief(*interactionEvent)
		if err != nil {
			return nil, err
		}

		// Create a new belief for each extracted belief string
		for _, beliefStr := range interpretedBeliefStrings {
			createBeliefOutput, err := de.bsvc.CreateBelief(&models.CreateBeliefInput{
				SelfModelID:   selfModelID,
				BeliefContent: beliefStr,
				DryRun:        dryRun,
			})

			if err != nil {
				log.Printf("Error in CreateBelief: %v", err)
				return nil, err
			}

			updatedBeliefs = append(updatedBeliefs, createBeliefOutput.Belief)
		}
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

func (de *DialecticalEpistemology) Respond(bs *models.BeliefSystem, event *models.DialecticEvent, answer string) (*models.DialecticResponse, error) {
	var response *models.DialecticResponse
	var err error

	var customQuestion *string
	pendingInteraction, i := getPendingInteraction(event.PreviousInteractions)

	if pendingInteraction != nil && pendingInteraction.Interaction != nil {
		if qa := pendingInteraction.Interaction.QuestionAnswer; qa != nil {
			qa.Answer.UserAnswer = answer
			pendingInteraction.Status = models.StatusAnswered
			// remove and update the interaction at the correct index
			event.PreviousInteractions = append(event.PreviousInteractions[:i], event.PreviousInteractions[i+1:]...)
			event.PreviousInteractions = append(event.PreviousInteractions, *pendingInteraction)
		}
	}

	nextInteraction, interactionErr := de.generatePendingDialecticalInteraction(event.PreviousInteractions, bs, customQuestion)
	if interactionErr != nil {
		err = interactionErr
	} else {
		response = &models.DialecticResponse{
			SelfModelID:          event.SelfModelID,
			PreviousInteractions: event.PreviousInteractions,
			NewInteraction:       nextInteraction,
		}
	}

	return response, err
}

func getPendingInteraction(UserInteractions []models.DialecticalInteraction) (*models.DialecticalInteraction, int) {
	if len(UserInteractions) == 0 {
		log.Printf("No interactions found in the dialectic")
		return nil, -1
	}
	i := len(UserInteractions) - 1
	latestInteraction := UserInteractions[i]
	if latestInteraction.Status != models.StatusPendingAnswer {
		log.Printf("Latest interaction is not pending")
		return nil, -1
	}
	return &latestInteraction, i
}

func getAnsweredInteraction(UserInteractions []models.DialecticalInteraction) (*models.DialecticalInteraction, error) {
	// Get the latest interaction
	if len(UserInteractions) == 0 {
		log.Printf("No interactions found in the dialectic")
		return nil, nil
	}
	latestInteraction := UserInteractions[len(UserInteractions)-1]
	// Check if the latest interaction is pending
	if latestInteraction.Status != models.StatusAnswered {
		log.Printf("Latest interaction is not answered")
		return nil, fmt.Errorf("latest interaction is not answered")
	}
	return &latestInteraction, nil
}

func getDialecticalInteractionAsEvent(interaction models.DialecticalInteraction) (*ai.InteractionEvent, error) {
	log.Printf("getDialecticalInteractionAsEvent called with interaction status: %v", interaction.Status)
	if interaction.Status != models.StatusAnswered {
		log.Printf("Interaction is not answered yet")
		return nil, fmt.Errorf("interaction is not answered yet")
	}
	qa := interaction.Interaction.QuestionAnswer
	if qa == nil {
		return nil, fmt.Errorf("interaction is not a QuestionAnswer type")
	}
	return &ai.InteractionEvent{
		Question: qa.Question.Question,
		Answer:   qa.Answer.UserAnswer,
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
	newInteraction := &models.DialecticalInteraction{
		ID:     uuid.New().String(),
		Status: models.StatusPendingAnswer,
		Type:   models.InteractionTypeQuestionAnswer,
		Interaction: &models.InteractionData{
			QuestionAnswer: &models.QuestionAnswerInteraction{
				Question: models.Question{
					Question:           question,
					CreatedAtMillisUTC: time.Now().UnixMilli(),
				},
			},
		},
		UpdatedAtMillisUTC: time.Now().UnixMilli(),
	}

	return newInteraction, nil
}
