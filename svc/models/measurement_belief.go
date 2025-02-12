package models

import (
	"encoding/json"
	"fmt"

	pb "epistemic-me-core/pb/models"
)

type MeasurementData struct {
	Value     float64 `json:"value"`
	Unit      string  `json:"unit"`
	Timestamp int64   `json:"timestamp"`
}

// UpdateBeliefFromMeasurement updates a belief based on measurement data from a resource.
// It evaluates the measurement value against possible states in the observation context
// and updates both the belief content and the observation context's current state.
//
// Parameters:
//   - belief: The belief to update
//   - resource: The resource containing measurement data (must be of type MEASUREMENT_DATA)
//   - observationCtx: The observation context containing possible states and their ranges
//
// Returns:
//   - Updated belief with new content reflecting the measurement
//   - Error if validation fails or measurement doesn't match any state ranges
func UpdateBeliefFromMeasurement(belief *pb.Belief, resource *pb.Resource, observationCtx *pb.ObservationContext) (*pb.Belief, error) {
	// Validate inputs
	if belief == nil || resource == nil || observationCtx == nil {
		return nil, fmt.Errorf("invalid input: belief, resource, and observationCtx must not be nil")
	}

	// Verify resource type and data
	if resource.Type != pb.ResourceType_MEASUREMENT_DATA {
		return nil, fmt.Errorf("invalid resource type: expected MEASUREMENT_DATA, got %v", resource.Type)
	}

	if resource.Content == "" {
		return nil, fmt.Errorf("invalid resource: missing measurement data")
	}

	// Parse measurement data from content
	var measurement MeasurementData
	if err := json.Unmarshal([]byte(resource.Content), &measurement); err != nil {
		return nil, fmt.Errorf("failed to parse measurement data: %v", err)
	}

	// Find the appropriate state based on measurement value
	var newStateId string
	for _, state := range observationCtx.PossibleStates {
		rangeStart := state.Properties["range_start"]
		rangeEnd := state.Properties["range_end"]

		if float32(measurement.Value) >= rangeStart && float32(measurement.Value) <= rangeEnd {
			newStateId = state.Id
			break
		}
	}

	if newStateId == "" {
		return nil, fmt.Errorf("measurement value %v does not fall within any defined state ranges", measurement.Value)
	}

	// Update observation context state
	observationCtx.CurrentStateId = newStateId
	observationCtx.LastUpdated = measurement.Timestamp

	// Find the state name for the belief content
	var stateName string
	for _, state := range observationCtx.PossibleStates {
		if state.Id == newStateId {
			stateName = state.Name
			break
		}
	}

	// Update belief content
	updatedBelief := &pb.Belief{
		Id:          belief.Id,
		SelfModelId: belief.SelfModelId,
		Version:     belief.Version,
		Type:        belief.Type,
		Content: []*pb.Content{{
			RawStr: fmt.Sprintf("User has Grip Strength biological age of %s based on measurement", stateName),
		}},
	}

	return updatedBelief, nil
}
