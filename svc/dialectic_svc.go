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
	log.Printf("UpdateDialectic called with input: %+v", input)

	dialectic, err := dsvc.retrieveDialecticValue(input.SelfModelID, input.ID)
	if err != nil {
		log.Printf("Error retrieving dialectic: %v", err)
		return nil, err
	}

	interaction, err := getPendingInteraction(*dialectic)
	if err != nil {
		log.Printf("Error getting pending interaction: %v", err)
		return nil, err
	}

	// Update the interaction's answer and status
	interaction.UserAnswer = input.Answer
	interaction.Status = models.StatusAnswered

	// Update the interaction in the dialectic's UserInteractions slice
	for i := range dialectic.UserInteractions {
		if dialectic.UserInteractions[i].Status == models.StatusPendingAnswer {
			dialectic.UserInteractions[i] = *interaction
			break
		}
	}

	interactionEvent, err := getDialecticalInteractionAsEvent(*interaction)
	if err != nil {
		log.Printf("Error in getDialecticalInteractionAsEvent: %v", err)
		return nil, err
	}

	strategy := determineDialecticStrategy(dialectic.Agent.DialecticType)

	dryRun := input.DryRun

	// given the interaction event update the users existing belief system
	// by updating old beliefs or creating new ones
	beliefSystem, err := dsvc.updateBeliefSystemForInteraction(*interactionEvent, input.SelfModelID, dryRun)
	if err != nil {
		return nil, err
	}

	// generate a new interaction given updated state of dialectic and user belief system
	newInteraction, err := dsvc.generatePendingDialecticalInteraction(
		input.SelfModelID,
		dialectic.UserInteractions,
		*beliefSystem,
		input.CustomQuestion,
	)
	if err != nil {
		return nil, err
	}

	dialectic.UserInteractions = append(dialectic.UserInteractions, *newInteraction)

	analysis, err := dsvc.aih.GenerateAnalysisForStrategy(strategy, beliefSystem, dialectic.UserInteractions, *interactionEvent)
	if err != nil {
		log.Printf("Error generating belief analysis: %v", err)
		return nil, err
	}

	dialectic.Analysis = analysis

	if !dryRun {
		err = dsvc.storeDialecticValue(input.SelfModelID, dialectic)
		if err != nil {
			return nil, fmt.Errorf("failed to store updated dialectic: %w", err)
		}
	}

	log.Printf("Final dialectic before returning: %+v", dialectic)
	return &models.UpdateDialecticOutput{
		Dialectic: *dialectic,
	}, nil
}

func (dsvc *DialecticService) updateBeliefSystemForInteraction(interactionEvent ai.InteractionEvent, selfModelID string, dryRun bool) (*models.BeliefSystem, error) {
	listBeliefsOutput, err := dsvc.bsvc.ListBeliefs(&models.ListBeliefsInput{
		SelfModelID: selfModelID,
	})

	if err != nil {
		return nil, err
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
				BeliefType:           models.Clarification,
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

	return &models.DialecticalInteraction{
		Question: models.Question{
			Question:           question,
			CreatedAtMillisUTC: time.Now().UnixMilli(),
		},
		Status: models.StatusPendingAnswer,
		Type:   models.InteractionTypeQuestionAnswer, // Set default type
	}, nil
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
func (dsvc *DialecticService) ExecuteAction(action *models.Action, interaction *models.DialecticalInteraction) (*models.Observation, error) {
	// Get the observation context from the interaction
	beliefContext, err := dsvc.getBeliefContextFromInteraction(interaction)
	if err != nil {
		return nil, fmt.Errorf("failed to get belief context: %w", err)
	}

	// Generate source based on action type
	var source *models.Source
	switch action.Type {
	case models.ActionTypeAnswerQuestion:
		source, err = dsvc.generateAnswerSource(action, interaction)
	case models.ActionTypeCollectEvidence:
		source, err = dsvc.generateEvidenceSource(action, interaction)
	case models.ActionTypeActuateOutcome:
		source, err = dsvc.generateOutcomeSource(action, interaction)
	default:
		return nil, fmt.Errorf("invalid action type: %v", action.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to generate source: %w", err)
	}

	// Create observation
	observation := &models.Observation{
		DialecticInteractionID: action.DialecticInteractionID,
		Type:                   models.ObservationType(1),
		Source:                 source,
		StateDistribution: map[string]float32{
			dsvc.interpretSourceAsState(source, beliefContext): 1.0,
		},
		Timestamp: time.Now().UnixMilli(),
	}

	return observation, nil
}

// Add missing methods
func (dsvc *DialecticService) getBeliefContextFromInteraction(interaction *models.DialecticalInteraction) (*models.BeliefContext, error) {
	// Implementation
	return nil, fmt.Errorf("not implemented")
}

func (dsvc *DialecticService) generateAnswerSource(action *models.Action, interaction *models.DialecticalInteraction) (*models.Source, error) {
	// Implementation
	return nil, fmt.Errorf("not implemented")
}

func (dsvc *DialecticService) generateEvidenceSource(action *models.Action, interaction *models.DialecticalInteraction) (*models.Source, error) {
	// Implementation
	return nil, fmt.Errorf("not implemented")
}

func (dsvc *DialecticService) generateOutcomeSource(action *models.Action, interaction *models.DialecticalInteraction) (*models.Source, error) {
	// Implementation
	return nil, fmt.Errorf("not implemented")
}

func (dsvc *DialecticService) interpretSourceAsState(source *models.Source, context *models.BeliefContext) string {
	// Implementation
	return ""
}

// generatePredictedObservation creates a predicted observation for a dialectical interaction
func (dsvc *DialecticService) generatePredictedObservation(interaction *models.DialecticalInteraction) (*models.Observation, error) {
	if interaction == nil {
		return nil, fmt.Errorf("interaction cannot be nil")
	}

	if interaction.Question.Question == "" {
		return nil, fmt.Errorf("interaction question cannot be empty")
	}

	// Use AI helper to predict likely answer based on belief system
	predictedAnswer, err := dsvc.aih.PredictAnswer(interaction.Question.Question)
	if err != nil {
		return nil, fmt.Errorf("failed to predict answer: %w", err)
	}

	return &models.Observation{
		DialecticInteractionID: interaction.ID,
		Type:                   models.Answer,
		Source:                 nil,
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
		Type:                   models.ActionTypeAnswerQuestion,
		DialecticInteractionID: interaction.ID,
		ResourceID:             selfModelID,
		Timestamp:              time.Now().UnixMilli(),
	}

	observation, err := dsvc.ExecuteAction(action, interaction)
	if err != nil {
		return nil, fmt.Errorf("failed to execute action: %w", err)
	}

	return observation, nil
}
