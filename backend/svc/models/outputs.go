package models

import (
	pbmodels "epistemic-me-backend/pb/models"
)

// ListBeliefsOutput represents an output containing a list of beliefs.
type ListBeliefsOutput struct {
	Beliefs      []*Belief    `json:"beliefs"`
	BeliefSystem BeliefSystem `json:"belief_system"`
}

// CreateBeliefOutput represents an output after creating a new belief.
type CreateBeliefOutput struct {
	Belief       Belief       `json:"belief"`
	BeliefSystem BeliefSystem `json:"belief_system"`
}

// UpdateBeliefOutput represents an output after updating a belief.
type UpdateBeliefOutput struct {
	Belief       Belief       `json:"belief"`
	BeliefSystem BeliefSystem `json:"belief_system"`
}

// CreateDialecticOutput represents an output after creating a new dialectic.
type CreateDialecticOutput struct {
	DialecticID string    `json:"dialectic_id"`
	Dialectic   Dialectic `json:"dialectic"`
}

// ListDialecticsOutput represents an output containing a list of dialectics.
type ListDialecticsOutput struct {
	Dialectics []Dialectic `json:"dialectics"`
}

// UpdateDialecticOutput represents an output after updating a dialectic.
type UpdateDialecticOutput struct {
	Dialectic Dialectic `json:"dialectic"`
}

// GetBeliefSystemOutput represents an output containing a belief system.
type GetBeliefSystemOutput struct {
	BeliefSystem *BeliefSystem `json:"belief_system"`
}

func (o *GetBeliefSystemOutput) ToProto() *pbmodels.BeliefSystem {
	return o.BeliefSystem.ToProto()
}
