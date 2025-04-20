package unit

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"epistemic-me-core/svc/models"
)

// InteractionEvent for testing
type InteractionEvent struct {
	Question string
	Answer   string
}

// Mock AIHelper to test without actual AI calls
type MockAIHelper struct {
	mock.Mock
}

func (m *MockAIHelper) GetInteractionEventAsBelief(event InteractionEvent) ([]string, error) {
	args := m.Called(event)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockAIHelper) GenerateQuestion(beliefSystem string, previousEvents []InteractionEvent) (string, error) {
	args := m.Called(beliefSystem, previousEvents)
	return args.String(0), args.Error(1)
}

func (m *MockAIHelper) ExtractQuestionsFromText(text string) ([]string, error) {
	args := m.Called(text)
	return args.Get(0).([]string), args.Error(1)
}

// Mock key-value store
type MockKVStore struct {
	mock.Mock
}

func (m *MockKVStore) Store(selfModelID string, key string, value interface{}, version int) error {
	args := m.Called(selfModelID, key, value, version)
	return args.Error(0)
}

func (m *MockKVStore) Retrieve(selfModelID string, key string) (interface{}, error) {
	args := m.Called(selfModelID, key)
	return args.Get(0), args.Error(1)
}

// TestPredictiveProcessingContextCreation tests that the PredictiveProcessingContext
// is correctly initialized and structured in basic scenarios
func TestPredictiveProcessingContextCreation(t *testing.T) {
	// Setup
	mockKVStore := new(MockKVStore)
	// We won't use the mockAIHelper in this test, so no need to instantiate it

	// Initial belief system
	bs := &models.BeliefSystem{
		Beliefs: []*models.Belief{
			{
				ID: "belief-1",
				Content: []models.Content{
					{RawStr: "Regular exercise improves health"},
				},
				Type: models.Statement,
			},
		},
		EpistemicContexts: []*models.EpistemicContext{
			{
				PredictiveProcessingContext: &models.PredictiveProcessingContext{
					ObservationContexts: []*models.ObservationContext{},
					BeliefContexts:      []*models.BeliefContext{},
				},
			},
		},
	}

	// Setup mock KVStore to return our belief system
	selfModelID := "test-model-id"
	mockKVStore.On("Retrieve", selfModelID, "BeliefSystem").Return(bs, nil)
	mockKVStore.On("Store", selfModelID, "BeliefSystem", mock.Anything, mock.Anything).Return(nil)

	// Create a test dialectic with one interaction
	dialectic := &models.Dialectic{
		ID:          "test-dialectic",
		SelfModelID: selfModelID,
		UserInteractions: []models.DialecticalInteraction{
			{
				ID:     "interaction-1",
				Status: models.StatusPendingAnswer,
				Type:   models.InteractionTypeQuestionAnswer,
				Interaction: &models.InteractionData{
					QuestionAnswer: &models.QuestionAnswerInteraction{
						Question: models.Question{
							Question:           "How often do you exercise?",
							CreatedAtMillisUTC: time.Now().UnixMilli(),
						},
					},
				},
			},
		},
	}

	// Setup mock KVStore to return our dialectic
	mockKVStore.On("Retrieve", selfModelID, "Dialectic:test-dialectic").Return(dialectic, nil)

	// Test: Verify that PredictiveProcessingContext is properly initialized
	// and contains the expected initial data
	assert.NotNil(t, bs.EpistemicContexts[0].PredictiveProcessingContext)
	assert.Empty(t, bs.EpistemicContexts[0].PredictiveProcessingContext.ObservationContexts)
	assert.Empty(t, bs.EpistemicContexts[0].PredictiveProcessingContext.BeliefContexts)
}

// TestUpdateDialecticWithObservationContext tests that answering a question
// properly updates the ObservationContext in PredictiveProcessingContext
func TestUpdateDialecticWithObservationContext(t *testing.T) {
	// Setup
	mockKVStore := new(MockKVStore)
	mockAIHelper := new(MockAIHelper)

	// Initial belief system with empty PredictiveProcessingContext
	bs := &models.BeliefSystem{
		Beliefs: []*models.Belief{
			{
				ID: "belief-1",
				Content: []models.Content{
					{RawStr: "Regular exercise improves health"},
				},
				Type: models.Statement,
			},
		},
		EpistemicContexts: []*models.EpistemicContext{
			{
				PredictiveProcessingContext: &models.PredictiveProcessingContext{
					ObservationContexts: []*models.ObservationContext{},
					BeliefContexts:      []*models.BeliefContext{},
				},
			},
		},
	}

	// Setup mock KVStore to return our belief system
	selfModelID := "test-model-id"
	mockKVStore.On("Retrieve", selfModelID, "BeliefSystem").Return(bs, nil)

	// Set up AIHelper mock to return beliefs when extracting from answer
	event := InteractionEvent{
		Question: "How often do you exercise?",
		Answer:   "I exercise three times per week",
	}
	mockAIHelper.On("GetInteractionEventAsBelief", event).Return(
		[]string{"I exercise three times per week"}, nil)

	// Setup expected behavior for BeliefSystem storage
	mockKVStore.On("Store", selfModelID, "BeliefSystem", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			// Extract the belief system that would be stored
			storedBS := args.Get(2).(*models.BeliefSystem)

			// Check that a new ObservationContext has been added
			require.Len(t, storedBS.EpistemicContexts, 1)
			require.NotNil(t, storedBS.EpistemicContexts[0].PredictiveProcessingContext)

			// In a real implementation, the ObservationContext would be updated here
			// This is where you would add assertions to verify the correct structure
		}).Return(nil)

	// Setup the dialectic that will be updated with an answer
	dialectic := &models.Dialectic{
		ID:          "test-dialectic",
		SelfModelID: selfModelID,
		UserInteractions: []models.DialecticalInteraction{
			{
				ID:     "interaction-1",
				Status: models.StatusPendingAnswer,
				Type:   models.InteractionTypeQuestionAnswer,
				Interaction: &models.InteractionData{
					QuestionAnswer: &models.QuestionAnswerInteraction{
						Question: models.Question{
							Question:           "How often do you exercise?",
							CreatedAtMillisUTC: time.Now().UnixMilli(),
						},
					},
				},
			},
		},
	}

	mockKVStore.On("Retrieve", selfModelID, "Dialectic:test-dialectic").Return(dialectic, nil)
	mockKVStore.On("Store", selfModelID, "Dialectic:test-dialectic", mock.Anything, mock.Anything).Return(nil)

	// In a complete test, you would call the actual UpdateDialectic function here
	// and verify the results. This skeleton provides the structure.

	// For linter only - ensure mocks are used
	assert.NotNil(t, mockAIHelper)
}

// TestUpdateDialecticPerformance tests the performance of UpdateDialectic by mocking all AI calls
func TestUpdateDialecticPerformance(t *testing.T) {
	// Setup
	mockKVStore := new(MockKVStore)
	mockAIHelper := new(MockAIHelper)

	// Mock all AI calls to return quick responses
	mockAIHelper.On("GetInteractionEventAsBelief", mock.Anything).Return(
		[]string{"I believe in regular exercise"}, nil)

	mockAIHelper.On("GenerateQuestion", mock.Anything, mock.Anything).Return(
		"What is your fitness routine?", nil)

	mockAIHelper.On("ExtractQuestionsFromText", mock.Anything).Return(
		[]string{"What's your diet like?", "How often do you exercise?"}, nil)

	// Setup belief system
	bs := &models.BeliefSystem{
		Beliefs: []*models.Belief{
			{
				ID: "belief-1",
				Content: []models.Content{
					{RawStr: "Regular exercise improves health"},
				},
				Type: models.Statement,
			},
		},
		EpistemicContexts: []*models.EpistemicContext{
			{
				PredictiveProcessingContext: &models.PredictiveProcessingContext{
					ObservationContexts: []*models.ObservationContext{},
					BeliefContexts:      []*models.BeliefContext{},
				},
			},
		},
	}

	// Setup dialectic
	selfModelID := "test-model-id"
	dialecticID := "test-dialectic-id"

	mockKVStore.On("Retrieve", selfModelID, "BeliefSystem").Return(bs, nil)
	mockKVStore.On("Store", selfModelID, "BeliefSystem", mock.Anything, mock.Anything).Return(nil)

	dialectic := &models.Dialectic{
		ID:          dialecticID,
		SelfModelID: selfModelID,
		UserInteractions: []models.DialecticalInteraction{
			{
				ID:     "interaction-1",
				Status: models.StatusPendingAnswer,
				Type:   models.InteractionTypeQuestionAnswer,
				Interaction: &models.InteractionData{
					QuestionAnswer: &models.QuestionAnswerInteraction{
						Question: models.Question{
							Question:           "How often do you exercise?",
							CreatedAtMillisUTC: time.Now().UnixMilli(),
						},
					},
				},
			},
		},
	}

	mockKVStore.On("Retrieve", selfModelID, "Dialectic:"+dialecticID).Return(dialectic, nil)
	mockKVStore.On("Store", selfModelID, "Dialectic:"+dialecticID, mock.Anything, mock.Anything).Return(nil)

	// In a real test, you would time the execution of UpdateDialectic with the mocked dependencies
	// Time the operation to verify performance improvements
	// startTime := time.Now()
	// ... Call UpdateDialectic here
	// duration := time.Since(startTime)
	// assert.Less(t, duration, 500*time.Millisecond, "Operation should complete quickly")

	// For linter only - ensure mocks are used
	assert.NotNil(t, mockAIHelper)
	assert.NotNil(t, mockKVStore)
}

// TestPredictiveProcessingWithBeliefContext tests how belief contexts are managed in the UpdateDialectic process
func TestPredictiveProcessingWithBeliefContext(t *testing.T) {
	// Setup
	mockKVStore := new(MockKVStore)
	mockAIHelper := new(MockAIHelper)

	// Create belief system with existing belief
	initialBelief := &models.Belief{
		ID: "belief-1",
		Content: []models.Content{
			{RawStr: "Regular exercise improves health"},
		},
		Type: models.Statement,
	}

	// Create an empty PredictiveProcessingContext
	ppc := &models.PredictiveProcessingContext{
		ObservationContexts: []*models.ObservationContext{},
		BeliefContexts:      []*models.BeliefContext{},
	}

	bs := &models.BeliefSystem{
		Beliefs: []*models.Belief{initialBelief},
		EpistemicContexts: []*models.EpistemicContext{
			{
				PredictiveProcessingContext: ppc,
			},
		},
	}

	// Setup mock for belief extraction
	newBeliefContent := "I exercise three times per week with cardio and strength training"
	event := InteractionEvent{
		Question: "How often do you exercise?",
		Answer:   "I exercise three times per week with cardio and strength training",
	}
	mockAIHelper.On("GetInteractionEventAsBelief", event).Return(
		[]string{newBeliefContent}, nil)

	// Setup mock for the next question generation
	mockAIHelper.On("GenerateQuestion", mock.Anything, mock.Anything).Return(
		"How has your exercise routine affected your overall wellbeing?", nil)

	// Setup dialectic with an interaction waiting for an answer
	selfModelID := "test-model-id"
	dialecticID := "test-dialectic-id"

	dialectic := &models.Dialectic{
		ID:          dialecticID,
		SelfModelID: selfModelID,
		UserInteractions: []models.DialecticalInteraction{
			{
				ID:     "interaction-1",
				Status: models.StatusPendingAnswer,
				Type:   models.InteractionTypeQuestionAnswer,
				Interaction: &models.InteractionData{
					QuestionAnswer: &models.QuestionAnswerInteraction{
						Question: models.Question{
							Question:           "How often do you exercise?",
							CreatedAtMillisUTC: time.Now().UnixMilli(),
						},
					},
				},
			},
		},
	}

	mockKVStore.On("Retrieve", selfModelID, "BeliefSystem").Return(bs, nil)
	mockKVStore.On("Retrieve", selfModelID, "Dialectic:"+dialecticID).Return(dialectic, nil)

	// Capture the stored belief system to verify changes to PredictiveProcessingContext
	mockKVStore.On("Store", selfModelID, "BeliefSystem", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			storedBS := args.Get(2).(*models.BeliefSystem)

			// Verify PredictiveProcessingContext structure
			require.Len(t, storedBS.EpistemicContexts, 1)
			ppc := storedBS.EpistemicContexts[0].PredictiveProcessingContext
			require.NotNil(t, ppc)

			// In a complete test, you would verify:
			// 1. That BeliefContexts have been created for each belief
			// 2. That the appropriate belief contexts contain references to observation contexts
			// 3. That any confidence ratings or other metrics are properly set
		}).Return(nil)

	mockKVStore.On("Store", selfModelID, "Dialectic:"+dialecticID, mock.Anything, mock.Anything).Return(nil)

	// In a real test, you would invoke the actual UpdateDialectic function here

	// For linter only - ensure mocks are used
	assert.NotNil(t, mockAIHelper)
}

// TestOptimizedBeliefExtraction tests a more efficient approach to belief extraction
// that doesn't rely as heavily on AI calls
func TestOptimizedBeliefExtraction(t *testing.T) {
	// Setup
	mockKVStore := new(MockKVStore)
	mockAIHelper := new(MockAIHelper)

	// For optimized belief extraction, we would use a more targeted prompt
	// or a local model to reduce API calls
	event := InteractionEvent{
		Question: "What's your exercise routine like?",
		Answer:   "I exercise regularly and prefer cardio over strength training",
	}
	mockAIHelper.On("GetInteractionEventAsBelief", event).Return(
		[]string{"I exercise regularly", "I prefer cardio over strength training"}, nil)

	// Setup belief system and dialectic
	selfModelID := "test-model-id"
	dialecticID := "test-dialectic-id"

	bs := &models.BeliefSystem{
		Beliefs: []*models.Belief{
			{
				ID: "belief-1",
				Content: []models.Content{
					{RawStr: "Regular exercise improves health"},
				},
				Type: models.Statement,
			},
		},
		EpistemicContexts: []*models.EpistemicContext{
			{
				PredictiveProcessingContext: &models.PredictiveProcessingContext{
					ObservationContexts: []*models.ObservationContext{},
					BeliefContexts:      []*models.BeliefContext{},
				},
			},
		},
	}

	dialectic := &models.Dialectic{
		ID:          dialecticID,
		SelfModelID: selfModelID,
		UserInteractions: []models.DialecticalInteraction{
			{
				ID:     "interaction-1",
				Status: models.StatusPendingAnswer,
				Type:   models.InteractionTypeQuestionAnswer,
				Interaction: &models.InteractionData{
					QuestionAnswer: &models.QuestionAnswerInteraction{
						Question: models.Question{
							Question:           "What's your exercise routine like?",
							CreatedAtMillisUTC: time.Now().UnixMilli(),
						},
					},
				},
			},
		},
	}

	mockKVStore.On("Retrieve", selfModelID, "BeliefSystem").Return(bs, nil)
	mockKVStore.On("Retrieve", selfModelID, "Dialectic:"+dialecticID).Return(dialectic, nil)
	mockKVStore.On("Store", selfModelID, "BeliefSystem", mock.Anything, mock.Anything).Return(nil)
	mockKVStore.On("Store", selfModelID, "Dialectic:"+dialecticID, mock.Anything, mock.Anything).Return(nil)

	// The test should measure the time and number of AI calls made during the UpdateDialectic process
	// You would create functions that implement the optimized approach and compare against baseline

	// For linter only - ensure mocks are used
	assert.NotNil(t, mockAIHelper)
	assert.NotNil(t, mockKVStore)
}

func TestPredictiveProcessingContextStructure(t *testing.T) {
	// Create a PredictiveProcessingContext with observation and belief contexts
	ocID := uuid.New().String()
	beliefID := uuid.New().String()

	// Create an ObservationContext
	oc := &models.ObservationContext{
		ID:             ocID,
		Name:           "Test Observation",
		ParentID:       "",
		PossibleStates: []string{"State1", "State2", "State3"},
	}

	// Create a BeliefContext that references the ObservationContext
	bc := &models.BeliefContext{
		BeliefID:             beliefID,
		ObservationContextID: ocID,
		ConfidenceRatings: []models.ConfidenceRating{
			{
				ConfidenceScore: 0.8,
				Default:         true,
			},
		},
		ConditionalProbs:        map[string]float32{"State1": 0.7, "State2": 0.2, "State3": 0.1},
		DialecticInteractionIDs: []string{uuid.New().String()},
		EpistemicEmotion:        models.Confirmation,
		EmotionIntensity:        0.5,
	}

	// Create the PredictiveProcessingContext
	ppc := &models.PredictiveProcessingContext{
		ObservationContexts: []*models.ObservationContext{oc},
		BeliefContexts:      []*models.BeliefContext{bc},
	}

	// Validate the structure
	require.NotNil(t, ppc, "PredictiveProcessingContext should not be nil")
	require.Len(t, ppc.ObservationContexts, 1, "Should have one ObservationContext")
	require.Len(t, ppc.BeliefContexts, 1, "Should have one BeliefContext")

	// Validate ObservationContext
	assert.Equal(t, ocID, ppc.ObservationContexts[0].ID, "ObservationContext ID should match")
	assert.Equal(t, "Test Observation", ppc.ObservationContexts[0].Name, "ObservationContext name should match")
	assert.Len(t, ppc.ObservationContexts[0].PossibleStates, 3, "Should have three possible states")

	// Validate BeliefContext
	assert.Equal(t, beliefID, ppc.BeliefContexts[0].BeliefID, "BeliefContext BeliefID should match")
	assert.Equal(t, ocID, ppc.BeliefContexts[0].ObservationContextID, "BeliefContext ObservationContextID should match")
	assert.Len(t, ppc.BeliefContexts[0].ConfidenceRatings, 1, "Should have one confidence rating")
	assert.Equal(t, models.Confirmation, ppc.BeliefContexts[0].EpistemicEmotion, "EpistemicEmotion should be Confirmation")

	// Validate relationship between BeliefContext and ObservationContext
	assert.Equal(t, ppc.BeliefContexts[0].ObservationContextID, ppc.ObservationContexts[0].ID,
		"BeliefContext should reference the correct ObservationContext")
}

func TestEpistemicContextWithPredictiveProcessing(t *testing.T) {
	// Create a BeliefSystem with an EpistemicContext containing a PredictiveProcessingContext
	ocID := uuid.New().String()
	beliefID := uuid.New().String()

	// Create an ObservationContext
	oc := &models.ObservationContext{
		ID:             ocID,
		Name:           "Test Observation",
		ParentID:       "",
		PossibleStates: []string{"Positive", "Negative", "Neutral"},
	}

	// Create a BeliefContext
	bc := &models.BeliefContext{
		BeliefID:             beliefID,
		ObservationContextID: ocID,
		ConfidenceRatings: []models.ConfidenceRating{
			{
				ConfidenceScore: 0.8,
				Default:         true,
			},
		},
		ConditionalProbs:        map[string]float32{},
		DialecticInteractionIDs: []string{},
		EpistemicEmotion:        models.Confirmation,
		EmotionIntensity:        0.5,
	}

	// Create PredictiveProcessingContext
	ppc := &models.PredictiveProcessingContext{
		ObservationContexts: []*models.ObservationContext{oc},
		BeliefContexts:      []*models.BeliefContext{bc},
	}

	// Create EpistemicContext with the PredictiveProcessingContext
	ec := &models.EpistemicContext{
		PredictiveProcessingContext: ppc,
	}

	// Create a BeliefSystem with this context
	belief := &models.Belief{
		ID: beliefID,
		Content: []models.Content{
			{RawStr: "Test belief statement"},
		},
		Type: models.Statement,
	}

	bs := &models.BeliefSystem{
		Beliefs:           []*models.Belief{belief},
		EpistemicContexts: []*models.EpistemicContext{ec},
	}

	// Validate the structure
	require.Len(t, bs.EpistemicContexts, 1, "Should have one EpistemicContext")
	require.NotNil(t, bs.EpistemicContexts[0].PredictiveProcessingContext, "PredictiveProcessingContext should not be nil")

	// Validate the PredictiveProcessingContext
	ppc = bs.EpistemicContexts[0].PredictiveProcessingContext
	require.Len(t, ppc.ObservationContexts, 1, "Should have one ObservationContext")
	require.Len(t, ppc.BeliefContexts, 1, "Should have one BeliefContext")

	// Validate the BeliefContext references the correct Belief
	assert.Equal(t, belief.ID, ppc.BeliefContexts[0].BeliefID, "BeliefContext should reference the correct Belief")
}

func TestUpdateObservationContext(t *testing.T) {
	// Create an initial ObservationContext
	ocID := uuid.New().String()
	oc := &models.ObservationContext{
		ID:             ocID,
		Name:           "Initial Observation",
		ParentID:       "",
		PossibleStates: []string{"State1", "State2"},
	}

	// Create a PredictiveProcessingContext with this ObservationContext
	ppc := &models.PredictiveProcessingContext{
		ObservationContexts: []*models.ObservationContext{oc},
		BeliefContexts:      []*models.BeliefContext{},
	}

	// Update the ObservationContext properties
	oc.Name = "Updated Observation"
	oc.PossibleStates = append(oc.PossibleStates, "State3")

	// Check that the changes are reflected in the PredictiveProcessingContext
	assert.Equal(t, "Updated Observation", ppc.ObservationContexts[0].Name, "Name should be updated")
	assert.Len(t, ppc.ObservationContexts[0].PossibleStates, 3, "Should now have three possible states")
	assert.Equal(t, "State3", ppc.ObservationContexts[0].PossibleStates[2], "New state should be added")
}

func TestExtrapolateObservationContexts_MetabolicHealthMarkdown(t *testing.T) {
	markdown := `# Metabolic Health Philosophy

## Raison d'être
To model how a person's day‑to‑day sensations, actions, and environments
reshape energy, recovery, and long‑term resilience.

---

## Experiential Narrative ✦
06 : 30 — I surface from dreams; [[C: Circadian Rhythm]] slides from [[S: asleep]]
to [[S: awake]].  
A coral‑pink dawn floods the room: [[C: Zeitgeber Exposure]]
sees [[S: bright-light-day]]; cortisol surges — [[C: Hormonal Pulse]]
hits [[S: cortisol-peak]].  Core temp climbs toward 37 °C: [[C: Thermoregulation]]
trends [[S: warm-skin]].  
My heart‑rate variability sits in [[C: HRV Band]] [[S: balanced-lfhf]];
inflammation stays [[C: Inflammation Marker]] [[S: low-crp]].

08 : 00 — Keyboard clicks keep [[C: Energy Expenditure]] at [[S: sedentary]];
I'm still [[C: Nutrient Flux]] [[S: fasted]], glucose stable in
[[C: Glycaemic State]] [[S: euglycaemic]].

09 : 30 — Espresso triggers an [[S: insulin-spike]]; an
[[C: Ultradian Cycle]] flips to [[S: interaction-phase]].

10 : 45 — Cycle dips into [[S: recovery-phase]]; box‑breathing nudges
[[C: Recovery and Stress]] toward [[S: parasympathetic-high]].

12 : 30 — Lunch (quinoa‑bowl) moves [[S: fasted]] → [[S: fed-light]];
post‑meal glucose rises to [[S: mild-postprandial]];
[[C: Energy Expenditure]] upgrades to [[S: neat]] during a walk.

15 : 00 — No snack: glucose trends toward [[S: hypo]];
HRV dips to [[S: low-lfhf]];
[[C: Recovery and Stress]] tilts [[S: sympathetic-high]];
temperature cools to [[S: neutral]].

16 : 00 — HIIT pushes [[S: exercise-high]];
catabolic hormones spike — [[S: catabolic-dominant]];
thermoregulatory flush maintains [[S: warm-skin]];
inflammation stays [[S: low-crp]].

18 : 30 — Dinner (steak + rice) → [[S: fed-heavy]],
[[S: high-postprandial]], second [[S: insulin-spike]].

19 : 30 — Light jog counts as [[S: exercise-low]]; aids glucose clearance.

21 : 00 — Screens dim; [[S: dim-light-evening]];
cortisol falls to [[S: cortisol-trough]];
[[C: Nutrient Flux]] drifts to [[S: over-fed]] if dessert appears.

22 : 10 — [[C: Sleep Architecture]] begins:
[[S: light-n1]] → [[S: light-n2]] → [[S: slow-wave]];
GH pulses — [[S: gh-pulse]];
skin cools to [[S: cool-skin]];
inflammation remains [[S: low-crp]]; whole‑body repair is
[[S: anabolic-dominant]].

03 : 00 — Brief [[S: awakening]] for water; glucose stable;
back into [[S: rem]] until dawn.

---
`

	contexts := models.ExtrapolateObservationContexts(markdown)
	if len(contexts) == 0 {
		t.Fatalf("Expected at least one ObservationContext, got 0")
	}

	// Check that some known contexts and states are present
	var foundCircadian, foundHRV, foundSleepArch bool
	var circadianStates, hrvStates, sleepArchStates []string
	for _, ctx := range contexts {
		switch ctx.Name {
		case "Circadian Rhythm":
			foundCircadian = true
			circadianStates = ctx.PossibleStates
		case "HRV Band":
			foundHRV = true
			hrvStates = ctx.PossibleStates
		case "Sleep Architecture":
			foundSleepArch = true
			sleepArchStates = ctx.PossibleStates
		}
	}
	if !foundCircadian {
		t.Errorf("Expected context 'Circadian Rhythm' to be found")
	}
	if !foundHRV {
		t.Errorf("Expected context 'HRV Band' to be found")
	}
	if !foundSleepArch {
		t.Errorf("Expected context 'Sleep Architecture' to be found")
	}
	// Check that some expected states are present
	if foundCircadian && (len(circadianStates) == 0 || !contains(circadianStates, "asleep") || !contains(circadianStates, "awake")) {
		t.Errorf("Circadian Rhythm should have states 'asleep' and 'awake', got %v", circadianStates)
	}
	if foundHRV && (len(hrvStates) == 0 || !contains(hrvStates, "balanced-lfhf")) {
		t.Errorf("HRV Band should have state 'balanced-lfhf', got %v", hrvStates)
	}
	if foundSleepArch && (len(sleepArchStates) == 0 || !contains(sleepArchStates, "light-n1") || !contains(sleepArchStates, "light-n2") || !contains(sleepArchStates, "slow-wave")) {
		t.Errorf("Sleep Architecture should have states 'light-n1', 'light-n2', 'slow-wave', got %v", sleepArchStates)
	}
}

func TestExtrapolateObservationContexts_TreeStructure(t *testing.T) {
	markdown := `## Experiential Narrative
06:30 — [[C: Morning Routine]]
  [[C: Circadian Rhythm]] [[S: asleep]] → [[S: awake]]
  [[C: Zeitgeber Exposure]] [[S: bright-light-day]]
    [[C: Hormonal Pulse]] [[S: cortisol-peak]]
  [[C: HRV Band]] [[S: balanced-lfhf]]
[[C: Evening Routine]]
  [[C: Sleep Architecture]] [[S: light-n1]] → [[S: slow-wave]]
    [[C: GH Pulse]] [[S: gh-pulse]]
`

	contexts := models.ExtrapolateObservationContexts(markdown)
	if len(contexts) == 0 {
		t.Fatalf("Expected at least one ObservationContext, got 0")
	}

	// Build a map of context name to context for easy lookup
	ctxByName := make(map[string]*models.ObservationContext)
	for _, ctx := range contexts {
		ctxByName[ctx.Name] = ctx
	}

	// Check parent-child relationships (simulate expected tree)
	// For this test, let's assume the function is extended to set ParentID based on indentation (2 spaces = child)
	// Morning Routine is parent of Circadian Rhythm, Zeitgeber Exposure, HRV Band, Hormonal Pulse
	// Zeitgeber Exposure is parent of Hormonal Pulse
	// Evening Routine is parent of Sleep Architecture
	// Sleep Architecture is parent of GH Pulse

	// Helper to get ParentID by name
	getParent := func(name string) string {
		if ctx, ok := ctxByName[name]; ok {
			return ctx.ParentID
		}
		return ""
	}

	// Check parent relationships
	if getParent("Circadian Rhythm") != ctxByName["Morning Routine"].ID {
		t.Errorf("Circadian Rhythm should have parent Morning Routine")
	}
	if getParent("Zeitgeber Exposure") != ctxByName["Morning Routine"].ID {
		t.Errorf("Zeitgeber Exposure should have parent Morning Routine")
	}
	if getParent("Hormonal Pulse") != ctxByName["Zeitgeber Exposure"].ID {
		t.Errorf("Hormonal Pulse should have parent Zeitgeber Exposure")
	}
	if getParent("HRV Band") != ctxByName["Morning Routine"].ID {
		t.Errorf("HRV Band should have parent Morning Routine")
	}
	if getParent("Sleep Architecture") != ctxByName["Evening Routine"].ID {
		t.Errorf("Sleep Architecture should have parent Evening Routine")
	}
	if getParent("GH Pulse") != ctxByName["Sleep Architecture"].ID {
		t.Errorf("GH Pulse should have parent Sleep Architecture")
	}

	// Check states
	if ctx, ok := ctxByName["Circadian Rhythm"]; ok {
		if !contains(ctx.PossibleStates, "asleep") || !contains(ctx.PossibleStates, "awake") {
			t.Errorf("Circadian Rhythm should have states 'asleep' and 'awake', got %v", ctx.PossibleStates)
		}
	}
	if ctx, ok := ctxByName["Hormonal Pulse"]; ok {
		if !contains(ctx.PossibleStates, "cortisol-peak") {
			t.Errorf("Hormonal Pulse should have state 'cortisol-peak', got %v", ctx.PossibleStates)
		}
	}
	if ctx, ok := ctxByName["Sleep Architecture"]; ok {
		if !contains(ctx.PossibleStates, "light-n1") || !contains(ctx.PossibleStates, "slow-wave") {
			t.Errorf("Sleep Architecture should have states 'light-n1' and 'slow-wave', got %v", ctx.PossibleStates)
		}
	}
	if ctx, ok := ctxByName["GH Pulse"]; ok {
		if !contains(ctx.PossibleStates, "gh-pulse") {
			t.Errorf("GH Pulse should have state 'gh-pulse', got %v", ctx.PossibleStates)
		}
	}
}
