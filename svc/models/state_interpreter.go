package models

import (
	"fmt"
	"math"

	pb "epistemic-me-core/pb/models"
)

// StateInterpreter defines how to interpret state properties and calculate likelihoods
type StateInterpreter interface {
	// Calculate likelihood of measurement given a state
	CalculateLikelihood(measurement float64, state *pb.State) float64
	// Validate that state properties match expected format
	ValidateState(state *pb.State) error
}

// RangeStateInterpreter handles states defined by min/max ranges
type RangeStateInterpreter struct {
	DistributionType string
}

func NewRangeStateInterpreter(distributionType string) StateInterpreter {
	return &RangeStateInterpreter{
		DistributionType: distributionType,
	}
}

func (r *RangeStateInterpreter) CalculateLikelihood(measurement float64, state *pb.State) float64 {
	minVal, exists := state.Properties["min_value"]
	maxVal, exists2 := state.Properties["max_value"]
	if !exists || !exists2 {
		return 0.0
	}

	if measurement >= float64(minVal) && measurement <= float64(maxVal) {
		return 15.0 // Increased from 5.0 to achieve probability > 0.9
	}

	// Calculate distance from nearest range boundary
	var distance float64
	if measurement < float64(minVal) {
		distance = float64(minVal) - measurement
	} else {
		distance = measurement - float64(maxVal)
	}

	// More gradual exponential decay for out-of-range values
	return math.Exp(-1.0 * distance) // Adjusted decay factor for better contrast
}

func (r *RangeStateInterpreter) ValidateState(state *pb.State) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}

	// Check min_value and max_value
	minValue, okMin := state.Properties["min_value"]
	maxValue, okMax := state.Properties["max_value"]
	if !okMin || !okMax {
		return fmt.Errorf("range state must have 'min_value' and 'max_value' properties")
	}
	if minValue >= maxValue {
		return fmt.Errorf("min_value must be less than max_value")
	}
	return nil
}
