package models

// EpistemicEvent represents some event that may provide new information
// to update beleifs. This can be refactored in the future to specify
// explicit inputs and outputs, but for now a given epistemology
// can evalute events for inputs and optional outputs
type EpistemicEvent struct {
	QuestionAnswerInteraction *QuestionAnswerInteraction
}

// Data structure used to house prediction
// todo: refactor dialectic to include observations, discrepencies etc, from Predictive Processing
type EpistemicPrediction struct {
	UserAnswer        *UserAnswer
	PredictionContext *PredictionContext
}
