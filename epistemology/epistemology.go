package epistemology

import "epistemic-me-core/svc/models"

// Epistemology defines an interface for handling and evaluating beliefs through structured epistemological processes.
// Higher level processes like dialectics can orchestrate epistemologies to produce new (and/or) updated beleifs
type Epistemology interface {

	// Validate Belief System, returns an error if a beleif system doesnt meet the invariants of the
	// given epistemology (e.g. PredictiveProcessing Context Not Available)
	ValidateBeliefSystem(models.BeliefSystem) error

	// Validates a Belief, returns an error if a beleif system doesnt meet the invariants of the
	// given epistemology (e.g. PredictiveProcessing Context Not Available)
	ValidateBelief(belief *models.Belief) error

	// This let's a given episteology inform the caller if a given event type is not compatible
	// with a given epistemology and should be ignored
	ValidateEpistemicEvents(events []*models.EpistemicEvent) error

	// Clarify Beleif System, examines all beleifs in a given beleif system and given the available epistemic events
	// updates the given beleifs in the beleif system. This provides the ability
	UpdateBeliefSystem(bs *models.BeliefSystem, events []*models.EpistemicEvent) (*models.BeliefSystem, error)

	// Clarify Beleif, examines a belief appropriately some given event. Optionally can produce new beleifs with epistemic contexts in the update step
	UpdateBelief(belief *models.Belief, contexts []*models.EpistemicContext, events []*models.EpistemicEvent) ([]*models.Belief, []*models.EpistemicContext, error)

	Predict(event []*models.EpistemicEvent) models.EpistemicPrediction
}
