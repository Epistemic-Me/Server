package svc

import (
	ai "epistemic-me-backend/ai"
	db "epistemic-me-backend/db"
	"epistemic-me-backend/svc/models"
	"fmt"
	"reflect"

	"github.com/google/uuid"
)

type DialecticService struct {
	kv   *db.KeyValueStore
	bsvc *BeliefService
	aih  *ai.AIHelper
}

// NewDialecticService initializes and returns a new DialecticService.
func NewDialecticService(kv *db.KeyValueStore, bsvc *BeliefService, aih *ai.AIHelper) *DialecticService {
	return &DialecticService{
		kv:   kv,
		bsvc: bsvc,
		aih:  aih,
	}
}

func (dsvc *DialecticService) CreateDialectic(input *models.CreateDialecticInput) (*models.CreateDialecticOutput, error) {

	// dialectics begin with a system generated question
	var userInteractions []models.DialecticalInteraction
	newInteraction, err := dsvc.generatePendingDialecticalInteraction(input.UserID, userInteractions)
	if err != nil {
		return nil, err
	}

	// append the first pending question to the dialectic
	userInteractions = append(userInteractions, *newInteraction)

	newDialecticId := "di_" + uuid.New().String()

	// instantiate the dialectic
	dialectic := &models.Dialectic{
		ID:     newDialecticId,
		UserID: input.UserID,
		Agent: models.Agent{
			AgentType:     models.AgentTypeGPTLatest,
			DialecticType: input.DialecticType,
		},
		UserInteractions: userInteractions,
	}

	err = dsvc.kv.Store(input.UserID, dialectic.ID, &dialectic)
	if err != nil {
		return nil, err
	}

	return &models.CreateDialecticOutput{
		DialecticID: dialectic.ID,
		Dialectic:   *dialectic,
	}, nil
}

func (dsvc *DialecticService) ListDialectics(input *models.ListDialecticsInput) (*models.ListDialecticsOutput, error) {
	// Retrieve all dialectics for the user
	dialectics, err := dsvc.kv.ListByType(input.UserID, reflect.TypeOf(models.Dialectic{}))
	if err != nil {
		return nil, err
	}

	// Convert the retrieved dialectics to the required type
	var dialecticModels []models.Dialectic
	for _, dialectic := range dialectics {
		storedDialectic := dialectic.(*models.Dialectic)
		dialecticModels = append(dialecticModels, *storedDialectic)
	}

	return &models.ListDialecticsOutput{
		Dialectics: dialecticModels,
	}, nil
}

func (dsvc *DialecticService) UpdateDialectic(input *models.UpdateDialecticInput) (*models.UpdateDialecticOutput, error) {
	kvResponse, err := dsvc.kv.Retrieve(input.UserID, input.DialecticID)
	if err != nil {
		return nil, err
	}

	// Type assertion to convert kvResponse to *Dialectic
	dialectic, ok := kvResponse.(*models.Dialectic)
	if !ok {
		return nil, fmt.Errorf("failed to assert kvResponse to *Dialectic")
	}

	interaction, err := getPendingInteraction(*dialectic)
	if err != nil {
		return nil, err
	}

	// update pending interaction with answer and mark it as answered
	interaction.UserAnswer = input.Answer
	interaction.Status = models.StatusAnswered

	// extract beliefs from the completed interaction
	interactionEvent, err := getDialecticalInteractionAsEvent(*interaction)
	if err != nil {
		return nil, err
	}

	interpretedBeliefStr, err := dsvc.aih.GetInteractionEventAsBelief(*interactionEvent)
	if err != nil {
		return nil, err
	}

	// store the interpeted belief as a user belief so it will be included in the belief system
	_, err = dsvc.bsvc.CreateBelief(&models.CreateBeliefInput{
		UserID:        input.UserID,
		BeliefContent: interpretedBeliefStr,
	})

	if err != nil {
		return nil, err
	}

	// generate a new interaction given updated state of dialectic and user belief system
	newInteraction, err := dsvc.generatePendingDialecticalInteraction(input.UserID, dialectic.UserInteractions)
	if err != nil {
		return nil, err
	}

	dialectic.UserInteractions = append(dialectic.UserInteractions, *newInteraction)

	err = dsvc.kv.Store(input.UserID, dialectic.ID, &dialectic)
	if err != nil {
		return nil, err
	}

	return &models.UpdateDialecticOutput{
		Dialectic: *dialectic,
	}, nil

}

func (dsvc *DialecticService) generatePendingDialecticalInteraction(userID string, previousInteractions []models.DialecticalInteraction) (*models.DialecticalInteraction, error) {
	// Get the Latest Belief System in orer to update the dialectic
	listBeliefsOutput, err := dsvc.bsvc.ListBeliefs(&models.ListBeliefsInput{
		UserID: userID,
	})

	if err != nil {
		return nil, err
	}

	user_belief_system := listBeliefsOutput.BeliefSystem

	var events []ai.InteractionEvent
	for _, interaction := range previousInteractions {
		interactionEvent, err := getDialecticalInteractionAsEvent(interaction)
		if err != nil {
			return nil, err
		}
		events = append(events, *interactionEvent)
	}

	question, err := dsvc.aih.GenerateQuestion(user_belief_system.RawStr, events)

	if err != nil {
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
		return nil, fmt.Errorf("no interactions found in the dialectic")
	}
	latestInteraction := dialectic.UserInteractions[len(dialectic.UserInteractions)-1]
	// Check if the latest interaction is pending
	if latestInteraction.Status != models.StatusPendingAnswer {
		return nil, fmt.Errorf("latest interaction is not pending")
	}
	return &latestInteraction, nil
}

func getDialecticalInteractionAsEvent(interaction models.DialecticalInteraction) (*ai.InteractionEvent, error) {
	if interaction.Status != models.StatusAnswered {
		return nil, fmt.Errorf("attempting to create interaction event from unanswered question")
	}
	return &ai.InteractionEvent{
		Question: interaction.Question.Question,
		Answer:   interaction.UserAnswer.UserAnswer,
	}, nil
}
