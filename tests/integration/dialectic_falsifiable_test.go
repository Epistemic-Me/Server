package integration

import (
	"encoding/json"
	"testing"
	"time"

	pb "epistemic-me-core/pb/models"
	"epistemic-me-core/svc/models"
	"github.com/stretchr/testify/assert"
)

func TestBiologicalAgeBelief(t *testing.T) {
	// Set up observation contexts for age ranges
	observationCtx := &pb.ObservationContext{
		Id:          "grip_strength_age",
		Name:        "Grip Strength Age Range",
		Description: "Biological age ranges based on grip strength measurements",
		PossibleStates: []*pb.State{
			{
				Id:   "20s",
				Name: "Male in 20s",
				Properties: map[string]float32{
					"range_start": 85,
					"range_end":   127,
				},
			},
			{
				Id:   "teens",
				Name: "Male in Teens",
				Properties: map[string]float32{
					"range_start": 81,
					"range_end":   85,
				},
			},
			{
				Id:   "30s",
				Name: "Male in 30s",
				Properties: map[string]float32{
					"range_start": 79,
					"range_end":   81,
				},
			},
			{
				Id:   "40s",
				Name: "Male in 40s",
				Properties: map[string]float32{
					"range_start": 76,
					"range_end":   79,
				},
			},
			{
				Id:   "50s",
				Name: "Male in 50s",
				Properties: map[string]float32{
					"range_start": 67,
					"range_end":   76,
				},
			},
			{
				Id:   "60s",
				Name: "Male in 60s",
				Properties: map[string]float32{
					"range_start": 60,
					"range_end":   67,
				},
			},
		},
		CurrentStateId: "", // Initially unknown
	}

	// Create initial falsifiable belief with no evidence
	belief := &pb.Belief{
		Type: pb.BeliefType_FALSIFIABLE,
		Content: []*pb.Content{{
			RawStr: "User has Grip Strength biological age in one of the defined ranges",
		}},
	}

	// Test initial state (high uncertainty across all states)
	assert.Empty(t, observationCtx.CurrentStateId, "Initial state should be unknown")

	// Create measurement source
	source := &pb.Source{
		Id:   "grip_strength_device_1",
		Type: pb.SourceType_SENSOR,
		Name: "Grip Strength Sensor",
	}

	// Add evidence from grip strength measurement
	measurementData := map[string]interface{}{
		"value": 95.0,
		"unit":  "kg",
		"timestamp": time.Now().Unix(),
	}
	
	content, err := json.Marshal(measurementData)
	assert.NoError(t, err, "Should marshal measurement data")

	resource := &pb.Resource{
		Id:       "measurement_1",
		Type:     pb.ResourceType_MEASUREMENT_DATA,
		SourceId: source.Id,
		Content:  string(content),
	}

	// Update belief with evidence
	updatedBelief, err := models.UpdateBeliefFromMeasurement(belief, resource, observationCtx)
	assert.NoError(t, err, "Should update belief without error")

	// Verify belief state transitions to "20s" based on measurement
	assert.Equal(t, "20s", observationCtx.CurrentStateId, "Should transition to 20s age range")
	assert.Contains(t, updatedBelief.Content[0].RawStr, "Male in 20s", "Updated belief should reference correct age range")
}

func TestEvidenceSourceValidation(t *testing.T) {
	// Set up a simple observation context
	observationCtx := &pb.ObservationContext{
		Id:   "test_context",
		Name: "Test Context",
		PossibleStates: []*pb.State{
			{
				Id:   "state_1",
				Name: "State 1",
				Properties: map[string]float32{
					"range_start": 0,
					"range_end":   50,
				},
			},
			{
				Id:   "state_2",
				Name: "State 2",
				Properties: map[string]float32{
					"range_start": 51,
					"range_end":   100,
				},
			},
		},
	}

	// Test cases for different evidence sources
	tests := []struct {
		name        string
		resource    *pb.Resource
		expectError bool
	}{
		{
			name: "Valid measurement",
			resource: &pb.Resource{
				Id:       "valid_measurement",
				Type:     pb.ResourceType_MEASUREMENT_DATA,
				SourceId: "sensor_1",
				Content:  `{"value": 25.0, "unit": "units", "timestamp": 1644644444}`,
			},
			expectError: false,
		},
		{
			name: "Missing measurement data",
			resource: &pb.Resource{
				Id:       "invalid_measurement",
				Type:     pb.ResourceType_MEASUREMENT_DATA,
				SourceId: "sensor_1",
			},
			expectError: true,
		},
		{
			name: "Invalid source type",
			resource: &pb.Resource{
				Id:       "invalid_source",
				Type:     pb.ResourceType_CHAT_LOG,
				SourceId: "chat_1",
				Content:  `{"value": 25.0, "unit": "units", "timestamp": 1644644444}`,
			},
			expectError: true,
		},
	}

	belief := &pb.Belief{
		Type: pb.BeliefType_FALSIFIABLE,
		Content: []*pb.Content{{
			RawStr: "Test belief",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := models.UpdateBeliefFromMeasurement(belief, tt.resource, observationCtx)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if err == nil {
					assert.Equal(t, "state_1", observationCtx.CurrentStateId)
				}
			}
		})
	}
}

func TestObservationContextStateTransitions(t *testing.T) {
	// Set up observation context with states
	observationCtx := &pb.ObservationContext{
		Id:   "test_transitions",
		Name: "Test Transitions",
		PossibleStates: []*pb.State{
			{
				Id:   "low",
				Name: "Low Range",
				Properties: map[string]float32{
					"range_start": 0,
					"range_end":   33,
				},
			},
			{
				Id:   "medium",
				Name: "Medium Range",
				Properties: map[string]float32{
					"range_start": 34,
					"range_end":   66,
				},
			},
			{
				Id:   "high",
				Name: "High Range",
				Properties: map[string]float32{
					"range_start": 67,
					"range_end":   100,
				},
			},
		},
	}

	belief := &pb.Belief{
		Type: pb.BeliefType_FALSIFIABLE,
		Content: []*pb.Content{{
			RawStr: "Test state transitions",
		}},
	}

	// Test state transitions with different measurements
	measurements := []struct {
		value         float64
		expectedState string
		stateName     string
	}{
		{20.0, "low", "Low Range"},
		{50.0, "medium", "Medium Range"},
		{80.0, "high", "High Range"},
		{35.0, "medium", "Medium Range"}, // Test transition back to medium
	}

	for _, m := range measurements {
		measurementData := map[string]interface{}{
			"value": m.value,
			"unit":  "units",
			"timestamp": time.Now().Unix(),
		}
		
		content, err := json.Marshal(measurementData)
		assert.NoError(t, err, "Should marshal measurement data")

		resource := &pb.Resource{
			Id:       "measurement",
			Type:     pb.ResourceType_MEASUREMENT_DATA,
			SourceId: "sensor_1",
			Content:  string(content),
		}

		updatedBelief, err := models.UpdateBeliefFromMeasurement(belief, resource, observationCtx)
		assert.NoError(t, err)
		assert.Equal(t, m.expectedState, observationCtx.CurrentStateId)
		assert.Contains(t, updatedBelief.Content[0].RawStr, m.stateName)
	}
}
