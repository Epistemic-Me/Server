package svc

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	"epistemic-me-core/db"
	"epistemic-me-core/svc/models"
)

// InteractionEvent represents a question-answer interaction
type InteractionEvent struct {
	Question string
	Answer   string
}

// OptimizedDialecticService provides performance-optimized implementations
// of DialecticService functions, particularly focusing on reducing AI calls
type OptimizedDialecticService struct {
	kvStore                    *db.KeyValueStore
	aiHelper                   AIHelperInterface
	dialecticEpiSvc            *DialecticalEpistemology
	enablePredictiveProcessing bool
}

// AIHelperInterface defines the methods required from an AI helper
type AIHelperInterface interface {
	GetInteractionEventAsBelief(event InteractionEvent) ([]string, error)
	GenerateQuestion(beliefSystem string, previousEvents []InteractionEvent) (string, error)
	ExtractQuestionsFromText(text string) ([]string, error)
}

// NewOptimizedDialecticService creates a new instance of OptimizedDialecticService
func NewOptimizedDialecticService(kvStore *db.KeyValueStore, aiHelper AIHelperInterface, dialecticEpiSvc *DialecticalEpistemology) *OptimizedDialecticService {
	return &OptimizedDialecticService{
		kvStore:                    kvStore,
		aiHelper:                   aiHelper,
		dialecticEpiSvc:            dialecticEpiSvc,
		enablePredictiveProcessing: true,
	}
}

// NewOptimizedDialecticServiceForTesting creates a service instance for testing purposes
// that can work with any types that implement the required interfaces
func NewOptimizedDialecticServiceForTesting(kvStore interface{}, aiHelper AIHelperInterface, dialecticEpiSvc *DialecticalEpistemology) *OptimizedDialecticService {
	// Use the existing service constructor but with type assertion to handle mock objects
	return &OptimizedDialecticService{
		kvStore:                    kvStore.(*db.KeyValueStore),
		aiHelper:                   aiHelper,
		dialecticEpiSvc:            dialecticEpiSvc,
		enablePredictiveProcessing: true,
	}
}

// OptimizedUpdateDialectic provides a more efficient implementation of UpdateDialectic
// with reduced AI calls and better structure for PredictiveProcessingContext
func (svc *OptimizedDialecticService) OptimizedUpdateDialectic(input *models.UpdateDialecticInput) (*models.UpdateDialecticOutput, error) {
	startTime := time.Now()

	// Retrieve the dialectic
	dialectic, err := svc.retrieveDialecticValue(input.SelfModelID, input.ID)
	if err != nil {
		return nil, err
	}
	log.Printf("Retrieved dialectic with %d interactions in %v",
		len(dialectic.UserInteractions), time.Since(startTime))

	if input.Answer.UserAnswer != "" {
		// OPTIMIZATION: Process the belief system with dialecticEpiSvc but enhance PredictiveProcessingContext
		bs, err := svc.dialecticEpiSvc.Process(&models.DialecticEvent{
			PreviousInteractions: dialectic.UserInteractions,
		}, input.DryRun, input.SelfModelID)
		if err != nil {
			return nil, err
		}

		// Additional PredictiveProcessingContext enhancements beyond what Process provides
		if svc.enablePredictiveProcessing {
			// Extract the current interaction for enhanced context tracking
			interactionEvent := InteractionEvent{
				Question: svc.getQuestion(&dialectic.UserInteractions[len(dialectic.UserInteractions)-1]),
				Answer:   input.Answer.UserAnswer,
			}

			// Extract beliefs from the latest answer (to ensure fresh context)
			extractedBeliefStrings, err := svc.aiHelper.GetInteractionEventAsBelief(interactionEvent)
			if err != nil {
				return nil, fmt.Errorf("failed to extract beliefs: %w", err)
			}

			// Convert to belief objects
			extractedBeliefs := make([]*models.Belief, 0, len(extractedBeliefStrings))
			for _, beliefStr := range extractedBeliefStrings {
				// Check if this belief already exists in updated belief system
				beliefExists := false
				for _, existingBelief := range bs.Beliefs {
					if strings.Contains(existingBelief.GetContentAsString(), beliefStr) {
						beliefExists = true
						break
					}
				}

				// Only add if it's a new insight
				if !beliefExists {
					extractedBelief := &models.Belief{
						ID:      uuid.New().String(),
						Content: []models.Content{{RawStr: beliefStr}},
						Type:    models.Statement,
					}
					extractedBeliefs = append(extractedBeliefs, extractedBelief)
					bs.Beliefs = append(bs.Beliefs, extractedBelief)
				}
			}

			// Enhanced PredictiveProcessingContext updates with conversation awareness
			svc.updatePredictiveProcessingContext(bs, dialectic, extractedBeliefs, interactionEvent)
		}

		// Store the updated belief system
		err = svc.kvStore.Store(input.SelfModelID, "BeliefSystem", *bs, len(bs.Beliefs))
		if err != nil {
			return nil, fmt.Errorf("failed to store updated belief system: %w", err)
		}

		// Update the last interaction with the answer and extracted beliefs
		lastIdx := len(dialectic.UserInteractions) - 1
		oldQA := svc.getQuestionAnswer(dialectic.UserInteractions[lastIdx].Interaction)

		// Get extracted beliefs from the belief system, which includes both updated and new beliefs
		var extractedBeliefs []*models.Belief
		if len(bs.Beliefs) > 0 {
			// Use the most recent beliefs (up to 5) for better relevance
			startIdx := 0
			if len(bs.Beliefs) > 5 {
				startIdx = len(bs.Beliefs) - 5
			}

			// Create new belief objects instead of referencing existing ones
			for i := startIdx; i < len(bs.Beliefs); i++ {
				belief := models.Belief{
					ID:      bs.Beliefs[i].ID,
					Content: bs.Beliefs[i].Content,
					Type:    bs.Beliefs[i].Type,
					Version: bs.Beliefs[i].Version,
				}
				extractedBeliefs = append(extractedBeliefs, &belief)
			}
		}

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

		// Generate next question using dialecticEpiSvc (leveraging the standard implementation)
		response, err := svc.dialecticEpiSvc.Respond(bs, &models.DialecticEvent{
			PreviousInteractions: dialectic.UserInteractions,
		}, "")
		if err != nil {
			return nil, fmt.Errorf("failed to generate next question: %w", err)
		}

		// Add the new interaction from the response
		if response.NewInteraction != nil {
			dialectic.UserInteractions = append(dialectic.UserInteractions, *response.NewInteraction)
		}
	}

	// Handle question blob (efficiently process multiple questions at once)
	if input.QuestionBlob != "" {
		err = svc.processQuestionBlob(dialectic, input)
		if err != nil {
			return nil, err
		}
	}

	// Handle answer blob
	if input.AnswerBlob != "" {
		// Retrieve the belief system (necessary for processing with dialecticEpiSvc)
		bs, err := svc.retrieveBeliefSystem(input.SelfModelID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve belief system: %w", err)
		}

		// Generate the next question using dialecticEpiSvc
		response, err := svc.dialecticEpiSvc.Respond(bs, &models.DialecticEvent{
			PreviousInteractions: dialectic.UserInteractions,
		}, "")
		if err != nil {
			return nil, fmt.Errorf("failed to generate next question: %w", err)
		}

		// Add the new interaction from the response
		if response.NewInteraction != nil {
			dialectic.UserInteractions = append(dialectic.UserInteractions, *response.NewInteraction)
		}
	}

	// Save changes if not a dry run
	if !input.DryRun {
		err = svc.storeDialecticValue(input.SelfModelID, dialectic)
		if err != nil {
			return nil, err
		}
	}
	log.Printf("Completed UpdateDialectic in %v", time.Since(startTime))

	return &models.UpdateDialecticOutput{
		Dialectic: *dialectic,
	}, nil
}

// processQuestionBlob efficiently processes a blob of text containing multiple questions
func (svc *OptimizedDialecticService) processQuestionBlob(
	dialectic *models.Dialectic,
	input *models.UpdateDialecticInput,
) error {
	// Extract questions in a single AI call
	questions, err := svc.aiHelper.ExtractQuestionsFromText(input.QuestionBlob)
	if err != nil {
		return fmt.Errorf("failed to extract questions: %w", err)
	}

	// Create new interactions for each question
	for _, q := range questions {
		if q == "" {
			continue
		}
		interaction := svc.createNewQuestionInteraction(q)
		dialectic.UserInteractions = append(dialectic.UserInteractions, interaction)
	}

	return nil
}

// updatePredictiveProcessingContext updates the PPC with the new observation and belief contexts
func (svc *OptimizedDialecticService) updatePredictiveProcessingContext(
	bs *models.BeliefSystem,
	dialectic *models.Dialectic,
	newBeliefs []*models.Belief,
	interactionEvent InteractionEvent,
) {
	// Initialize PredictiveProcessingContext if not already present
	if len(bs.EpistemicContexts) == 0 {
		bs.EpistemicContexts = []*models.EpistemicContext{
			{
				PredictiveProcessingContext: &models.PredictiveProcessingContext{
					ObservationContexts: []*models.ObservationContext{},
					BeliefContexts:      []*models.BeliefContext{},
				},
			},
		}
	}

	ppc := bs.EpistemicContexts[0].PredictiveProcessingContext
	if ppc == nil {
		ppc = &models.PredictiveProcessingContext{
			ObservationContexts: []*models.ObservationContext{},
			BeliefContexts:      []*models.BeliefContext{},
		}
		bs.EpistemicContexts[0].PredictiveProcessingContext = ppc
	}

	// Create an ObservationContext for this interaction
	ocID := uuid.New().String()
	oc := &models.ObservationContext{
		ID:             ocID,
		Name:           fmt.Sprintf("Response to '%s'", interactionEvent.Question),
		ParentID:       "",
		PossibleStates: []string{"Positive", "Negative", "Neutral"},
	}
	ppc.ObservationContexts = append(ppc.ObservationContexts, oc)

	// Create BeliefContext entries for each new belief
	for _, belief := range newBeliefs {
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

// retrieveBeliefSystem gets the belief system from the key-value store
func (svc *OptimizedDialecticService) retrieveBeliefSystem(selfModelID string) (*models.BeliefSystem, error) {
	bsValue, err := svc.kvStore.Retrieve(selfModelID, "BeliefSystem")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve belief system: %w", err)
	}

	bs, ok := bsValue.(*models.BeliefSystem)
	if !ok {
		return nil, fmt.Errorf("invalid belief system type")
	}

	return bs, nil
}

// Helper functions reused from the original DialecticService

func (svc *OptimizedDialecticService) storeDialecticValue(selfModelID string, dialectic *models.Dialectic) error {
	return svc.kvStore.Store(selfModelID, fmt.Sprintf("Dialectic:%s", dialectic.ID), dialectic, len(dialectic.UserInteractions))
}

func (svc *OptimizedDialecticService) retrieveDialecticValue(selfModelID, dialecticID string) (*models.Dialectic, error) {
	value, err := svc.kvStore.Retrieve(selfModelID, fmt.Sprintf("Dialectic:%s", dialecticID))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve dialectic: %w", err)
	}

	dialectic, ok := value.(*models.Dialectic)
	if !ok {
		return nil, fmt.Errorf("invalid dialectic type")
	}

	return dialectic, nil
}

// getQuestionAnswer extracts the QuestionAnswer from an interaction
func (svc *OptimizedDialecticService) getQuestionAnswer(data *models.InteractionData) *models.QuestionAnswerInteraction {
	if data != nil && data.QuestionAnswer != nil {
		return data.QuestionAnswer
	}
	return nil
}

// getQuestion extracts the question string from a dialectical interaction
func (svc *OptimizedDialecticService) getQuestion(interaction *models.DialecticalInteraction) string {
	if interaction.Interaction != nil && interaction.Interaction.QuestionAnswer != nil {
		return interaction.Interaction.QuestionAnswer.Question.Question
	}
	return ""
}

// createNewQuestionInteraction creates a new interaction with a question
func (svc *OptimizedDialecticService) createNewQuestionInteraction(question string) models.DialecticalInteraction {
	now := time.Now().UnixMilli()
	return models.DialecticalInteraction{
		ID:                 uuid.New().String(),
		Status:             models.StatusPendingAnswer,
		Type:               models.InteractionTypeQuestionAnswer,
		UpdatedAtMillisUTC: now,
		Interaction: &models.InteractionData{
			QuestionAnswer: &models.QuestionAnswerInteraction{
				Question: models.Question{
					Question:           question,
					CreatedAtMillisUTC: now,
				},
				UpdatedAtMillisUTC: now,
			},
		},
	}
}
