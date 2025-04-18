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
	startTime := time.Now()
	var updatedBeliefs []models.Belief

	// Get answered interaction
	answeredInteraction, err := getAnsweredInteraction(event.PreviousInteractions)
	if answeredInteraction == nil {
		return &models.BeliefSystem{}, nil
	}
	if err != nil {
		return nil, err
	}

	// Get interaction event
	interactionEvent, err := getDialecticalInteractionAsEvent(*answeredInteraction)
	if err != nil {
		return nil, err
	}

	// Get belief system once
	bs, err := de.bsvc.GetBeliefSystem(selfModelID)
	if err != nil {
		return nil, err
	}

	// OPTIMIZATION: Process beliefs in batches when possible
	// First, check existing beliefs for updates
	for _, existingBelief := range bs.Beliefs {
		// Use existing AIHelper method but with added context
		shouldUpdate, interpretedBeliefStr, err := de.ai.UpdateBeliefWithInteractionEvent(*interactionEvent, existingBelief.GetContentAsString())
		if err != nil {
			log.Printf("Error in UpdateBeliefWithInteractionEvent: %v", err)
			return nil, err
		}

		if shouldUpdate {
			// Store the interpreted belief as a user belief
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

	// OPTIMIZATION: For new beliefs, get all in a single API call
	if len(updatedBeliefs) == 0 {
		// No existing beliefs were updated, extract new beliefs
		// Use existing AIHelper method but with improved context
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
		// Initialize PredictiveProcessingContext if not present
		if len(beliefSystem.EpistemicContexts) == 0 {
			beliefSystem.EpistemicContexts = []*models.EpistemicContext{
				{
					PredictiveProcessingContext: &models.PredictiveProcessingContext{
						ObservationContexts: []*models.ObservationContext{},
						BeliefContexts:      []*models.BeliefContext{},
					},
				},
			}
		}

		// Update PredictiveProcessingContext with new observations
		de.updatePredictiveProcessingWithInteraction(beliefSystem, interactionEvent, beliefPointers)
	}

	log.Printf("Process completed in %v", time.Since(startTime))
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

// buildConversationContext creates a string representation of recent conversation history
// to provide context for AI operations
func buildConversationContext(interactions []models.DialecticalInteraction, maxHistory int) string {
	if len(interactions) == 0 {
		return ""
	}

	// Limit the number of interactions to consider
	startIdx := 0
	if len(interactions) > maxHistory {
		startIdx = len(interactions) - maxHistory
	}

	var conversationBuilder strings.Builder
	conversationBuilder.WriteString("Conversation context:\n")

	for i := startIdx; i < len(interactions); i++ {
		interaction := interactions[i]
		if interaction.Interaction != nil && interaction.Interaction.QuestionAnswer != nil {
			qa := interaction.Interaction.QuestionAnswer
			conversationBuilder.WriteString(fmt.Sprintf("AI: %s\n", qa.Question.Question))

			if interaction.Status == models.StatusAnswered && qa.Answer.UserAnswer != "" {
				conversationBuilder.WriteString(fmt.Sprintf("User: %s\n", qa.Answer.UserAnswer))
			}
		}
	}

	return conversationBuilder.String()
}

// updatePredictiveProcessingWithInteraction updates the PredictiveProcessingContext with new
// observations and belief contexts based on the current interaction
func (de *DialecticalEpistemology) updatePredictiveProcessingWithInteraction(
	beliefSystem *models.BeliefSystem,
	interactionEvent *ai.InteractionEvent,
	updatedBeliefs []*models.Belief,
) {
	if len(beliefSystem.EpistemicContexts) == 0 ||
		beliefSystem.EpistemicContexts[0].PredictiveProcessingContext == nil {
		return
	}

	ppc := beliefSystem.EpistemicContexts[0].PredictiveProcessingContext

	// Create an observation context for this interaction
	ocID := uuid.New().String()
	oc := &models.ObservationContext{
		ID:             ocID,
		Name:           fmt.Sprintf("Response to '%s'", interactionEvent.Question),
		ParentID:       "",
		PossibleStates: []string{"Positive", "Negative", "Neutral"},
	}
	ppc.ObservationContexts = append(ppc.ObservationContexts, oc)

	// Create belief contexts for each updated belief
	for _, belief := range updatedBeliefs {
		bc := &models.BeliefContext{
			BeliefID:             belief.ID,
			ObservationContextID: ocID,
			ConfidenceRatings: []models.ConfidenceRating{
				{
					ConfidenceScore: 0.8, // Default confidence score
					Default:         true,
				},
			},
			ConditionalProbs:        map[string]float32{},
			DialecticInteractionIDs: []string{},
			EpistemicEmotion:        models.Confirmation, // Default to confirmation
			EmotionIntensity:        0.5,                 // Default intensity
		}
		ppc.BeliefContexts = append(ppc.BeliefContexts, bc)
	}
}
