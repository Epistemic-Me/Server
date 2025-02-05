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
	kvStore                 *db.KeyValueStore
	aih                     *ai.AIHelper
	perspectiveTakingEpiSvc *PerspectiveTakingEpistemology
	dialecticEpiSvc         *DialecticalEpistemology
}

// NewDialecticService initializes and returns a new DialecticService.
func NewDialecticService(kvStore *db.KeyValueStore, aih *ai.AIHelper,
	perspectiveTakingEpiSvc *PerspectiveTakingEpistemology,
	dialecticEpistemologySvc *DialecticalEpistemology) *DialecticService {
	return &DialecticService{
		kvStore:                 kvStore,
		aih:                     aih,
		perspectiveTakingEpiSvc: perspectiveTakingEpiSvc,
		dialecticEpiSvc:         dialecticEpistemologySvc,
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
		SelfModelID: input.SelfModelID,
		Agent: models.Agent{
			AgentType:     models.AgentTypeGPTLatest,
			DialecticType: input.DialecticType,
		},
		UserInteractions:  []models.DialecticalInteraction{},
		LearningObjective: input.LearningObjective,
	}

	// If there's a learning objective, generate initial question based on it
	if input.LearningObjective != nil {
		// Generate the first question based on learning objective
		question, err := dsvc.aih.GenerateQuestionForLearningObjective(input.LearningObjective, dialectic.UserInteractions)
		if err != nil {
			return nil, fmt.Errorf("failed to generate initial question: %w", err)
		}

		// Create initial interaction with the generated question
		interaction := createNewQuestionInteraction(question)
		dialectic.UserInteractions = append(dialectic.UserInteractions, interaction)
	} else {
		// Generate the first interaction using existing logic for non-learning objective dialectics
		response, err := dsvc.dialecticEpiSvc.Respond(&models.BeliefSystem{}, &models.DialecticEvent{
			PreviousInteractions: dialectic.UserInteractions,
		}, "")
		if err != nil {
			return nil, err
		}
		dialectic.UserInteractions = append(dialectic.UserInteractions, *response.NewInteraction)
	}

	// Add perspective selves if specified
	if len(input.PerspectiveModelIDs) > 0 {
		dialectic.PerspectiveModelIDs = input.PerspectiveModelIDs
		log.Printf("Adding perspective selves: %v", dialectic.PerspectiveModelIDs)
	}

	err := dsvc.storeDialecticValue(input.SelfModelID, dialectic)
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
		bs, err := dsvc.dialecticEpiSvc.Process(&models.DialecticEvent{
			PreviousInteractions: dialectic.UserInteractions,
		}, input.DryRun, input.SelfModelID)
		if err != nil {
			return nil, err
		}

		// Extract beliefs from the answer
		interactionEvent := ai.InteractionEvent{
			Question: getQuestion(&dialectic.UserInteractions[len(dialectic.UserInteractions)-1]),
			Answer:   input.Answer.UserAnswer,
		}

		extractedBeliefStrings, err := dsvc.aih.GetInteractionEventAsBelief(interactionEvent)
		if err != nil {
			return nil, fmt.Errorf("failed to extract beliefs: %w", err)
		}

		extractedBeliefs := make([]*models.Belief, 0, len(extractedBeliefStrings))
		for _, beliefStr := range extractedBeliefStrings {
			extractedBelief := &models.Belief{
				ID:      uuid.New().String(),
				Content: []models.Content{{RawStr: beliefStr}},
				Type:    models.Statement,
			}
			extractedBeliefs = append(extractedBeliefs, extractedBelief)
		}

		// Add the extracted beliefs to the BeliefSystem
		bs.Beliefs = append(bs.Beliefs, extractedBeliefs...)
		err = dsvc.kvStore.Store(input.SelfModelID, "BeliefSystem", *bs, len(bs.Beliefs))
		if err != nil {
			return nil, fmt.Errorf("failed to store updated belief system: %w", err)
		}

		// Update the last interaction with the answer and extracted beliefs
		lastIdx := len(dialectic.UserInteractions) - 1
		oldQA := getQuestionAnswer(dialectic.UserInteractions[lastIdx].Interaction)
		qa := &models.QuestionAnswerInteraction{
			Question: oldQA.Question,
			Answer: models.UserAnswer{
				UserAnswer:         input.Answer.UserAnswer,
				CreatedAtMillisUTC: time.Now().UnixMilli(),
			},
			ExtractedBeliefs:   extractedBeliefs,
			UpdatedAtMillisUTC: time.Now().UnixMilli(),
		}

		dialectic.UserInteractions[lastIdx].Status = models.StatusAnswered
		dialectic.UserInteractions[lastIdx].Type = models.InteractionTypeQuestionAnswer
		dialectic.UserInteractions[lastIdx].Interaction = &models.InteractionData{
			QuestionAnswer: qa,
		}
		dialectic.UserInteractions[lastIdx].UpdatedAtMillisUTC = time.Now().UnixMilli()

		// If we have a learning objective, check completion and generate next question
		if dialectic.LearningObjective != nil {
			// Get the current belief system
			bs, err := dsvc.dialecticEpiSvc.Process(&models.DialecticEvent{
				PreviousInteractions: dialectic.UserInteractions,
			}, false, dialectic.SelfModelID)
			if err != nil {
				return nil, fmt.Errorf("failed to get belief system: %w", err)
			}

			// Get the self model using existing KeyValueStore.Retrieve
			selfModelValue, err := dsvc.kvStore.Retrieve(dialectic.SelfModelID, "SelfModel")
			if err != nil {
				return nil, fmt.Errorf("failed to get self model: %w", err)
			}

			// Convert to SelfModel type
			selfModel, ok := selfModelValue.(*models.SelfModel)
			if !ok {
				return nil, fmt.Errorf("invalid self model type")
			}

			// Update the belief system with current beliefs
			selfModel.BeliefSystem = bs

			completionPercentage, err := dsvc.aih.CheckLearningObjectiveCompletion(dialectic.LearningObjective, selfModel)
			if err != nil {
				return nil, fmt.Errorf("failed to check learning objective completion: %w", err)
			}
			dialectic.LearningObjective.CompletionPercentage = completionPercentage

			// If not complete (less than 95%), generate next question based on learning objective
			if completionPercentage < 95 {
				nextQuestion, err := dsvc.aih.GenerateQuestionForLearningObjective(dialectic.LearningObjective, dialectic.UserInteractions)
				if err != nil {
					return nil, fmt.Errorf("failed to generate next question: %w", err)
				}

				interaction := createNewQuestionInteraction(nextQuestion)
				dialectic.UserInteractions = append(dialectic.UserInteractions, interaction)
			}
		} else {
			// Generate the next interaction using existing logic for non-learning objective dialectics
			response, err := dsvc.dialecticEpiSvc.Respond(bs, &models.DialecticEvent{
				PreviousInteractions: dialectic.UserInteractions,
			}, input.Answer.UserAnswer)
			if err != nil {
				return nil, err
			}

			dialectic.UserInteractions = append(dialectic.UserInteractions, *response.NewInteraction)
		}
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

		bs, err := dsvc.dialecticEpiSvc.Process(&models.DialecticEvent{
			PreviousInteractions: dialectic.UserInteractions,
		}, input.DryRun, input.SelfModelID)
		if err != nil {
			return nil, err
		}

		lastInteraction := dialectic.UserInteractions[len(dialectic.UserInteractions)-1]

		// For all perspectives we've attached to the dialectic, provide perspectives on the latest
		// dialectic interaction
		if dialectic.PerspectiveModelIDs != nil {
			for _, perspectiveModelID := range dialectic.PerspectiveModelIDs {
				perspective, err := dsvc.perspectiveTakingEpiSvc.Respond(bs, models.EpistemicRequest{
					SelfModelID: perspectiveModelID,
					Content: map[string]interface{}{
						"question": lastInteraction.Interaction.QuestionAnswer.Question,
						"answer":   lastInteraction.Interaction.QuestionAnswer.Answer,
					},
				})

				if err != nil {
					return nil, err
				}

				lastInteraction.Perspectives = append(lastInteraction.Perspectives, models.Perspective{
					Response:    *perspective,
					SelfModelID: perspectiveModelID,
				})
			}
		}

		// Generate the first interaction
		response, err := dsvc.dialecticEpiSvc.Respond(bs, &models.DialecticEvent{
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

			extractedBeliefStrings, err := dsvc.aih.GetInteractionEventAsBelief(interactionEvent)
			if err != nil {
				return fmt.Errorf("failed to extract beliefs: %w", err)
			}

			extractedBeliefs := make([]*models.Belief, 0, len(extractedBeliefStrings))
			for _, beliefStr := range extractedBeliefStrings {
				belief := &models.Belief{
					ID:      uuid.New().String(),
					Content: []models.Content{{RawStr: beliefStr}},
					Type:    models.Statement,
				}
				extractedBeliefs = append(extractedBeliefs, belief)
				log.Printf("Extracted belief: %+v", belief)
			}

			if len(extractedBeliefs) > 0 {
				qa.ExtractedBeliefs = append(qa.ExtractedBeliefs, extractedBeliefs...)
				log.Printf("Added %d extracted beliefs to QuestionAnswer", len(extractedBeliefs))
			}

			// Update the interaction
			dialectic.UserInteractions[idx].Type = models.InteractionTypeQuestionAnswer
			dialectic.UserInteractions[idx].Interaction = &models.InteractionData{
				QuestionAnswer: qa,
			}

			log.Printf("Created QuestionAnswer interaction with Question: %q, Answer: %q", qa.Question.Question, qa.Answer.UserAnswer)
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

// extractQuestionsFromBlob extracts questions from a blob, ignoring summary sections
func extractQuestionsFromBlob(blob string) string {
	// Split on section markers
	sections := strings.Split(blob, "---")

	// Get the last section which contains the new questions
	if len(sections) > 0 {
		return sections[len(sections)-1]
	}
	return blob
}

// PreprocessQuestionAnswers processes question and answer blobs into structured Q&A pairs
func (dsvc *DialecticService) PreprocessQuestionAnswers(input *models.PreprocessQuestionAnswerInput) (*models.PreprocessQuestionAnswerOutput, error) {
	var questions []string

	// Process question blobs
	for i, questionBlob := range input.QuestionBlobs {
		log.Printf("Processing Question Blob %d:\n%s\n", i+1, questionBlob)

		// Extract only the question section
		questionSection := extractQuestionsFromBlob(questionBlob)
		log.Printf("Question Section %d:\n%s\n", i+1, questionSection)

		extractedQuestions, err := dsvc.aih.ExtractQuestionsFromText(questionSection)
		if err != nil {
			return nil, fmt.Errorf("failed to extract questions: %w", err)
		}
		log.Printf("Extracted Questions %d: %v\n", i+1, extractedQuestions)
		questions = append(questions, extractedQuestions...)
	}

	// Initialize QA pairs with empty answers
	qaPairs := make([]*models.QuestionAnswerPair, len(questions))
	for i, question := range questions {
		qaPairs[i] = &models.QuestionAnswerPair{
			Question: question,
			Answer:   "No answer provided",
		}
	}

	// Process answer blobs
	allAnswers := strings.Join(input.AnswerBlobs, "\n\n")
	log.Printf("Combined Answer Blob:\n%s\n", allAnswers)
	matches, err := dsvc.aih.MatchAnswersToQuestions(allAnswers, questions)
	if err != nil {
		return nil, fmt.Errorf("failed to match answers: %w", err)
	}
	log.Printf("Matched Answers: %v\n", matches)

	// Update answers for each question
	for i, match := range matches {
		if i < len(qaPairs) && match != "" {
			qaPairs[i].Answer = match
		}
	}

	return &models.PreprocessQuestionAnswerOutput{
		QAPairs: qaPairs,
	}, nil
}
