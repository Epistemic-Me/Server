package metric

import "epistemic-me-core/svc/models"

// Clarification is a metric of how much a belief has been
// engaged with by a user through using some Dialectical Strategy.
// A good proxy for this, is how many times a belief has been
// udpated as part of engaging within a dialectical strategy
func ComputeClarifiedMetric(versionedBeliefs [][]models.Belief) (models.Metric, error) {
	var clarification_scores []models.Metric
	for _, beliefVersions := range versionedBeliefs {
		clarifiedBeliefs := 0
		allBeliefs := len(beliefVersions)
		// for a given belief we track how many versions of the belief have been clarifications
		// todo: @deen, this score can never reach 100% since we treat the first belief as a
		// alway a statement. Will need to adust this computaiton to be a better metric once
		// clarification is sufficiently high
		for _, beliefVersion := range beliefVersions {
			if beliefVersion.Type == models.Causal {
				clarifiedBeliefs += 1
			}
		}
		clarification_scores = append(clarification_scores, models.Metric{
			Label:       "Belief Clarification Score",
			Numerator:   int32(clarifiedBeliefs),
			Denominator: int32(allBeliefs),
		})
	}
	// Return the average clarification across all beliefs
	return models.Average(clarification_scores)
}
