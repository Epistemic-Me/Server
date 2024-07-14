package models

// ListBeliefsInput represents an input to list beliefs.
type ListBeliefsInput struct {
	UserID    string   `json:"user_id"`
	BeliefIDs []string `json:"belief_ids,omitempty"`
}

// CreateBeliefInput represents an input to create a new belief.
type CreateBeliefInput struct {
	UserID string `json:"user_id"`
	Belief Belief `json:"belief"`
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
	DialecticID string     `json:"dialectic_id"`
	Answer      UserAnswer `json:"answer"`
}
