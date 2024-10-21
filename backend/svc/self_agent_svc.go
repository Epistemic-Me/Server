package svc

import (
	"context"
	"fmt"

	"epistemic-me-backend/db"
	"epistemic-me-backend/svc/models"

	"github.com/google/uuid"
)

type SelfAgentService struct {
	kvStore *db.KeyValueStore
	dsvc    *DialecticService
	bsvc    *BeliefService // Add this to access belief-related functionality
}

func NewSelfAgentService(kvStore *db.KeyValueStore, dsvc *DialecticService, bsvc *BeliefService) *SelfAgentService {
	return &SelfAgentService{
		kvStore: kvStore,
		dsvc:    dsvc,
		bsvc:    bsvc,
	}
}

func (s *SelfAgentService) CreateSelfAgent(ctx context.Context, input *models.CreateSelfAgentInput) (*models.CreateSelfAgentOutput, error) {
	if input.ID == "" {
		return nil, fmt.Errorf("self agent ID cannot be empty")
	}

	// Create an empty belief system for the self agent
	emptyBeliefSystem := &models.BeliefSystem{
		Beliefs:             []*models.Belief{},
		ObservationContexts: []*models.ObservationContext{},
	}

	// Store the belief system separately
	err := s.kvStore.Store(input.ID, "BeliefSystem", *emptyBeliefSystem, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to store belief system: %v", err)
	}

	selfAgent := &models.SelfAgent{
		ID:           input.ID,
		Philosophies: input.Philosophies,
		BeliefSystem: emptyBeliefSystem,
		Dialectics:   []*models.Dialectic{},
	}

	// Store the self agent
	err = s.kvStore.Store(input.ID, "SelfAgent", *selfAgent, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to store self agent: %v", err)
	}

	return &models.CreateSelfAgentOutput{SelfAgent: selfAgent}, nil
}

func (s *SelfAgentService) GetSelfAgent(ctx context.Context, input *models.GetSelfAgentInput) (*models.GetSelfAgentOutput, error) {
	storedSelfAgent, err := s.kvStore.Retrieve(input.SelfAgentID, "SelfAgent")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve self agent: %v", err)
	}

	selfAgent, ok := storedSelfAgent.(*models.SelfAgent)
	if !ok {
		return nil, fmt.Errorf("invalid self agent data")
	}

	// Retrieve the belief system separately
	storedBeliefSystem, err := s.kvStore.Retrieve(input.SelfAgentID, "BeliefSystem")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve belief system: %v", err)
	}

	beliefSystem, ok := storedBeliefSystem.(*models.BeliefSystem)
	if !ok {
		return nil, fmt.Errorf("invalid belief system data")
	}

	selfAgent.BeliefSystem = beliefSystem

	// Fetch associated dialectics
	dialectics, err := s.dsvc.ListDialectics(&models.ListDialecticsInput{UserID: selfAgent.ID})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve dialectics: %v", err)
	}

	selfAgent.Dialectics = make([]*models.Dialectic, len(dialectics.Dialectics))
	for i := range dialectics.Dialectics {
		selfAgent.Dialectics[i] = &dialectics.Dialectics[i]
	}

	return &models.GetSelfAgentOutput{SelfAgent: selfAgent}, nil
}

func (s *SelfAgentService) AddPhilosophy(ctx context.Context, input *models.AddPhilosophyInput) (*models.AddPhilosophyOutput, error) {
	storedSelfAgent, err := s.kvStore.Retrieve(input.SelfAgentID, "SelfAgent")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve self agent: %v", err)
	}

	selfAgent, ok := storedSelfAgent.(*models.SelfAgent)
	if !ok {
		return nil, fmt.Errorf("invalid self agent data")
	}

	selfAgent.Philosophies = append(selfAgent.Philosophies, input.PhilosophyID)

	err = s.kvStore.Store(input.SelfAgentID, "SelfAgent", *selfAgent, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to update self agent: %v", err)
	}

	return &models.AddPhilosophyOutput{UpdatedSelfAgent: selfAgent}, nil
}

func (s *SelfAgentService) CreatePhilosophy(ctx context.Context, input *models.CreatePhilosophyInput) (*models.CreatePhilosophyOutput, error) {
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

// Add this method to update the belief system of a self agent
func (s *SelfAgentService) UpdateSelfAgentBeliefSystem(ctx context.Context, selfAgentID string, beliefSystem *models.BeliefSystem) error {
	err := s.kvStore.Store(selfAgentID, "BeliefSystem", *beliefSystem, 1)
	if err != nil {
		return fmt.Errorf("failed to update belief system: %v", err)
	}
	return nil
}
