package svc

import (
	"context"
	"fmt"

	"epistemic-me-backend/db"
	"epistemic-me-backend/svc/models"

	"github.com/google/uuid"
)

type SelfModelService struct {
	kvStore *db.KeyValueStore
	dsvc    *DialecticService
	bsvc    *BeliefService
}

func NewSelfModelService(kvStore *db.KeyValueStore, dsvc *DialecticService, bsvc *BeliefService) *SelfModelService {
	return &SelfModelService{
		kvStore: kvStore,
		dsvc:    dsvc,
		bsvc:    bsvc,
	}
}

func (s *SelfModelService) CreateSelfModel(ctx context.Context, input *models.CreateSelfModelInput) (*models.CreateSelfModelOutput, error) {
	if input.ID == "" {
		return nil, fmt.Errorf("self model ID cannot be empty")
	}

	// Create an empty belief system for the self model
	emptyBeliefSystem := &models.BeliefSystem{
		Beliefs:             []*models.Belief{},
		ObservationContexts: []*models.ObservationContext{},
	}

	// Store the belief system separately
	err := s.kvStore.Store(input.ID, "BeliefSystem", *emptyBeliefSystem, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to store belief system: %v", err)
	}

	selfModel := &models.SelfModel{
		ID:           input.ID,
		Philosophies: input.Philosophies,
		BeliefSystem: emptyBeliefSystem,
		Dialectics:   []*models.Dialectic{},
	}

	// Store the self model
	err = s.kvStore.Store(input.ID, "SelfModel", *selfModel, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to store self model: %v", err)
	}

	return &models.CreateSelfModelOutput{SelfModel: selfModel}, nil
}

func (s *SelfModelService) GetSelfModel(ctx context.Context, input *models.GetSelfModelInput) (*models.GetSelfModelOutput, error) {
	storedSelfModel, err := s.kvStore.Retrieve(input.SelfModelID, "SelfModel")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve self model: %v", err)
	}

	selfModel, ok := storedSelfModel.(*models.SelfModel)
	if !ok {
		return nil, fmt.Errorf("invalid self model data")
	}

	// Retrieve the belief system separately
	storedBeliefSystem, err := s.kvStore.Retrieve(input.SelfModelID, "BeliefSystem")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve belief system: %v", err)
	}

	beliefSystem, ok := storedBeliefSystem.(*models.BeliefSystem)
	if !ok {
		return nil, fmt.Errorf("invalid belief system data")
	}

	selfModel.BeliefSystem = beliefSystem

	// Fetch associated dialectics
	dialectics, err := s.dsvc.ListDialectics(&models.ListDialecticsInput{UserID: selfModel.ID})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve dialectics: %v", err)
	}

	selfModel.Dialectics = make([]*models.Dialectic, len(dialectics.Dialectics))
	for i := range dialectics.Dialectics {
		selfModel.Dialectics[i] = &dialectics.Dialectics[i]
	}

	return &models.GetSelfModelOutput{SelfModel: selfModel}, nil
}

func (s *SelfModelService) AddPhilosophy(ctx context.Context, input *models.AddPhilosophyInput) (*models.AddPhilosophyOutput, error) {
	storedSelfModel, err := s.kvStore.Retrieve(input.SelfModelID, "SelfModel")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve self model: %v", err)
	}

	selfModel, ok := storedSelfModel.(*models.SelfModel)
	if !ok {
		return nil, fmt.Errorf("invalid self model data")
	}

	selfModel.Philosophies = append(selfModel.Philosophies, input.PhilosophyID)

	err = s.kvStore.Store(input.SelfModelID, "SelfModel", *selfModel, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to update self model: %v", err)
	}

	return &models.AddPhilosophyOutput{UpdatedSelfModel: selfModel}, nil
}

func (s *SelfModelService) CreatePhilosophy(ctx context.Context, input *models.CreatePhilosophyInput) (*models.CreatePhilosophyOutput, error) {
	philosophy := &models.Philosophy{
		ID:                  uuid.New().String(),
		Description:         input.Description,
		ExtrapolateContexts: input.ExtrapolateContexts,
	}

	err := s.kvStore.Store(philosophy.ID, "Philosophy", *philosophy, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to store philosophy: %v", err)
	}

	return &models.CreatePhilosophyOutput{Philosophy: philosophy}, nil
}

// Add this method to update the belief system of a self model
func (s *SelfModelService) UpdateSelfModelBeliefSystem(ctx context.Context, selfModelID string, beliefSystem *models.BeliefSystem) error {
	err := s.kvStore.Store(selfModelID, "BeliefSystem", *beliefSystem, 1)
	if err != nil {
		return fmt.Errorf("failed to update belief system: %v", err)
	}
	return nil
}
