package ai

import (
	"fmt"
)

type BeliefOntology struct {
	Nodes []OntologyNode
	Edges []OntologyEdge
	Depth int
}

type ComputeScaler struct {
	levels []ComputeLevel
}

func NewComputeScaler(levels []ComputeLevel) *ComputeScaler {
	return &ComputeScaler{
		levels: levels,
	}
}

func (cs *ComputeScaler) ScaleCompute(ambiguityScore float64) (ComputeLevel, error) {
	if ambiguityScore < 0 || ambiguityScore > 1 {
		return ComputeLevel{}, fmt.Errorf("ambiguity score must be between 0 and 1, got %f", ambiguityScore)
	}

	levelIndex := int(ambiguityScore * float64(len(cs.levels)-1))
	return cs.levels[levelIndex], nil
}

func (cs *ComputeScaler) ApplyComputeConstraints(level ComputeLevel, ontology *BeliefOntology) error {
	if ontology == nil {
		return fmt.Errorf("ontology cannot be nil")
	}

	if err := ontology.PruneToDepth(level.maxContextDepth); err != nil {
		return fmt.Errorf("failed to prune ontology: %w", err)
	}

	if err := ontology.LimitBranches(level.maxBranches); err != nil {
		return fmt.Errorf("failed to limit branches: %w", err)
	}

	if err := ontology.LimitBeliefs(level.maxBeliefs); err != nil {
		return fmt.Errorf("failed to limit beliefs: %w", err)
	}

	return nil
}
