package models

import (
	pbmodels "epistemic-me-core/pb/models"
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

// UpdateBeliefOutput represents an output after updating a belief.
type DeleteBeliefOutput struct {
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

type CreateDeveloperOutput struct {
	Developer Developer `json:"developer"`
}

type CreateUserOutput struct {
	User User `json:"user"`
}

type Developer struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Email     string   `json:"email"`
	APIKeys   []string `json:"api_keys"`
	CreatedAt int64    `json:"created_at"`
	UpdatedAt int64    `json:"updated_at"`
}

func (d *Developer) ToProto() *pbmodels.Developer {
	return &pbmodels.Developer{
		Id:        d.ID,
		Name:      d.Name,
		Email:     d.Email,
		ApiKeys:   d.APIKeys,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

type User struct {
	ID          string `json:"id"`
	DeveloperID string `json:"developer_id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

func (u *User) ToProto() *pbmodels.User {
	return &pbmodels.User{
		Id:          u.ID,
		DeveloperId: u.DeveloperID,
		Name:        u.Name,
		Email:       u.Email,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}
