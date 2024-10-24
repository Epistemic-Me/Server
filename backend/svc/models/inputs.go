package models

// ListBeliefsInput represents an input to list beliefs.
type ListBeliefsInput struct {
	SelfModelID string   `json:"self_model_id"`
	BeliefIDs   []string `json:"belief_ids,omitempty"`
}

// CreateBeliefInput represents an input to create a new belief.
type CreateBeliefInput struct {
	SelfModelID   string `json:"self_model_id"`
	BeliefContent string `json:"belief_content"`
	DryRun        bool   `json:"dry_run"`
}

// UpdateBeliefInput represents an input to update an existing belief
type UpdateBeliefInput struct {
	SelfModelID          string     `json:"self_model_id"`
	ID                   string     `json:"belief_id"`
	BeliefType           BeliefType `json:"belief_type"`
	CurrentVersion       int32      `json:"current_version"` // also utilized as idempotency key for update
	UpdatedBeliefContent string     `json:"updated_belief_content"`
	DryRun               bool       `json:"dry_run"`
}

// CreateDialecticInput represents an input to create a new dialectic.
type CreateDialecticInput struct {
	SelfModelID   string        `json:"self_model_id"`
	DialecticType DialecticType `json:"dialectic_type"`
}

// ListDialecticsInput represents an input to list dialectics.
type ListDialecticsInput struct {
	SelfModelID string `json:"self_model_id"`
}

// UpdateDialecticInput represents an input to update an existing dialectic.
type UpdateDialecticInput struct {
	ID          string     `json:"dialectic_id"`
	SelfModelID string     `json:"self_model_id"`
	Answer      UserAnswer `json:"answer"`
	DryRun      bool       `json:"dry_run"`
}

// GetBeliefSystemInput represents an input to get belief system details.
type GetBeliefSystemInput struct {
	SelfModelID string `json:"self_model_id"`
}
