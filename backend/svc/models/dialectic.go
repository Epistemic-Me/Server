package models

// Question represents a request for information from a user.
type Question struct {
	Question           string `json:"question"`
	CreatedAtMillisUTC int64  `json:"created_at_millis_utc"`
}

// UserAnswer represents a user's answer to a question.
type UserAnswer struct {
	UserAnswer         string `json:"user_answer"`
	CreatedAtMillisUTC int64  `json:"created_at_millis_utc"`
}

// DialecticalInteraction represents a question and answer interaction.
type DialecticalInteraction struct {
	Status             DialecticalInteractionStatus `json:"status"`
	Question           Question                     `json:"question"`
	UserAnswer         UserAnswer                   `json:"user_answer"`
	Beliefs            []Belief                     `json:"beliefs"`
	UpdatedAtMillisUTC int64                        `json:"updated_at_millis_utc"`
}

// DialecticalInteractionStatus represents the status of the interaction.
type DialecticalInteractionStatus int32

const (
	StatusInvalid       DialecticalInteractionStatus = 0
	StatusPendingAnswer DialecticalInteractionStatus = 1
	StatusAnswered      DialecticalInteractionStatus = 2
)

// DialecticType represents the type of dialectic strategy.
type DialecticType int32

const (
	DialecticTypeInvalid DialecticType = 0
	Default              DialecticType = 1
	Hegelian             DialecticType = 2
)

// AgentType represents the type of agent.
type AgentType int32

const (
	AgentTypeInvalid   AgentType = 0
	AgentTypeGPTLatest AgentType = 1
)

// Agent represents the system or user interacting with the user.
type Agent struct {
	AgentType     AgentType     `json:"agent_type"`
	DialecticType DialecticType `json:"dialectic_type"`
}

// Dialectic represents a session to determine and clarify a user's beliefs.
type Dialectic struct {
	ID               string                   `json:"id"`
	UserID           string                   `json:"user_id"`
	Agent            Agent                    `json:"agent"`
	UserInteractions []DialecticalInteraction `json:"user_interactions"`
}
