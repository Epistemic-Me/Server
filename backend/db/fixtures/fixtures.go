package fixture_models

type BeliefSystemFixture struct {
	BeliefSystem struct {
		Examples []struct {
			Name               string `yaml:"name"`
			ObservationContext []struct {
				ContextName  string `yaml:"context_name"`
				Description  string `yaml:"description"`
				NestedWithin string `yaml:"nested_within"`
			} `yaml:"observation_context"`
			Beliefs []struct {
				BeliefName            string `yaml:"belief_name"`
				Context               string `yaml:"context"`
				Description           string `yaml:"description"`
				PredictedOutcome      string `yaml:"predicted_outcome"`
				CounterfactualOutcome string `yaml:"counterfactual_outcome"`
			} `yaml:"beliefs"`
			Evidence []struct {
				Belief       string `yaml:"belief"`
				Qualitative  string `yaml:"qualitative"`
				Quantitative string `yaml:"quantitative"`
				Research     string `yaml:"research"`
			} `yaml:"evidence"`
			Discrepancy []struct {
				Belief           string   `yaml:"belief"`
				Conditional      string   `yaml:"conditional"`
				EpistemicActions []string `yaml:"epistemic_actions"`
			} `yaml:"discrepancy"`
		} `yaml:"examples"`
	} `yaml:"belief_system"`
}
