package models

// ListBeliefsInput represents an input to list beliefs.
type ListBeliefsInput struct {
	SelfModelID     string   `json:"self_model_id"`
	BeliefIDs       []string `json:"belief_ids,omitempty"`
	GetBeliefSystem bool     `json:"compute_belief_system"`
}

// CreateBeliefInput represents an input to create a new belief.
type CreateBeliefInput struct {
	SelfModelID    string          `json:"self_model_id"`
	BeliefContent  string          `json:"belief_content"`
	BeliefType     BeliefType      `json:"belief_type"`
	DryRun         bool            `json:"dry_run"`
	BeliefEvidence *BeliefEvidence `json:"evidence,omitempty"`
}

// BeliefEvidence represents evidence for a belief
type BeliefEvidence struct {
	Type              EvidenceType `json:"type"`
	Content           string       `json:"content,omitempty"`
	IsCounter         bool         `json:"is_counter,omitempty"`
	Action            string       `json:"action,omitempty"`
	Outcome           string       `json:"outcome,omitempty"`
	PerspectiveSelves []string     `json:"perspective_selves"`
}

type EvidenceType int

const (
	EvidenceTypeHypothesis EvidenceType = iota
	EvidenceTypeAction
)

// UpdateBeliefInput represents an input to update an existing belief
type UpdateBeliefInput struct {
	SelfModelID          string     `json:"self_model_id"`
	ID                   string     `json:"belief_id"`
	BeliefType           BeliefType `json:"belief_type"`
	CurrentVersion       int32      `json:"current_version"` // also utilized as idempotency key for update
	UpdatedBeliefContent string     `json:"updated_belief_content"`
	DryRun               bool       `json:"dry_run"`
}

// UpdateBeliefInput represents an input to update an existing belief
type DeleteBeliefInput struct {
	SelfModelID         string `json:"self_model_id"`
	ID                  string `json:"belief_id"`
	DryRun              bool   `json:"dry_run"`
	ComputeBeliefSystem bool   `json:"compute_belief_system"`
}

// CreateDialecticInput represents an input to create a new dialectic.
type CreateDialecticInput struct {
	SelfModelID         string             `json:"self_model_id"`
	DialecticType       DialecticType      `json:"dialectic_type"`
	PerspectiveModelIDs []string           `json:"perspective_model_ids,omitempty"`
	LearningObjective   *LearningObjective `json:"learning_objective,omitempty"`
}

// ListDialecticsInput represents an input to list dialectics.
type ListDialecticsInput struct {
	SelfModelID string `json:"self_model_id"`
}

// UpdateDialecticInput represents an input to update an existing dialectic.
type UpdateDialecticInput struct {
	ID             string     `json:"dialectic_id"`
	SelfModelID    string     `json:"self_model_id"`
	Answer         UserAnswer `json:"answer"`
	DryRun         bool       `json:"dry_run"`
	CustomQuestion *string    `json:"custom_question,omitempty"`
	QuestionBlob   string
	AnswerBlob     string
}

// GetBeliefSystemInput represents an input to get belief system details.
type GetBeliefSystemInput struct {
	SelfModelID string `json:"self_model_id"`
}

type CreateDeveloperInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type GetDeveloperInput struct {
	ID string
}
type CreateUserInput struct {
	DeveloperID string `json:"developer_id"`
	Name        string `json:"name"`  // Can be empty
	Email       string `json:"email"` // Can be empty
}
