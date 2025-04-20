package models

import (
	pbmodels "epistemic-me-core/pb/models"
)

// SelfModel represents a self-model with its associated data
type SelfModel struct {
	ID           string        `json:"id"` // This will be the same as UserID
	Philosophies []string      `json:"philosophies"`
	BeliefSystem *BeliefSystem `json:"belief_system"`
	Dialectics   []*Dialectic  `json:"dialectics"`
}

func (sa *SelfModel) ToProto() *pbmodels.SelfModel {
	protoDialectics := make([]*pbmodels.Dialectic, len(sa.Dialectics))
	for i, d := range sa.Dialectics {
		protoDialectics[i] = d.ToProto()
	}

	protoSelfModel := &pbmodels.SelfModel{
		Id:           sa.ID,
		Philosophies: sa.Philosophies,
		Dialectics:   protoDialectics,
	}

	if sa.BeliefSystem != nil {
		protoSelfModel.BeliefSystem = sa.BeliefSystem.ToProto()
	}

	return protoSelfModel
}

// Philosophy represents a philosophy associated with a self-model
type Philosophy struct {
	ID                  string `json:"id"`
	Description         string `json:"description"`
	ExtrapolateContexts bool   `json:"extrapolate_contexts"`
}

func (p *Philosophy) ToProto() *pbmodels.Philosophy {
	return &pbmodels.Philosophy{
		Id:                  p.ID,
		Description:         p.Description,
		ExtrapolateContexts: p.ExtrapolateContexts,
	}
}

// CreateSelfModelInput represents the input for creating a new self-model
type CreateSelfModelInput struct {
	ID           string   `json:"id"`
	Philosophies []string `json:"philosophies"`
}

// CreateSelfModelOutput represents the output after creating a new self-model
type CreateSelfModelOutput struct {
	SelfModel *SelfModel `json:"self_model"`
}

// GetSelfModelInput represents the input for retrieving a self-model
type GetSelfModelInput struct {
	SelfModelID        string `json:"self_model_id"`
	BypassDeveloperKey bool   `json:"bypass_developer_key"`
}

// GetSelfModelOutput represents the output after retrieving a self-model
type GetSelfModelOutput struct {
	SelfModel *SelfModel `json:"self_model"`
}

// AddPhilosophyInput represents the input for adding a philosophy to a self-model
type AddPhilosophyInput struct {
	SelfModelID  string `json:"self_model_id"`
	PhilosophyID string `json:"philosophy_id"`
}

// AddPhilosophyOutput represents the output after adding a philosophy to a self-model
type AddPhilosophyOutput struct {
	UpdatedSelfModel *SelfModel `json:"updated_self_model"`
}

// CreatePhilosophyInput represents the input for creating a new philosophy
type CreatePhilosophyInput struct {
	Description         string `json:"description"`
	ExtrapolateContexts bool   `json:"extrapolate_contexts"`
}

// CreatePhilosophyOutput represents the output after creating a new philosophy
type CreatePhilosophyOutput struct {
	Philosophy                      *Philosophy           `json:"philosophy"`
	ExtrapolatedObservationContexts []*ObservationContext `json:"extrapolated_observation_contexts,omitempty"`
}

// UpdatePhilosophyInput represents the input for updating an existing philosophy
// PhilosophyID is required, Description and ExtrapolateContexts are the new values
// If a field is empty/zero, it will be updated to that value
// (no partial update logic for now)
type UpdatePhilosophyInput struct {
	PhilosophyID        string `json:"philosophy_id"`
	Description         string `json:"description"`
	ExtrapolateContexts bool   `json:"extrapolate_contexts"`
}

// UpdatePhilosophyOutput represents the output after updating a philosophy
type UpdatePhilosophyOutput struct {
	Philosophy                      *Philosophy           `json:"philosophy"`
	ExtrapolatedObservationContexts []*ObservationContext `json:"extrapolated_observation_contexts,omitempty"`
}
