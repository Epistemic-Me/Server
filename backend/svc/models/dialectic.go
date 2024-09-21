package models

import (
	"encoding/json"
	pbmodels "epistemic-me-backend/pb/models"
)

// Question represents a request for information from a user.
type Question struct {
	Question           string `json:"question"`
	CreatedAtMillisUTC int64  `json:"created_at_millis_utc"`
}

func (q Question) ToProto() *pbmodels.Question {
	return &pbmodels.Question{
		Question:           q.Question,
		CreatedAtMillisUtc: q.CreatedAtMillisUTC,
	}
}

// UserAnswer represents a user's answer to a question.
type UserAnswer struct {
	UserAnswer         string `json:"user_answer"`
	CreatedAtMillisUTC int64  `json:"created_at_millis_utc"`
}

func (ua UserAnswer) ToProto() *pbmodels.UserAnswer {
	return &pbmodels.UserAnswer{
		UserAnswer:         ua.UserAnswer,
		CreatedAtMillisUtc: ua.CreatedAtMillisUTC,
	}
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
	DialecticTypeInvalid DialecticType = iota
	DialecticTypeDefault
	DialecticTypeSleepDietExercise
)

func (d DialecticType) ToProto() pbmodels.DialecticType {
	switch d {
	case DialecticTypeDefault:
		return pbmodels.DialecticType_DEFAULT
	case DialecticTypeSleepDietExercise:
		return pbmodels.DialecticType_SLEEP_DIET_EXERCISE
	default:
		return pbmodels.DialecticType_INVALID
	}
}

func DialecticTypeFromProto(d pbmodels.DialecticType) DialecticType {
	switch d {
	case pbmodels.DialecticType_DEFAULT:
		return DialecticTypeDefault
	case pbmodels.DialecticType_SLEEP_DIET_EXERCISE:
		return DialecticTypeSleepDietExercise
	default:
		return DialecticTypeInvalid
	}
}

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

func (a Agent) ToProto() *pbmodels.Agent {
	var protoAgentType pbmodels.Agent_AgentType
	switch a.AgentType {
	case AgentTypeGPTLatest:
		protoAgentType = pbmodels.Agent_AGENT_TYPE_GPT_LATEST
	default:
		protoAgentType = pbmodels.Agent_AGENT_TYPE_INVALID
	}

	var protoDialecticType pbmodels.DialecticType
	switch a.DialecticType {
	case DialecticTypeDefault:
		protoDialecticType = pbmodels.DialecticType_DEFAULT
	case DialecticTypeSleepDietExercise:
		protoDialecticType = pbmodels.DialecticType_SLEEP_DIET_EXERCISE
	default:
		protoDialecticType = pbmodels.DialecticType_INVALID
	}

	return &pbmodels.Agent{
		AgentType:     protoAgentType,
		DialecticType: protoDialecticType,
	}
}

// DialecticalInteraction represents a question and answer interaction.
type DialecticalInteraction struct {
	Status             DialecticalInteractionStatus `json:"status"`
	Question           Question                     `json:"question"`
	UserAnswer         UserAnswer                   `json:"user_answer"`
	Beliefs            []Belief                     `json:"beliefs"`
	UpdatedAtMillisUTC int64                        `json:"updated_at_millis_utc"`
}

func (di DialecticalInteraction) ToProto() *pbmodels.DialecticalInteraction {
	protoBeliefs := make([]*pbmodels.Belief, len(di.Beliefs))
	for i, belief := range di.Beliefs {
		protoBeliefs[i] = belief.ToProto()
	}

	var protoStatus pbmodels.DialecticalInteraction_STATUS
	switch di.Status {
	case StatusPendingAnswer:
		protoStatus = pbmodels.DialecticalInteraction_STATUS_PENDING_ANSWER
	case StatusAnswered:
		protoStatus = pbmodels.DialecticalInteraction_STATUS_ANSWERED
	default:
		protoStatus = pbmodels.DialecticalInteraction_STATUS_INVALID
	}

	return &pbmodels.DialecticalInteraction{
		Status:             protoStatus,
		Question:           di.Question.ToProto(),
		UserAnswer:         di.UserAnswer.ToProto(),
		Beliefs:            protoBeliefs,
		UpdatedAtMillisUtc: di.UpdatedAtMillisUTC,
	}
}

// Dialectic represents a session to determine and clarify a user's beliefs.
type Dialectic struct {
	ID               string                   `json:"id"`
	UserID           string                   `json:"user_id"`
	Agent            Agent                    `json:"agent"`
	UserInteractions []DialecticalInteraction `json:"user_interactions"`
	BeliefSystem     *BeliefSystem            `json:"belief_system"`
	Analysis         *BeliefAnalysis          `json:"analysis,omitempty"`
}

func (d *Dialectic) MarshalBinary() ([]byte, error) {
	return json.Marshal(d)
}

func (d *Dialectic) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, d)
}

func (d Dialectic) ToProto() *pbmodels.Dialectic {
	protoInteractions := make([]*pbmodels.DialecticalInteraction, len(d.UserInteractions))
	for i, interaction := range d.UserInteractions {
		protoInteractions[i] = interaction.ToProto()
	}

	protoDialectic := &pbmodels.Dialectic{
		Id:               d.ID,
		UserId:           d.UserID,
		Agent:            d.Agent.ToProto(),
		UserInteractions: protoInteractions,
	}

	if d.BeliefSystem != nil {
		protoDialectic.BeliefSystem = d.BeliefSystem.ToProto()
	}

	if d.Analysis != nil {
		protoDialectic.Analysis = d.Analysis.ToProto()
	}

	return protoDialectic
}

// BeliefAnalysis represents the analysis of a belief system.
type BeliefAnalysis struct {
	Coherence       float32  `json:"coherence"`
	Consistency     float32  `json:"consistency"`
	Falsifiability  float32  `json:"falsifiability"`
	OverallScore    float32  `json:"overall_score"`
	Feedback        string   `json:"feedback"`
	Recommendations []string `json:"recommendations"`
	VerifiedBeliefs []string `json:"verified_beliefs"`
}

func (ba BeliefAnalysis) ToProto() *pbmodels.BeliefAnalysis {
	return &pbmodels.BeliefAnalysis{
		Coherence:       ba.Coherence,
		Consistency:     ba.Consistency,
		Falsifiability:  ba.Falsifiability,
		OverallScore:    ba.OverallScore,
		Feedback:        ba.Feedback,
		Recommendations: ba.Recommendations,
		VerifiedBeliefs: ba.VerifiedBeliefs,
	}
}
