package svc

import (
	ai "epistemic-me-core/ai"
	db "epistemic-me-core/db"
	"epistemic-me-core/svc/models"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
)

type DialecticService struct {
	kvStore *db.KeyValueStore
	bsvc    *BeliefService
	aih     *ai.AIHelper
}

// NewDialecticService initializes and returns a new DialecticService.
func NewDialecticService(kvStore *db.KeyValueStore, bsvc *BeliefService, aih *ai.AIHelper) *DialecticService {
	return &DialecticService{
		kvStore: kvStore,
		bsvc:    bsvc,
		aih:     aih,
	}
}

// Add this method to DialecticService
func (dsvc *DialecticService) storeDialecticValue(selfModelID string, dialectic *models.Dialectic) error {
	log.Printf("Storing dialectic: %+v", dialectic)
	return dsvc.kvStore.Store(selfModelID, dialectic.ID, *dialectic, len(dialectic.UserInteractions))
}

// Add this method to DialecticService
func (dsvc *DialecticService) retrieveDialecticValue(selfModelID, dialecticID string) (*models.Dialectic, error) {
	value, err := dsvc.kvStore.Retrieve(selfModelID, dialecticID)
	if err != nil {
		return nil, err
	}
	log.Printf("Retrieved dialectic value: %+v", value)
	switch d := value.(type) {
	case models.Dialectic:
		return &d, nil
	case *models.Dialectic:
		return d, nil
	default:
		return nil, fmt.Errorf("retrieved value is not a Dialectic: %T", value)
	}
}

func (dsvc *DialecticService) CreateDialectic(input *models.CreateDialecticInput) (*models.CreateDialecticOutput, error) {
	newDialecticId := "di_" + uuid.New().String()

	dialectic := &models.Dialectic{
		ID:          newDialecticId,
		SelfModelID: input.SelfModelID, // Changed from UserID
		Agent: models.Agent{
			AgentType:     models.AgentTypeGPTLatest,
			DialecticType: input.DialecticType,
		},
		UserInteractions: []models.DialecticalInteraction{},
	}

	// Generate the first interaction
	newInteraction, err := dsvc.generatePendingDialecticalInteraction(input.SelfModelID, dialectic.UserInteractions, models.BeliefSystem{}, nil)
	if err != nil {
		return nil, err
	}

	dialectic.UserInteractions = append(dialectic.UserInteractions, *newInteraction)

	err = dsvc.storeDialecticValue(input.SelfModelID, dialectic)
	if err != nil {
		return nil, fmt.Errorf("failed to store new dialectic: %w", err)
	}

	return &models.CreateDialecticOutput{
		DialecticID: newDialecticId,
		Dialectic:   *dialectic,
	}, nil
}

func (dsvc *DialecticService) ListDialectics(input *models.ListDialecticsInput) (*models.ListDialecticsOutput, error) {
	dialectics, err := dsvc.kvStore.ListByType(input.SelfModelID, reflect.TypeOf(models.Dialectic{}))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve dialectics: %v", err)
	}

	var dialecticValues []models.Dialectic
	for _, d := range dialectics {
		if dialectic, ok := d.(*models.Dialectic); ok && dialectic.SelfModelID == input.SelfModelID {
			dialecticValues = append(dialecticValues, *dialectic)
		}
	}

	return &models.ListDialecticsOutput{
		Dialectics: dialecticValues,
	}, nil
}

func (dsvc *DialecticService) UpdateDialectic(input *models.UpdateDialecticInput) (*models.UpdateDialecticOutput, error) {
	dialectic, err := dsvc.retrieveDialecticValue(input.SelfModelID, input.ID)
	if err != nil {
		return nil, err
	}
	log.Printf("Retrieved dialectic with %d interactions", len(dialectic.UserInteractions))

	// Handle question blob (from assistant)
	if input.QuestionBlob != "" {
		// Extract potential questions from the blob using AI
		questions, err := dsvc.aih.ExtractQuestionsFromText(input.QuestionBlob)
		if err != nil {
			return nil, fmt.Errorf("failed to extract questions: %w", err)
		}

		// Create new interactions for each question
		for _, q := range questions {
			// Skip if question is empty
			if q == "" {
				log.Printf("Skipping empty question")
				continue
			}

			interaction := models.DialecticalInteraction{
				ID:     uuid.New().String(),
				Status: models.StatusPendingAnswer,
				Type:   models.InteractionTypeQuestionAnswer,
				Question: models.Question{
					Question:           q,
					CreatedAtMillisUTC: time.Now().UnixMilli(),
				},
				Interaction: &models.QuestionAnswerInteraction{
					Question: models.Question{
						Question:           q,
						CreatedAtMillisUTC: time.Now().UnixMilli(),
					},
				},
			}

			// Add predicted observation
			predictedObs, err := dsvc.generatePredictedObservation(&interaction)
			if err != nil {
				return nil, fmt.Errorf("failed to generate predicted observation: %w", err)
			}
			interaction.PredictedObservation = predictedObs

			dialectic.UserInteractions = append(dialectic.UserInteractions, interaction)
		}
	}

	// Handle answer blob (from user)
	if input.AnswerBlob != "" {
		// Find all pending questions
		var pendingIndices []int
		for i, interaction := range dialectic.UserInteractions {
			if interaction.Status == models.StatusPendingAnswer {
				pendingIndices = append(pendingIndices, i)
			}
		}

		if len(pendingIndices) > 0 {
			// Try to match answers to questions using AI
			matches, err := dsvc.aih.MatchAnswersToQuestions(
				input.AnswerBlob,
				getPendingQuestions(dialectic.UserInteractions, pendingIndices),
			)
			if err != nil {
				return nil, fmt.Errorf("failed to match answers: %w", err)
			}

			// Update matched interactions
			for questionIdx, answer := range matches {
				// Skip if answer is empty
				if answer == "" {
					log.Printf("Skipping empty answer for question index %d", questionIdx)
					continue
				}

				idx := pendingIndices[questionIdx]

				// Validate the existing question is not empty
				if dialectic.UserInteractions[idx].Question.Question == "" {
					log.Printf("Skipping interaction with empty question at index %d", idx)
					continue
				}

				// Create the QuestionAnswer interaction
				qa := &models.QuestionAnswerInteraction{
					Question: dialectic.UserInteractions[idx].Question,
					Answer: models.UserAnswer{
						UserAnswer:         answer,
						CreatedAtMillisUTC: time.Now().UnixMilli(),
					},
				}

				// Double-check both question and answer
				if qa.Question.Question == "" || qa.Answer.UserAnswer == "" {
					log.Printf("Skipping interaction - missing question or answer. Question: %q, Answer: %q",
						qa.Question.Question, qa.Answer.UserAnswer)
					continue
				}

				log.Printf("Created QuestionAnswer interaction with Question: %q, Answer: %q",
					qa.Question.Question, qa.Answer.UserAnswer)

				// Extract beliefs for this Q&A pair
				interactionEvent := ai.InteractionEvent{
					Question: qa.Question.Question,
					Answer:   answer,
				}

				extractedBeliefStr, err := dsvc.aih.GetInteractionEventAsBelief(interactionEvent)
				if err != nil {
					return nil, fmt.Errorf("failed to extract belief: %w", err)
				}

				extractedBelief := &models.Belief{
					ID:      uuid.New().String(),
					Content: []models.Content{{RawStr: extractedBeliefStr}},
					Type:    models.Statement,
				}

				log.Printf("Extracted belief: %+v", extractedBelief)

				qa.ExtractedBeliefs = append(qa.ExtractedBeliefs, extractedBelief)
				log.Printf("QuestionAnswer interaction after adding belief: %+v", qa)

				dialectic.UserInteractions[idx].Type = models.InteractionTypeQuestionAnswer
				dialectic.UserInteractions[idx].Interaction = qa
				dialectic.UserInteractions[idx].Status = models.StatusAnswered

				// Create action and observation
				action := &models.Action{
					ID:                     uuid.New().String(),
					Type:                   models.ActionTypeAnswerQuestion,
					DialecticInteractionID: dialectic.UserInteractions[idx].ID,
					Timestamp:              time.Now().UnixMilli(),
				}

				observation, err := dsvc.ExecuteAction(action, &dialectic.UserInteractions[idx])
				if err != nil {
					return nil, fmt.Errorf("failed to execute action: %w", err)
				}

				// Update the interaction
				dialectic.UserInteractions[idx].Action = action
				dialectic.UserInteractions[idx].Observation = observation

				log.Printf("Created QuestionAnswer interaction with Question: %q, Answer: %q", qa.Question.Question, qa.Answer.UserAnswer)
				log.Printf("Extracted belief: %+v", extractedBelief)
				log.Printf("QuestionAnswer interaction after adding belief: %+v", qa)
			}
		}
	}

	if !input.DryRun {
		err = dsvc.storeDialecticValue(input.SelfModelID, dialectic)
		if err != nil {
			return nil, err
		}
	}
	log.Printf("Storing dialectic with %d interactions", len(dialectic.UserInteractions))

	return &models.UpdateDialecticOutput{
		Dialectic: *dialectic,
	}, nil
}

// Helper function to get questions from pending interactions
func getPendingQuestions(interactions []models.DialecticalInteraction, indices []int) []string {
	questions := make([]string, len(indices))
	for i, idx := range indices {
		questions[i] = interactions[idx].Question.Question
	}
	return questions
}

func (dsvc *DialecticService) updateBeliefSystemForInteraction(interactionEvent ai.InteractionEvent, selfModelID string, dryRun bool) (*models.BeliefSystem, error) {
	// Try to get existing beliefs
	listBeliefsOutput, err := dsvc.bsvc.ListBeliefs(&models.ListBeliefsInput{
		SelfModelID: selfModelID,
	})

	// Initialize an empty belief system if none exists
	if err != nil && strings.Contains(err.Error(), "key not found") {
		// Create initial belief from the interaction
		interpretedBeliefStr, err := dsvc.aih.GetInteractionEventAsBelief(interactionEvent)
		if err != nil {
			return nil, err
		}

		createBeliefOutput, err := dsvc.bsvc.CreateBelief(&models.CreateBeliefInput{
			SelfModelID:   selfModelID,
			BeliefContent: interpretedBeliefStr,
			DryRun:        dryRun,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to create initial belief: %w", err)
		}

		// Return the new belief system
		return &models.BeliefSystem{
			Beliefs:             []*models.Belief{&createBeliefOutput.Belief},
			ObservationContexts: []*models.ObservationContext{},
			BeliefContexts:      []*models.BeliefContext{},
		}, nil
	} else if err != nil {
		return nil, fmt.Errorf("error retrieving beliefs: %w", err)
	}

	var updatedBeliefs []models.Belief

	// for each of the user beliefs, check to see if event has relevance and update accordingly
	for _, existingBelief := range listBeliefsOutput.Beliefs {

		shouldUpdate, interpretedBeliefStr, err := dsvc.aih.UpdateBeliefWithInteractionEvent(interactionEvent, existingBelief.GetContentAsString())
		if err != nil {
			log.Printf("Error in UpdateBeliefWithInteractionEvent: %v", err)
			return nil, err
		}

		if shouldUpdate {
			// store the interpeted belief as a user belief so it will be included in the belief system
			updatedBeliefOutput, err := dsvc.bsvc.UpdateBelief(&models.UpdateBeliefInput{
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
		interpretedBeliefStr, err := dsvc.aih.GetInteractionEventAsBelief(interactionEvent)
		if err != nil {
			return nil, err
		}

		// store the interpeted belief as a user belief so it will be included in the belief system
		createBeliefOutput, err := dsvc.bsvc.CreateBelief(&models.CreateBeliefInput{
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
	beliefSystem, err := dsvc.bsvc.getBeliefSystemFromBeliefs(beliefPointers)
	if err != nil {
		return nil, err
	}

	return beliefSystem, nil
}

func (dsvc *DialecticService) generatePendingDialecticalInteraction(selfModelID string, previousInteractions []models.DialecticalInteraction, userBeliefSystem models.BeliefSystem, customQuestion *string) (*models.DialecticalInteraction, error) {
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
		question, err = dsvc.aih.GenerateQuestion(strings.Join(beliefStrings, " "), events)
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

func getPendingInteraction(dialectic models.Dialectic) (*models.DialecticalInteraction, error) {
	// Get the latest interaction
	if len(dialectic.UserInteractions) == 0 {
		log.Printf("No interactions found in the dialectic")
		return nil, fmt.Errorf("no interactions found in the dialectic")
	}
	latestInteraction := dialectic.UserInteractions[len(dialectic.UserInteractions)-1]
	// Check if the latest interaction is pending
	if latestInteraction.Status != models.StatusPendingAnswer {
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

func determineDialecticStrategy(dialecticType models.DialecticType) ai.DialecticStrategy {
	switch dialecticType {
	case models.DialecticTypeSleepDietExercise:
		return ai.StrategySleepDietExercise
	// Add more cases for other dialectic types
	default:
		return ai.StrategyDefault
	}
}

// ExecuteAction performs an action and produces an observation
func (dsvc *DialecticService) ExecuteAction(action *models.Action, interaction *models.DialecticalInteraction, answer ...string) (*models.Observation, error) {
	// beliefContext, err := dsvc.getBeliefContextFromInteraction(interaction)
	// if err != nil {
	//  return nil, fmt.Errorf("failed to get belief context: %w", err)
	// }
	// Set action ID if not already set
	if action.ID == "" {
		action.ID = uuid.New().String()
	}

	// Generate resource based on action type
	var resource *models.Resource
	switch action.Type {
	case models.ActionTypeAnswerQuestion:
		resource = &models.Resource{
			ID:       uuid.New().String(),
			Type:     models.ResourceTypeChatLog,
			Content:  interaction.UserAnswer.UserAnswer,
			SourceID: interaction.ID,
			ActionID: action.ID,
			Metadata: map[string]string{"interaction_type": "survey"},
		}
	case models.ActionTypeCollectEvidence:
		resource = &models.Resource{
			ID:       uuid.New().String(),
			Type:     models.ResourceTypeScientificPaper,
			Content:  interaction.Question.Question,
			SourceID: interaction.ID,
			ActionID: action.ID,
			Metadata: map[string]string{"interaction_type": "evidence"},
		}
	case models.ActionTypeActuateOutcome:
		resource = &models.Resource{
			ID:       uuid.New().String(),
			Type:     models.ResourceTypeMeasurementData,
			Content:  interaction.Question.Question,
			SourceID: interaction.ID,
			ActionID: action.ID,
			Metadata: map[string]string{"interaction_type": "outcome"},
		}
	default:
		return nil, fmt.Errorf("invalid action type: %v", action.Type)
	}

	// Use provided answer if available, otherwise use existing interpretation
	stateDistribution := map[string]float32{}
	if len(answer) > 0 && answer[0] != "" {
		stateDistribution[answer[0]] = 1.0
	} else {
		stateDistribution[dsvc.interpretResourceAsState(resource, nil)] = 1.0
	}

	observation := &models.Observation{
		DialecticInteractionID: action.DialecticInteractionID,
		Type:                   models.Answer,
		Resource:               resource,
		StateDistribution:      stateDistribution,
		Timestamp:              time.Now().UnixMilli(),
	}

	return observation, nil
}

// Add missing methods
func (dsvc *DialecticService) getBeliefContextFromInteraction(interaction *models.DialecticalInteraction) (*models.BeliefContext, error) {
	// Implementation
	return nil, fmt.Errorf("not implemented")
}

func (dsvc *DialecticService) interpretResourceAsState(resource *models.Resource, context *models.BeliefContext) string {
	return resource.Content
}

// generatePredictedObservation creates a predicted observation for a dialectical interaction
func (dsvc *DialecticService) generatePredictedObservation(interaction *models.DialecticalInteraction) (*models.Observation, error) {
	if interaction == nil {
		return nil, fmt.Errorf("interaction cannot be nil")
	}

	predictedAnswer, err := dsvc.aih.PredictAnswer(interaction.Question.Question)
	if err != nil {
		return nil, fmt.Errorf("failed to predict answer: %w", err)
	}

	return &models.Observation{
		DialecticInteractionID: interaction.ID,
		Type:                   models.Answer,
		StateDistribution:      map[string]float32{predictedAnswer: 1.0},
		Timestamp:              time.Now().UnixMilli(),
	}, nil
}

// handleQuestionAnswerInteraction processes a question-answer interaction
func (dsvc *DialecticService) handleQuestionAnswerInteraction(interaction *models.DialecticalInteraction, selfModelID string) (*models.Observation, error) {
	if interaction == nil {
		return nil, fmt.Errorf("interaction cannot be nil")
	}

	if selfModelID == "" {
		return nil, fmt.Errorf("selfModelID cannot be empty")
	}

	predictedObs, err := dsvc.generatePredictedObservation(interaction)
	if err != nil {
		return nil, fmt.Errorf("failed to generate predicted observation: %w", err)
	}

	interaction.PredictedObservation = predictedObs

	action := &models.Action{
		ID:                     uuid.New().String(),
		Type:                   models.ActionTypeAnswerQuestion,
		DialecticInteractionID: interaction.ID,
		Timestamp:              time.Now().UnixMilli(),
	}

	observation, err := dsvc.ExecuteAction(action, interaction)
	if err != nil {
		return nil, fmt.Errorf("failed to execute action: %w", err)
	}

	return observation, nil
}

func (dsvc *DialecticService) MatchAnswerToQuestion(question, potentialAnswer string) (bool, error) {
	// Use AI helper to determine if the answer matches the question
	return dsvc.aih.IsAnswerToQuestion(question, potentialAnswer)
}
