package main

import (
	"math"
	"strconv"
)

type RangeStateInterpreter struct {
	Properties map[string]string
}

func (r *RangeStateInterpreter) CalculateLikelihood(measurement float64) float64 {
	// Parse min and max values
	min, err := strconv.ParseFloat(r.Properties["min"], 64)
	if err != nil {
		return 0.0
	}
	max, err := strconv.ParseFloat(r.Properties["max"], 64)
	if err != nil {
		return 0.0
	}

	// Calculate center and range width
	center := (min + max) / 2
	width := max - min

	// If measurement is within range
	if measurement >= min && measurement <= max {
		// Calculate distance from center as a proportion of the range width
		distanceFromCenter := math.Abs(measurement - center)
		normalizedDistance := distanceFromCenter / (width / 2)

		// Return value that peaks at 1.0 at center and drops to 0.5 at edges
		return 1.0 - (0.5 * normalizedDistance)
	}

	// For measurements outside range, calculate exponential decay
	distanceOutside := 0.0
	if measurement < min {
		distanceOutside = min - measurement
	} else {
		distanceOutside = measurement - max
	}

	// Return value that decays exponentially with distance outside range
	return 0.5 * math.Exp(-2.0*distanceOutside/width)
}
