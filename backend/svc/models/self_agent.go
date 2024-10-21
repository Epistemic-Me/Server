package models

import (
	pbmodels "epistemic-me-backend/pb/models"
)

// SelfAgent represents a self-agent with its associated data
type SelfAgent struct {
	ID           string        `json:"id"` // This will be the same as UserID
	Philosophies []string      `json:"philosophies"`
	BeliefSystem *BeliefSystem `json:"belief_system"`
	Dialectics   []*Dialectic  `json:"dialectics"`
}

func (sa *SelfAgent) ToProto() *pbmodels.SelfAgent {
	protoDialectics := make([]*pbmodels.Dialectic, len(sa.Dialectics))
	for i, d := range sa.Dialectics {
		protoDialectics[i] = d.ToProto()
	}

	protoSelfAgent := &pbmodels.SelfAgent{
		Id:           sa.ID,
		Philosophies: sa.Philosophies,
		Dialectics:   protoDialectics,
	}

	if sa.BeliefSystem != nil {
		protoSelfAgent.BeliefSystem = sa.BeliefSystem.ToProto()
	}

	return protoSelfAgent
}

// Philosophy represents a philosophy associated with a self-agent
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

// CreateSelfAgentInput represents the input for creating a new self-agent
type CreateSelfAgentInput struct {
	ID           string   `json:"id"`
	Philosophies []string `json:"philosophies"`
}

// CreateSelfAgentOutput represents the output after creating a new self-agent
type CreateSelfAgentOutput struct {
	SelfAgent *SelfAgent `json:"self_agent"`
}

// GetSelfAgentInput represents the input for retrieving a self-agent
type GetSelfAgentInput struct {
	SelfAgentID string `json:"self_agent_id"`
}

// GetSelfAgentOutput represents the output after retrieving a self-agent
type GetSelfAgentOutput struct {
	SelfAgent *SelfAgent `json:"self_agent"`
}

// AddPhilosophyInput represents the input for adding a philosophy to a self-agent
type AddPhilosophyInput struct {
	SelfAgentID  string `json:"self_agent_id"`
	PhilosophyID string `json:"philosophy_id"`
}

// AddPhilosophyOutput represents the output after adding a philosophy to a self-agent
type AddPhilosophyOutput struct {
	UpdatedSelfAgent *SelfAgent `json:"updated_self_agent"`
}

// CreatePhilosophyInput represents the input for creating a new philosophy
type CreatePhilosophyInput struct {
	Description         string `json:"description"`
	ExtrapolateContexts bool   `json:"extrapolate_contexts"`
}

// CreatePhilosophyOutput represents the output after creating a new philosophy
type CreatePhilosophyOutput struct {
	Philosophy *Philosophy `json:"philosophy"`
}
