package svc

import (
	"context"
	"fmt"

	"github.com/your-project/ai"
	"github.com/your-project/models"
)

type DialecticService struct {
	processor     *ai.DialecticProcessor
	computeScaler *ai.ComputeScaler
}

func NewDialecticService(processor *ai.DialecticProcessor, scaler *ai.ComputeScaler) *DialecticService {
	return &DialecticService{
		processor:     processor,
		computeScaler: scaler,
	}
}

func (s *DialecticService) UpdateDialectic(ctx context.Context, req *models.UpdateDialecticRequest) (*models.UpdateDialecticResponse, error) {
	if req == nil || req.Interaction == nil {
		return nil, fmt.Errorf("invalid request: request or interaction is nil")
	}

	// Determine compute level
	computeLevel, err := s.processor.determineComputeLevel(req.Interaction)
	if err != nil {
		return nil, fmt.Errorf("failed to determine compute level: %w", err)
	}

	// Generate ontology with compute constraints
	ontology, err := s.processor.GenerateOntology(req.Interaction, computeLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ontology: %w", err)
	}

	// Apply compute constraints
	if err := s.computeScaler.ApplyComputeConstraints(computeLevel, ontology); err != nil {
		return nil, fmt.Errorf("failed to apply compute constraints: %w", err)
	}

	// Process beliefs with scaled compute
	updatedBeliefs, err := s.processor.ProcessBeliefs(ontology, computeLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to process beliefs: %w", err)
	}

	return &models.UpdateDialecticResponse{
		UpdatedBeliefs: updatedBeliefs,
		ComputeLevel:   int32(computeLevel),
	}, nil
}
