package models

import pbmodels "epistemic-me-core/pb/models"

type EpistemicContext struct {
	// optional field to associate a given epistemic context with a particular beleif or set of beleifs
	AssociatedBeleifs           []string
	PredictiveProcessingContext *PredictiveProcessingContext `json:"context,omitempty"`
}

func (ec EpistemicContext) ToProto() *pbmodels.EpistemicContext {
	if ec.PredictiveProcessingContext != nil {
		return &pbmodels.EpistemicContext{
			Context: ec.PredictiveProcessingContext.ToProto(),
		}
	}
	return nil
}

type DialecticResponse struct {
	SelfModelID    string
	NewInteraction *DialecticalInteraction
}

// EpistemicEvent represents some event that may provide new information
// to update beleifs. This can be refactored in the future to specify
// explicit inputs and outputs, but for now a given epistemology
// can evalute events for inputs and optional outputs
type DialecticEvent struct {
	SelfModelID          string
	PreviousInteractions []DialecticalInteraction
}
