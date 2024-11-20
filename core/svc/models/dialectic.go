package models

import (
	"encoding/json"
	pbmodels "epistemic-me-core/pb/models"
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

// Add Action type
type ActionType int32

const (
	ActionTypeInvalid ActionType = iota
	ActionTypeAnswerQuestion
	ActionTypeCollectEvidence
	ActionTypeActuateOutcome
)

// Action represents a way to change the world state
type Action struct {
	Type                   ActionType `json:"type"`
	DialecticInteractionID string     `json:"dialectic_interaction_id"`
	ResourceID             string     `json:"resource_id"`
	InterventionID         string     `json:"intervention_id,omitempty"`
	Timestamp              int64      `json:"timestamp"`
}

func (a Action) ToProto() *pbmodels.Action {
	return &pbmodels.Action{
		Type:                   pbmodels.ActionType(a.Type),
		DialecticInteractionId: a.DialecticInteractionID,
		ResourceId:             a.ResourceID,
		InterventionId:         a.InterventionID,
		Timestamp:              a.Timestamp,
	}
}

// Observation represents an observed state after an action
type Observation struct {
	DialecticInteractionID string             `json:"dialectic_interaction_id"`
	Type                   ObservationType    `json:"type"`
	Source                 *Source            `json:"source"`
	StateDistribution      map[string]float32 `json:"state_distribution"`
	Timestamp              int64              `json:"timestamp"`
}

func (o Observation) ToProto() *pbmodels.Observation {
	return &pbmodels.Observation{
		DialecticInteractionId: o.DialecticInteractionID,
		Type:                   pbmodels.ObservationType(o.Type),
		Source:                 o.Source.ToProto(),
		StateDistribution:      o.StateDistribution,
		Timestamp:              o.Timestamp,
	}
}

// InteractionType represents the type of dialectical interaction.
type InteractionType int32

const (
	// InteractionTypeInvalid represents an invalid interaction type
	InteractionTypeInvalid InteractionType = iota
	// InteractionTypeQuestionAnswer represents a question-answer interaction
	InteractionTypeQuestionAnswer
	// InteractionTypeHypothesisEvidence represents a hypothesis-evidence interaction
	InteractionTypeHypothesisEvidence
	// InteractionTypeActionOutcome represents an action-outcome interaction
	InteractionTypeActionOutcome
)

func (it InteractionType) ToProto() pbmodels.InteractionType {
	switch it {
	case InteractionTypeQuestionAnswer:
		return pbmodels.InteractionType_QUESTION_ANSWER
	case InteractionTypeHypothesisEvidence:
		return pbmodels.InteractionType_HYPOTHESIS_EVIDENCE
	case InteractionTypeActionOutcome:
		return pbmodels.InteractionType_ACTION_OUTCOME
	default:
		return pbmodels.InteractionType_INTERACTION_TYPE_INVALID
	}
}

// Add State type
type State struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Properties  map[string]float32 `json:"properties"`
}

func (s State) ToProto() *pbmodels.State {
	return &pbmodels.State{
		Id:          s.ID,
		Name:        s.Name,
		Description: s.Description,
		Properties:  s.Properties,
	}
}

func statesToProto(states []State) []*pbmodels.State {
	result := make([]*pbmodels.State, len(states))
	for i, s := range states {
		result[i] = s.ToProto()
	}
	return result
}

// Update DialecticalInteraction struct
type DialecticalInteraction struct {
	ID                   string                       `json:"id"`
	Status               DialecticalInteractionStatus `json:"status"`
	Type                 InteractionType              `json:"type"`
	Question             Question                     `json:"question"`
	UserAnswer           UserAnswer                   `json:"user_answer"`
	PredictedObservation *Observation                 `json:"predicted_observation,omitempty"`
	Action               *Action                      `json:"action,omitempty"`
	Observation          *Observation                 `json:"observation,omitempty"`
	Discrepancy          *Discrepancy                 `json:"discrepancy,omitempty"`
	UpdatedAtMillisUTC   int64                        `json:"updated_at_millis_utc"`
}

func (di DialecticalInteraction) ToProto() *pbmodels.DialecticalInteraction {
	proto := &pbmodels.DialecticalInteraction{
		Status:             pbmodels.STATUS(di.Status),
		Type:               pbmodels.InteractionType(di.Type),
		Id:                 di.ID,
		UpdatedAtMillisUtc: di.UpdatedAtMillisUTC,
	}

	if di.PredictedObservation != nil {
		proto.PredictedObservation = di.PredictedObservation.ToProto()
	}
	if di.Action != nil {
		proto.Action = di.Action.ToProto()
	}
	if di.Observation != nil {
		proto.Observation = di.Observation.ToProto()
	}
	if di.Discrepancy != nil {
		proto.Discrepancy = di.Discrepancy.ToProto()
	}

	switch di.Type {
	case InteractionTypeQuestionAnswer:
		proto.Interaction = &pbmodels.DialecticalInteraction_QuestionAnswer{
			QuestionAnswer: &pbmodels.QuestionAnswerInteraction{
				Question: di.Question.ToProto(),
				Answer:   di.UserAnswer.ToProto(),
			},
		}
	case InteractionTypeHypothesisEvidence:
		// Add implementation for hypothesis evidence
	case InteractionTypeActionOutcome:
		// Add implementation for action outcome
	}

	return proto
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

// Dialectic represents a session to determine and clarify a user's beliefs.
type Dialectic struct {
	ID               string                   `json:"id"`
	SelfModelID      string                   `json:"self_model_id"`
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

func (d *Dialectic) ToProto() *pbmodels.Dialectic {
	protoInteractions := make([]*pbmodels.DialecticalInteraction, len(d.UserInteractions))
	for i, interaction := range d.UserInteractions {
		protoInteractions[i] = interaction.ToProto()
	}

	protoDialectic := &pbmodels.Dialectic{
		Id:               d.ID,
		SelfModelId:      d.SelfModelID,
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
