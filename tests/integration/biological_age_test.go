package integration

import (
	"testing"

	pb "epistemic-me-core/pb/models"
	"epistemic-me-core/svc/models"

	"github.com/stretchr/testify/assert"
)

func TestBiologicalAgeContext(t *testing.T) {
	// Create an observation context for grip strength
	ctx := &pb.ObservationContext{
		Id:   "grip_strength",
		Name: "Grip Strength Assessment",
		PossibleStates: []*pb.State{
			{
				Id:   "strong_20s",
				Name: "Strong (20s)",
				Properties: map[string]float32{
					"min_value": 85.0,
					"max_value": 120.0,
					"age_min":   20.0,
					"age_max":   29.0,
				},
			},
			{
				Id:   "moderate_30s",
				Name: "Moderate (30s)",
				Properties: map[string]float32{
					"min_value": 80.0,
					"max_value": 85.0,
					"age_min":   30.0,
					"age_max":   39.0,
				},
			},
			{
				Id:   "declining_40s",
				Name: "Declining (40s)",
				Properties: map[string]float32{
					"min_value": 75.0,
					"max_value": 80.0,
					"age_min":   40.0,
					"age_max":   49.0,
				},
			},
		},
	}

	// Create entropy calculator and add context
	calc := models.NewEntropyCalculator()
	calc.AddContext(ctx)

	// Test initial entropy (should be maximum for uniform distribution)
	initialEntropy := calc.CalculateContextEntropy(ctx)
	assert.InDelta(t, 1.58, initialEntropy, 0.01) // log2(3) ≈ 1.58 for 3 states

	// Add a measurement indicating strong grip strength
	measurement := &pb.Resource{
		Type: pb.ResourceType_MEASUREMENT_DATA,
		Metadata: map[string]string{
			"measurement_value": "90.0",
			"measurement_type":  "grip_strength",
			"unit":              "kg",
		},
	}

	// Update entropy with measurement
	err := calc.UpdateEntropy(ctx.Id, measurement)
	assert.NoError(t, err)

	// Entropy should decrease as we become more certain about the state
	updatedEntropy := calc.CalculateContextEntropy(ctx)
	assert.Less(t, updatedEntropy, initialEntropy)
}

func TestMultipleIndicators(t *testing.T) {
	// Create contexts for different biological age indicators
	gripCtx := &pb.ObservationContext{
		Id:   "grip_strength",
		Name: "Grip Strength Assessment",
		PossibleStates: []*pb.State{
			{
				Id:   "young_adult",
				Name: "Young Adult (20s-30s)",
				Properties: map[string]float32{
					"min_value": 85.0,
					"max_value": 120.0,
					"age_min":   20.0,
					"age_max":   39.0,
				},
			},
			{
				Id:   "middle_age",
				Name: "Middle Age (40s-50s)",
				Properties: map[string]float32{
					"min_value": 75.0,
					"max_value": 85.0,
					"age_min":   40.0,
					"age_max":   59.0,
				},
			},
		},
	}

	reactionCtx := &pb.ObservationContext{
		Id:   "reaction_time",
		Name: "Reaction Time Assessment",
		PossibleStates: []*pb.State{
			{
				Id:   "quick",
				Name: "Quick Response (20s-30s)",
				Properties: map[string]float32{
					"min_value": 0.15,
					"max_value": 0.25,
					"age_min":   20.0,
					"age_max":   39.0,
				},
			},
			{
				Id:   "moderate",
				Name: "Moderate Response (40s-50s)",
				Properties: map[string]float32{
					"min_value": 0.25,
					"max_value": 0.35,
					"age_min":   40.0,
					"age_max":   59.0,
				},
			},
		},
	}

	// Create entropy calculator and add contexts
	calc := models.NewEntropyCalculator()
	calc.AddContext(gripCtx)
	calc.AddContext(reactionCtx)

	// Test initial progress across both contexts
	contexts := []*pb.ObservationContext{gripCtx, reactionCtx}
	initialProgress := calc.CalculateProgress(contexts)
	assert.Equal(t, 0.0, initialProgress)

	// Add measurements
	gripMeasurement := &pb.Resource{
		Type: pb.ResourceType_MEASUREMENT_DATA,
		Metadata: map[string]string{
			"measurement_value": "90.0",
			"measurement_type":  "grip_strength",
			"unit":              "kg",
		},
	}

	reactionMeasurement := &pb.Resource{
		Type: pb.ResourceType_MEASUREMENT_DATA,
		Metadata: map[string]string{
			"measurement_value": "0.20",
			"measurement_type":  "reaction_time",
			"unit":              "seconds",
		},
	}

	// Update entropy with measurements
	err := calc.UpdateEntropy(gripCtx.Id, gripMeasurement)
	assert.NoError(t, err)
	err = calc.UpdateEntropy(reactionCtx.Id, reactionMeasurement)
	assert.NoError(t, err)

	// Test final progress (should indicate learning)
	finalProgress := calc.CalculateProgress(contexts)
	assert.Greater(t, finalProgress, 0.0)
	assert.LessOrEqual(t, finalProgress, 1.0)
}

func TestStateTransitions(t *testing.T) {
	// Create an observation context with age-related states
	ctx := &pb.ObservationContext{
		Id:   "biological_age",
		Name: "Biological Age Assessment",
		PossibleStates: []*pb.State{
			{
				Id:   "young",
				Name: "Biologically Young",
				Properties: map[string]float32{
					"age_min":              20.0,
					"age_max":              35.0,
					"confidence_threshold": 0.7,
				},
			},
			{
				Id:   "middle",
				Name: "Average Age",
				Properties: map[string]float32{
					"age_min":              36.0,
					"age_max":              50.0,
					"confidence_threshold": 0.7,
				},
			},
			{
				Id:   "advanced",
				Name: "Biologically Advanced",
				Properties: map[string]float32{
					"age_min":              51.0,
					"age_max":              65.0,
					"confidence_threshold": 0.7,
				},
			},
		},
	}

	// Create entropy calculator and add context
	calc := models.NewEntropyCalculator()
	calc.AddContext(ctx)

	// Add a series of measurements that should trigger state transitions
	measurements := []*pb.Resource{
		{
			Type: pb.ResourceType_MEASUREMENT_DATA,
			Metadata: map[string]string{
				"measurement_value": "25.0", // Young biological age
				"measurement_type":  "biological_age",
				"unit":              "years",
			},
		},
		{
			Type: pb.ResourceType_MEASUREMENT_DATA,
			Metadata: map[string]string{
				"measurement_value": "30.0", // Still young biological age
				"measurement_type":  "biological_age",
				"unit":              "years",
			},
		},
		{
			Type: pb.ResourceType_MEASUREMENT_DATA,
			Metadata: map[string]string{
				"measurement_value": "45.0", // Middle biological age
				"measurement_type":  "biological_age",
				"unit":              "years",
			},
		},
	}

	// Process measurements and check entropy changes
	var prevEntropy float64
	for i, m := range measurements {
		err := calc.UpdateEntropy(ctx.Id, m)
		assert.NoError(t, err)

		entropy := calc.CalculateContextEntropy(ctx)
		if i > 0 {
			// Entropy should change with each new measurement
			assert.NotEqual(t, prevEntropy, entropy)
		}
		prevEntropy = entropy
	}

	// Final entropy should indicate increased certainty
	finalEntropy := calc.CalculateContextEntropy(ctx)
	assert.Less(t, finalEntropy, 1.58) // Less than initial maximum entropy
}
