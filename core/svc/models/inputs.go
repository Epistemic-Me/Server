package models

// ListBeliefsInput represents an input to list beliefs.
type ListBeliefsInput struct {
	UserID    string   `json:"user_id"`
	BeliefIDs []string `json:"belief_ids,omitempty"`
}

// CreateBeliefInput represents an input to create a new belief.
type CreateBeliefInput struct {
	UserID        string `json:"user_id"`
	BeliefContent string `json:"belief_content"`
	DryRun        bool   `json:"dry_run"`
}

// UpdateBeliefInput represents an input to update an existing belieff
type UpdateBeliefInput struct {
	UserID               string     `json:"user_id"`
	BeliefID             string     `json:"belief_id"`
	BeliefType           BeliefType `json:"belief_type"`
	CurrentVersion       int32      `json:"current_version"` // also utilized as idempotency key for update
	UpdatedBeliefContent string     `json:"updated_belief_content"`
	DryRun               bool       `json:"dry_run"`
}

// CreateDialecticInput represents an input to create a new dialectic.
type CreateDialecticInput struct {
	UserID        string        `json:"user_id"`
	DialecticType DialecticType `json:"dialectic_type"`
}

// ListDialecticsInput represents an input to list dialectics.
type ListDialecticsInput struct {
	UserID string `json:"user_id"`
}

// UpdateDialecticInput represents an input to update an existing dialectic.
type UpdateDialecticInput struct {
	UserID      string     `json:"user_id"`
	DialecticID string     `json:"dialectic_id"`
	Answer      UserAnswer `json:"answer"`
	DryRun      bool       `json:"dry_run"`
}
