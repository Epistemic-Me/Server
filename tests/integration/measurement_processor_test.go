package integration

import (
	"testing"
	"time"

	pb "epistemic-me-core/pb/models"
	"epistemic-me-core/svc/models"

	"github.com/stretchr/testify/assert"
)

func TestBatchProcessing(t *testing.T) {
	processor := models.NewMeasurementProcessor()

	// Create contexts for different measurement types
	gripCtx := &pb.ObservationContext{
		Id:   "grip_strength",
		Name: "Grip Strength Assessment",
		PossibleStates: []*pb.State{
			{
				Id:   "strong",
				Name: "Strong",
				Properties: map[string]float32{
					"min_value": 85.0,
					"max_value": 120.0,
					"age_min":   20.0,
					"age_max":   35.0,
				},
			},
			{
				Id:   "moderate",
				Name: "Moderate",
				Properties: map[string]float32{
					"min_value": 75.0,
					"max_value": 85.0,
					"age_min":   36.0,
					"age_max":   50.0,
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
				Name: "Quick",
				Properties: map[string]float32{
					"min_value": 0.15,
					"max_value": 0.25,
					"age_min":   20.0,
					"age_max":   35.0,
				},
			},
			{
				Id:   "moderate",
				Name: "Moderate",
				Properties: map[string]float32{
					"min_value": 0.25,
					"max_value": 0.35,
					"age_min":   36.0,
					"age_max":   50.0,
				},
			},
		},
	}

	processor.AddContext("grip_strength", gripCtx)
	processor.AddContext("reaction_time", reactionCtx)

	// Test batch processing with multiple measurement types
	measurements := []*pb.Resource{
		{
			Type: pb.ResourceType_MEASUREMENT_DATA,
			Metadata: map[string]string{
				"measurement_value": "90.0",
				"measurement_type":  "grip_strength",
				"unit":              "kg",
				"timestamp":         time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
			},
		},
		{
			Type: pb.ResourceType_MEASUREMENT_DATA,
			Metadata: map[string]string{
				"measurement_value": "0.20",
				"measurement_type":  "reaction_time",
				"unit":              "seconds",
				"timestamp":         time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
			},
		},
		{
			Type: pb.ResourceType_MEASUREMENT_DATA,
			Metadata: map[string]string{
				"measurement_value": "85.0",
				"measurement_type":  "grip_strength",
				"unit":              "kg",
				"timestamp":         time.Now().Format(time.RFC3339),
			},
		},
	}

	// Process measurements
	err := processor.ProcessMeasurements(measurements)
	assert.NoError(t, err)

	// Test error handling for unknown measurement type
	invalidMeasurements := []*pb.Resource{
		{
			Type: pb.ResourceType_MEASUREMENT_DATA,
			Metadata: map[string]string{
				"measurement_value": "100.0",
				"measurement_type":  "unknown_type",
				"unit":              "unknown",
			},
		},
	}

	err = processor.ProcessMeasurements(invalidMeasurements)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no context found for measurement type")
}

func TestAgeEstimation(t *testing.T) {
	processor := models.NewMeasurementProcessor()

	// Create context for grip strength
	gripCtx := &pb.ObservationContext{
		Id:   "grip_strength",
		Name: "Grip Strength Assessment",
		PossibleStates: []*pb.State{
			{
				Id:   "young",
				Name: "Young Adult",
				Properties: map[string]float32{
					"min_value": 85.0,
					"max_value": 120.0,
					"age_min":   20.0,
					"age_max":   35.0,
				},
			},
			{
				Id:   "middle",
				Name: "Middle Age",
				Properties: map[string]float32{
					"min_value": 75.0,
					"max_value": 85.0,
					"age_min":   36.0,
					"age_max":   50.0,
				},
			},
		},
	}

	processor.AddContext("grip_strength", gripCtx)

	// Test initial state (no measurements)
	estimate, err := processor.GetAgeEstimate()
	assert.Error(t, err)
	assert.Nil(t, estimate)

	// Add a measurement
	measurements := []*pb.Resource{
		{
			Type: pb.ResourceType_MEASUREMENT_DATA,
			Metadata: map[string]string{
				"measurement_value": "90.0",
				"measurement_type":  "grip_strength",
				"unit":              "kg",
			},
		},
	}

	err = processor.ProcessMeasurements(measurements)
	assert.NoError(t, err)

	// Test age estimate after measurement
	estimate, err = processor.GetAgeEstimate()
	assert.NoError(t, err)
	assert.NotNil(t, estimate)
	assert.Equal(t, 20.0, estimate.MinAge)
	assert.Equal(t, 35.0, estimate.MaxAge)
	assert.Greater(t, estimate.Confidence, 0.0)
	assert.Contains(t, estimate.Sources, "grip_strength")
}

func TestSuggestionGeneration(t *testing.T) {
	processor := models.NewMeasurementProcessor()

	// Create contexts for different measurement types
	gripCtx := &pb.ObservationContext{
		Id:   "grip_strength",
		Name: "Grip Strength Assessment",
		PossibleStates: []*pb.State{
			{
				Id:   "strong",
				Name: "Strong",
				Properties: map[string]float32{
					"min_value": 85.0,
					"max_value": 120.0,
					"age_min":   20.0,
					"age_max":   35.0,
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
				Name: "Quick",
				Properties: map[string]float32{
					"min_value": 0.15,
					"max_value": 0.25,
					"age_min":   20.0,
					"age_max":   35.0,
				},
			},
		},
	}

	processor.AddContext("grip_strength", gripCtx)
	processor.AddContext("reaction_time", reactionCtx)

	// Test initial suggestions (no measurements)
	suggestions := processor.GetSuggestions()
	assert.Len(t, suggestions, 2)
	assert.Contains(t, suggestions, "Need initial grip_strength measurement")
	assert.Contains(t, suggestions, "Need initial reaction_time measurement")

	// Add one measurement
	measurements := []*pb.Resource{
		{
			Type: pb.ResourceType_MEASUREMENT_DATA,
			Metadata: map[string]string{
				"measurement_value": "90.0",
				"measurement_type":  "grip_strength",
				"unit":              "kg",
			},
		},
	}

	err := processor.ProcessMeasurements(measurements)
	assert.NoError(t, err)

	// Test suggestions after partial measurements
	suggestions = processor.GetSuggestions()
	assert.Contains(t, suggestions, "Need initial reaction_time measurement")
}
