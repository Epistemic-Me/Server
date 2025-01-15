package svc

import (
	ai "epistemic-me-core/ai"
	db "epistemic-me-core/db"
	"epistemic-me-core/svc/models"
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/google/uuid"
)

type DialecticService struct {
	kvStore *db.KeyValueStore
	aih     *ai.AIHelper
	// deen: todo refactor to use epistemology interface
	epistemology *DialecticalEpistemology
}

// NewDialecticService initializes and returns a new DialecticService.
func NewDialecticService(kvStore *db.KeyValueStore, aih *ai.AIHelper, epistemology *DialecticalEpistemology) *DialecticService {
	return &DialecticService{
		kvStore:      kvStore,
		aih:          aih,
		epistemology: epistemology,
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
	response, err := dsvc.epistemology.Respond(&models.BeliefSystem{}, &models.DialecticEvent{
		PreviousInteractions: dialectic.UserInteractions,
	}, "")
	if err != nil {
		return nil, err
	}

	dialectic.UserInteractions = append(dialectic.UserInteractions, *response.NewInteraction)

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

	if input.Answer.UserAnswer != "" {
		bs, err := dsvc.epistemology.Process(&models.DialecticEvent{
			PreviousInteractions: dialectic.UserInteractions,
		}, input.DryRun, input.SelfModelID)
		if err != nil {
			return nil, err
		}

		// Extract belief from the answer
		interactionEvent := ai.InteractionEvent{
			Question: getQuestion(&dialectic.UserInteractions[len(dialectic.UserInteractions)-1]),
			Answer:   input.Answer.UserAnswer,
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

		// Update the last interaction with the answer and extracted belief
		lastIdx := len(dialectic.UserInteractions) - 1
		oldQA := getQuestionAnswer(dialectic.UserInteractions[lastIdx].Interaction)
		qa := &models.QuestionAnswerInteraction{
			Question: oldQA.Question,
			Answer: models.UserAnswer{
				UserAnswer:         input.Answer.UserAnswer,
				CreatedAtMillisUTC: time.Now().UnixMilli(),
			},
			ExtractedBeliefs:   []*models.Belief{extractedBelief},
			UpdatedAtMillisUTC: time.Now().UnixMilli(),
		}

		dialectic.UserInteractions[lastIdx].Status = models.StatusAnswered
		dialectic.UserInteractions[lastIdx].Type = models.InteractionTypeQuestionAnswer
		dialectic.UserInteractions[lastIdx].Interaction = &models.InteractionData{
			QuestionAnswer: qa,
		}
		dialectic.UserInteractions[lastIdx].UpdatedAtMillisUTC = time.Now().UnixMilli()

		// Generate the next interaction
		response, err := dsvc.epistemology.Respond(bs, &models.DialecticEvent{
			PreviousInteractions: dialectic.UserInteractions,
		}, input.Answer.UserAnswer)
		if err != nil {
			return nil, err
		}

		dialectic.UserInteractions = append(dialectic.UserInteractions, *response.NewInteraction)
	}

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

			interaction := createNewQuestionInteraction(q)

			dialectic.UserInteractions = append(dialectic.UserInteractions, interaction)
		}
	}

	// Handle answer blob (from user)
	if input.AnswerBlob != "" {

		bs, err := dsvc.epistemology.Process(&models.DialecticEvent{
			PreviousInteractions: dialectic.UserInteractions,
		}, input.DryRun, input.SelfModelID)
		if err != nil {
			return nil, err
		}

		// Generate the first interaction
		response, err := dsvc.epistemology.Respond(bs, &models.DialecticEvent{
			PreviousInteractions: dialectic.UserInteractions,
		}, "")
		if err != nil {
			return nil, err
		}

		dialectic.UserInteractions = append(dialectic.UserInteractions, *response.NewInteraction)
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
		if qa := getQuestionAnswer(interactions[idx].Interaction); qa != nil {
			questions[i] = qa.Question.Question
		}
	}
	return questions
}

// Helper functions to access interaction data
func getQuestionAnswer(interaction *models.InteractionData) *models.QuestionAnswerInteraction {
	if interaction != nil {
		return interaction.QuestionAnswer
	}
	return nil
}

// Helper to get the question from an interaction
func getQuestion(interaction *models.DialecticalInteraction) string {
	if qa := getQuestionAnswer(interaction.Interaction); qa != nil {
		return qa.Question.Question
	}
	return ""
}

// Helper to get the answer from an interaction
func getAnswer(interaction *models.DialecticalInteraction) string {
	if qa := getQuestionAnswer(interaction.Interaction); qa != nil {
		return qa.Answer.UserAnswer
	}
	return ""
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

func (dsvc *DialecticService) PreprocessDialectic(answerBlob *string, dialectic *models.Dialectic) error {
	var pendingIndices []int
	for i, interaction := range dialectic.UserInteractions {
		if interaction.Status == models.StatusPendingAnswer {
			pendingIndices = append(pendingIndices, i)
		}
	}

	if len(pendingIndices) > 0 {
		// Try to match answers to questions using AI
		matches, err := dsvc.aih.MatchAnswersToQuestions(
			*answerBlob,
			getPendingQuestions(dialectic.UserInteractions, pendingIndices),
		)
		if err != nil {
			return fmt.Errorf("failed to match answers: %w", err)
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
			if qa := getQuestionAnswer(dialectic.UserInteractions[idx].Interaction); qa != nil && qa.Question.Question == "" {
				log.Printf("Skipping interaction with empty question at index %d", idx)
				continue
			}

			// Create the QuestionAnswer interaction
			qa := &models.QuestionAnswerInteraction{
				Question: getQuestionAnswer(dialectic.UserInteractions[idx].Interaction).Question,
				Answer: models.UserAnswer{
					UserAnswer:         answer,
					CreatedAtMillisUTC: time.Now().UnixMilli(),
				},
				ExtractedBeliefs: []*models.Belief{},
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
				return fmt.Errorf("failed to extract belief: %w", err)
			}

			extractedBelief := &models.Belief{
				ID:      uuid.New().String(),
				Content: []models.Content{{RawStr: extractedBeliefStr}},
				Type:    models.Statement,
			}

			log.Printf("Extracted belief: %+v", extractedBelief)

			if extractedBelief != nil {
				qa.ExtractedBeliefs = append(qa.ExtractedBeliefs, extractedBelief)
				log.Printf("Added extracted belief to QuestionAnswer: %+v", extractedBelief)
			}

			// Update the interaction
			dialectic.UserInteractions[idx].Type = models.InteractionTypeQuestionAnswer
			dialectic.UserInteractions[idx].Interaction = &models.InteractionData{
				QuestionAnswer: qa,
			}

			log.Printf("Created QuestionAnswer interaction with Question: %q, Answer: %q", qa.Question.Question, qa.Answer.UserAnswer)
			log.Printf("Extracted belief: %+v", extractedBelief)
			log.Printf("QuestionAnswer interaction after adding belief: %+v", qa)
		}
	}

	return nil
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
			Content:  getQuestion(interaction),
			SourceID: interaction.ID,
			ActionID: action.ID,
			Metadata: map[string]string{"interaction_type": "survey"},
		}
	case models.ActionTypeCollectEvidence:
		resource = &models.Resource{
			ID:       uuid.New().String(),
			Type:     models.ResourceTypeScientificPaper,
			Content:  getQuestion(interaction),
			SourceID: interaction.ID,
			ActionID: action.ID,
			Metadata: map[string]string{"interaction_type": "evidence"},
		}
	case models.ActionTypeActuateOutcome:
		resource = &models.Resource{
			ID:       uuid.New().String(),
			Type:     models.ResourceTypeMeasurementData,
			Content:  getQuestion(interaction),
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
func (dsvc *DialecticService) generatePredictedObservation(interaction *models.DialecticalInteraction) (*models.Prediction, error) {
	if interaction == nil {
		return nil, fmt.Errorf("interaction cannot be nil")
	}

	predictedAnswer, err := dsvc.aih.PredictAnswer(getQuestion(interaction))
	if err != nil {
		return nil, fmt.Errorf("failed to predict answer: %w", err)
	}

	return &models.Prediction{
		Observation: &models.Observation{
			DialecticInteractionID: interaction.ID,
			Type:                   models.Answer,
			StateDistribution:      map[string]float32{predictedAnswer: 1.0},
			Timestamp:              time.Now().UnixMilli(),
		},
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

// Update where we create new question interactions
func createNewQuestionInteraction(question string) models.DialecticalInteraction {
	return models.DialecticalInteraction{
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
}
