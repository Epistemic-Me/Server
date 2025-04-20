package svc

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	"epistemic-me-core/db"
	"epistemic-me-core/svc/models"

	"github.com/google/uuid"
)

type SelfModelService struct {
	kvStore *db.KeyValueStore
	dsvc    *DialecticService
	bsvc    *BeliefService
	cache   map[string][]*models.ObservationContext
	cacheMu sync.RWMutex
}

func NewSelfModelService(kvStore *db.KeyValueStore, dsvc *DialecticService, bsvc *BeliefService) *SelfModelService {
	return &SelfModelService{
		kvStore: kvStore,
		dsvc:    dsvc,
		bsvc:    bsvc,
		cache:   make(map[string][]*models.ObservationContext),
	}
}

func (s *SelfModelService) CreateSelfModel(ctx context.Context, input *models.CreateSelfModelInput) (*models.CreateSelfModelOutput, error) {
	if input.ID == "" {
		return nil, fmt.Errorf("self model ID cannot be empty")
	}

	// Create an empty belief system for the self model
	emptyBeliefSystem := &models.BeliefSystem{
		Beliefs: []*models.Belief{},
		EpistemicContexts: []*models.EpistemicContext{
			{
				PredictiveProcessingContext: &models.PredictiveProcessingContext{
					ObservationContexts: []*models.ObservationContext{},
					BeliefContexts:      []*models.BeliefContext{},
				},
			},
		},
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
	dialectics, err := s.dsvc.ListDialectics(&models.ListDialecticsInput{SelfModelID: selfModel.ID})
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

func extrapolateObservationContexts(description string) []*models.ObservationContext {
	re := regexp.MustCompile(`\[\[(C|S): ([^\]]+)\]\]`)
	matches := re.FindAllStringSubmatch(description, -1)
	unique := make(map[string]struct{})
	var contexts []*models.ObservationContext
	for _, m := range matches {
		if len(m) > 2 {
			name := m[2]
			if _, exists := unique[name]; !exists {
				unique[name] = struct{}{}
				contexts = append(contexts, &models.ObservationContext{
					ID:   uuid.New().String(),
					Name: name,
				})
			}
		}
	}
	return contexts
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

	var extrapolated []*models.ObservationContext
	if input.ExtrapolateContexts {
		s.cacheMu.RLock()
		cached, ok := s.cache[philosophy.ID]
		s.cacheMu.RUnlock()
		if ok {
			extrapolated = cached
		} else {
			extrapolated = extrapolateObservationContexts(input.Description)
			s.cacheMu.Lock()
			s.cache[philosophy.ID] = extrapolated
			s.cacheMu.Unlock()
		}
	}

	return &models.CreatePhilosophyOutput{Philosophy: philosophy, ExtrapolatedObservationContexts: extrapolated}, nil
}

// Add this method to update the belief system of a self model
func (s *SelfModelService) UpdateSelfModelBeliefSystem(ctx context.Context, selfModelID string, beliefSystem *models.BeliefSystem) error {
	err := s.kvStore.Store(selfModelID, "BeliefSystem", *beliefSystem, 1)
	if err != nil {
		return fmt.Errorf("failed to update belief system: %v", err)
	}
	return nil
}

// UpdatePhilosophy updates an existing philosophy by ID.
func (s *SelfModelService) UpdatePhilosophy(ctx context.Context, input *models.UpdatePhilosophyInput) (*models.UpdatePhilosophyOutput, error) {
	if input.PhilosophyID == "" {
		return nil, fmt.Errorf("philosophy ID cannot be empty")
	}

	storedPhilosophy, err := s.kvStore.Retrieve(input.PhilosophyID, "Philosophy")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve philosophy: %v", err)
	}

	philosophy, ok := storedPhilosophy.(*models.Philosophy)
	if !ok {
		return nil, fmt.Errorf("invalid philosophy data")
	}

	philosophy.Description = input.Description
	philosophy.ExtrapolateContexts = input.ExtrapolateContexts

	err = s.kvStore.Store(philosophy.ID, "Philosophy", *philosophy, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to store updated philosophy: %v", err)
	}

	// Invalidate cache on update
	s.cacheMu.Lock()
	delete(s.cache, philosophy.ID)
	s.cacheMu.Unlock()

	var extrapolated []*models.ObservationContext
	if input.ExtrapolateContexts {
		extrapolated = extrapolateObservationContexts(input.Description)
		s.cacheMu.Lock()
		s.cache[philosophy.ID] = extrapolated
		s.cacheMu.Unlock()
	}

	return &models.UpdatePhilosophyOutput{Philosophy: philosophy, ExtrapolatedObservationContexts: extrapolated}, nil
}

func (s *SelfModelService) Cache() map[string][]*models.ObservationContext {
	return s.cache
}

func (s *SelfModelService) CacheMu() *sync.RWMutex {
	return &s.cacheMu
}
