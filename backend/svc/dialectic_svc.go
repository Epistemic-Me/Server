package svc

import (
	ai "epistemic-me-backend/ai"
	db "epistemic-me-backend/db"
	"epistemic-me-backend/svc/models"
	"fmt"
	"log"
	"reflect"
	"strings"

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
	newInteraction, err := dsvc.generatePendingDialecticalInteraction(input.SelfModelID, dialectic.UserInteractions, models.BeliefSystem{})
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

	interaction.UserAnswer = input.Answer
	interaction.Status = models.StatusAnswered

	interactionEvent, err := getDialecticalInteractionAsEvent(*interaction)
	if err != nil {
		log.Printf("Error in getDialecticalInteractionAsEvent: %v", err)
		return nil, err
	}

	strategy := determineDialecticStrategy(dialectic.Agent.DialecticType)

	dryRun := input.DryRun

	// given the interaction event update the users existing belief system
	// by updating old beleifs or creating new ones
	beliefSystem, err := dsvc.updateBeliefSystemForInteraction(*interactionEvent, input.SelfModelID, dryRun)
	if err != nil {
		return nil, err
	}

	// generate a new interaction given updated state of dialectic and user belief system
	newInteraction, err := dsvc.generatePendingDialecticalInteraction(input.SelfModelID, dialectic.UserInteractions, *beliefSystem)
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

func (dsvc *DialecticService) generatePendingDialecticalInteraction(selfModelID string, previousInteractions []models.DialecticalInteraction, userBeliefSystem models.BeliefSystem) (*models.DialecticalInteraction, error) {

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

	// Update this line:
	question, err := dsvc.aih.GenerateQuestion(strings.Join(beliefStrings, " "), events)

	if err != nil {
		log.Printf("Error in GenerateQuestion: %v", err)
		return nil, err
	}

	return &models.DialecticalInteraction{
		Question: models.Question{
			Question: question,
		},
		Status: models.StatusPendingAnswer,
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
