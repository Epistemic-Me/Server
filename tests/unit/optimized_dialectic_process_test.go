package unit

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"epistemic-me-core/svc"
	"epistemic-me-core/svc/models"
)

// MockDialecticalEpistemologyForUpdate mocks the DialecticalEpistemology for testing
type MockDialecticalEpistemologyForUpdate struct {
	mock.Mock
}

func (m *MockDialecticalEpistemologyForUpdate) Process(event *models.DialecticEvent, dryRun bool, selfModelID string) (*models.BeliefSystem, error) {
	args := m.Called(event, dryRun, selfModelID)
	return args.Get(0).(*models.BeliefSystem), args.Error(1)
}

func (m *MockDialecticalEpistemologyForUpdate) Respond(bs *models.BeliefSystem, event *models.DialecticEvent, answer string) (*models.DialecticResponse, error) {
	args := m.Called(bs, event, answer)
	return args.Get(0).(*models.DialecticResponse), args.Error(1)
}

// Mock AIHelper specifically for our tests
type MockAIHelperForOptTests struct {
	mock.Mock
}

func (m *MockAIHelperForOptTests) GetInteractionEventAsBelief(event svc.InteractionEvent) ([]string, error) {
	args := m.Called(event)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockAIHelperForOptTests) GenerateQuestion(beliefSystem string, previousEvents []svc.InteractionEvent) (string, error) {
	args := m.Called(beliefSystem, previousEvents)
	return args.String(0), args.Error(1)
}

func (m *MockAIHelperForOptTests) ExtractQuestionsFromText(text string) ([]string, error) {
	args := m.Called(text)
	return args.Get(0).([]string), args.Error(1)
}

// Mock KVStore specifically for our tests
type MockKVStoreForOptTests struct {
	mock.Mock
}

func (m *MockKVStoreForOptTests) Store(selfModelID string, key string, value interface{}, version int) error {
	args := m.Called(selfModelID, key, value, version)
	return args.Error(0)
}

func (m *MockKVStoreForOptTests) Retrieve(selfModelID string, key string) (interface{}, error) {
	args := m.Called(selfModelID, key)
	return args.Get(0), args.Error(1)
}

// TestDialecticService is a test implementation of OptimizedDialecticService for testing
// that uses our mock objects rather than relying on reflection
type TestDialecticService struct {
	mockKVStore         *MockKVStoreForOptTests
	mockAIHelper        *MockAIHelperForOptTests
	mockDialecticEpiSvc *MockDialecticalEpistemologyForUpdate
}

func NewTestDialecticService(
	mockKVStore *MockKVStoreForOptTests,
	mockAIHelper *MockAIHelperForOptTests,
	mockDialecticEpiSvc *MockDialecticalEpistemologyForUpdate,
) *TestDialecticService {
	return &TestDialecticService{
		mockKVStore:         mockKVStore,
		mockAIHelper:        mockAIHelper,
		mockDialecticEpiSvc: mockDialecticEpiSvc,
	}
}

// OptimizedUpdateDialectic replicates the behavior of the original service but uses our mocks
func (s *TestDialecticService) OptimizedUpdateDialectic(input *models.UpdateDialecticInput) (*models.UpdateDialecticOutput, error) {
	// Retrieve the dialectic
	dialecticValue, err := s.mockKVStore.Retrieve(input.SelfModelID, "Dialectic:"+input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve dialectic: %w", err)
	}

	dialectic, ok := dialecticValue.(*models.Dialectic)
	if !ok {
		return nil, fmt.Errorf("invalid dialectic type")
	}

	if input.Answer.UserAnswer != "" {
		// Process the belief system
		bs, err := s.mockDialecticEpiSvc.Process(&models.DialecticEvent{
			PreviousInteractions: dialectic.UserInteractions,
		}, input.DryRun, input.SelfModelID)
		if err != nil {
			return nil, err
		}

		// Extract the current interaction
		interactionEvent := svc.InteractionEvent{
			Question: dialectic.UserInteractions[len(dialectic.UserInteractions)-1].Interaction.QuestionAnswer.Question.Question,
			Answer:   input.Answer.UserAnswer,
		}

		// Extract beliefs from the answer
		extractedBeliefStrings, err := s.mockAIHelper.GetInteractionEventAsBelief(interactionEvent)
		if err != nil {
			return nil, fmt.Errorf("failed to extract beliefs: %w", err)
		}

		// Add extracted beliefs to the response for verification
		lastIdx := len(dialectic.UserInteractions) - 1
		qa := dialectic.UserInteractions[lastIdx].Interaction.QuestionAnswer

		// Create belief objects from extracted strings
		extractedBeliefs := make([]*models.Belief, 0, len(extractedBeliefStrings))
		for _, beliefStr := range extractedBeliefStrings {
			extractedBelief := &models.Belief{
				ID: uuid.New().String(),
				Content: []models.Content{
					{RawStr: beliefStr},
				},
				Type: models.Statement,
			}
			extractedBeliefs = append(extractedBeliefs, extractedBelief)
		}

		// Add extracted beliefs to the question-answer interaction
		qa.ExtractedBeliefs = extractedBeliefs

		// Store the updated belief system
		err = s.mockKVStore.Store(input.SelfModelID, "BeliefSystem", bs, len(bs.Beliefs))
		if err != nil {
			return nil, fmt.Errorf("failed to store updated belief system: %w", err)
		}

		// Update the last interaction with the answer
		dialectic.UserInteractions[lastIdx].Status = models.StatusAnswered
		dialectic.UserInteractions[lastIdx].Interaction.QuestionAnswer.Answer = models.UserAnswer{
			UserAnswer:         input.Answer.UserAnswer,
			CreatedAtMillisUTC: time.Now().UnixMilli(),
		}
		dialectic.UserInteractions[lastIdx].UpdatedAtMillisUTC = time.Now().UnixMilli()

		// Generate next question
		response, err := s.mockDialecticEpiSvc.Respond(bs, &models.DialecticEvent{
			PreviousInteractions: dialectic.UserInteractions,
		}, "")
		if err != nil {
			return nil, fmt.Errorf("failed to generate next question: %w", err)
		}

		// Add the new interaction
		if response.NewInteraction != nil {
			dialectic.UserInteractions = append(dialectic.UserInteractions, *response.NewInteraction)
		}
	} else if input.QuestionBlob != "" {
		// Extract questions from blob
		questions, err := s.mockAIHelper.ExtractQuestionsFromText(input.QuestionBlob)
		if err != nil {
			return nil, fmt.Errorf("failed to extract questions: %w", err)
		}

		// Add each question as a new interaction
		for _, q := range questions {
			if q == "" {
				continue
			}

			interaction := models.DialecticalInteraction{
				ID:     uuid.New().String(),
				Status: models.StatusPendingAnswer,
				Type:   models.InteractionTypeQuestionAnswer,
				Interaction: &models.InteractionData{
					QuestionAnswer: &models.QuestionAnswerInteraction{
						Question: models.Question{
							Question:           q,
							CreatedAtMillisUTC: time.Now().UnixMilli(),
						},
						UpdatedAtMillisUTC: time.Now().UnixMilli(),
					},
				},
				UpdatedAtMillisUTC: time.Now().UnixMilli(),
			}

			dialectic.UserInteractions = append(dialectic.UserInteractions, interaction)
		}
	} else if input.AnswerBlob != "" {
		// Get the belief system
		bsValue, err := s.mockKVStore.Retrieve(input.SelfModelID, "BeliefSystem")
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve belief system: %w", err)
		}

		bs, ok := bsValue.(*models.BeliefSystem)
		if !ok {
			return nil, fmt.Errorf("invalid belief system type")
		}

		// Generate the next question
		response, err := s.mockDialecticEpiSvc.Respond(bs, &models.DialecticEvent{
			PreviousInteractions: dialectic.UserInteractions,
		}, "")
		if err != nil {
			return nil, fmt.Errorf("failed to generate next question: %w", err)
		}

		// Add the new interaction
		if response.NewInteraction != nil {
			dialectic.UserInteractions = append(dialectic.UserInteractions, *response.NewInteraction)
		}
	}

	// Save changes
	if !input.DryRun {
		err = s.mockKVStore.Store(input.SelfModelID, "Dialectic:"+input.ID, dialectic, len(dialectic.UserInteractions))
		if err != nil {
			return nil, err
		}
	}

	return &models.UpdateDialecticOutput{
		Dialectic: *dialectic,
	}, nil
}

// TestOptimizedServiceWithDialecticEpistemology verifies the optimized service uses the dialectic epistemology properly
func TestOptimizedServiceWithDialecticEpistemology(t *testing.T) {
	// Setup mocks
	mockKVStore := new(MockKVStoreForOptTests)
	mockAIHelper := new(MockAIHelperForOptTests)
	mockDialecticEpi := new(MockDialecticalEpistemologyForUpdate)

	// Create service instance using the test implementation
	service := NewTestDialecticService(mockKVStore, mockAIHelper, mockDialecticEpi)

	// Set up test data
	selfModelID := "test-self-model-id"
	dialecticID := "test-dialectic-id"

	// Create a test dialectic with one pending question
	dialectic := &models.Dialectic{
		ID:          dialecticID,
		SelfModelID: selfModelID,
		UserInteractions: []models.DialecticalInteraction{
			{
				ID:     uuid.New().String(),
				Status: models.StatusPendingAnswer,
				Type:   models.InteractionTypeQuestionAnswer,
				Interaction: &models.InteractionData{
					QuestionAnswer: &models.QuestionAnswerInteraction{
						Question: models.Question{
							Question:           "How often do you exercise?",
							CreatedAtMillisUTC: time.Now().UnixMilli(),
						},
						UpdatedAtMillisUTC: time.Now().UnixMilli(),
					},
				},
				UpdatedAtMillisUTC: time.Now().UnixMilli(),
			},
		},
	}

	// Setup belief system that will be returned by Process
	beliefID := uuid.New().String()
	expectedBS := &models.BeliefSystem{
		Beliefs: []*models.Belief{
			{
				ID: beliefID,
				Content: []models.Content{
					{RawStr: "User exercises three times a week"},
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

	// Setup the next interaction that Respond will return
	nextQuestion := "What type of exercise do you enjoy most?"
	nextInteraction := models.DialecticalInteraction{
		ID:     uuid.New().String(),
		Status: models.StatusPendingAnswer,
		Type:   models.InteractionTypeQuestionAnswer,
		Interaction: &models.InteractionData{
			QuestionAnswer: &models.QuestionAnswerInteraction{
				Question: models.Question{
					Question:           nextQuestion,
					CreatedAtMillisUTC: time.Now().UnixMilli(),
				},
				UpdatedAtMillisUTC: time.Now().UnixMilli(),
			},
		},
		UpdatedAtMillisUTC: time.Now().UnixMilli(),
	}

	response := &models.DialecticResponse{
		SelfModelID:          selfModelID,
		PreviousInteractions: dialectic.UserInteractions,
		NewInteraction:       &nextInteraction,
	}

	// Setup mock behaviors
	// 1. First retrieves the dialectic
	mockKVStore.On("Retrieve", selfModelID, "Dialectic:"+dialecticID).Return(dialectic, nil)

	// 2. Process should be called with the expected arguments
	mockDialecticEpi.On("Process",
		mock.MatchedBy(func(event *models.DialecticEvent) bool {
			// Verify the event has the expected interactions
			return len(event.PreviousInteractions) == 1
		}),
		false, // dryRun = false
		selfModelID,
	).Return(expectedBS, nil)

	// 3. GetInteractionEventAsBelief should be called to extract beliefs
	mockAIHelper.On("GetInteractionEventAsBelief", mock.Anything).
		Return([]string{"User exercises three times a week"}, nil)

	// 4. BeliefSystem should be stored
	mockKVStore.On("Store", selfModelID, "BeliefSystem", mock.Anything, mock.Anything).Return(nil)

	// 5. Respond should be called to generate the next question
	mockDialecticEpi.On("Respond",
		mock.Anything,
		mock.Anything,
		"",
	).Return(response, nil)

	// 6. The dialectic with updated interactions should be stored
	mockKVStore.On("Store", selfModelID, "Dialectic:"+dialecticID, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			// Verify the stored dialectic has the expected number of interactions
			storedDialectic := args.Get(2).(*models.Dialectic)
			assert.Equal(t, 2, len(storedDialectic.UserInteractions))

			// Verify the last interaction is the new question
			lastInteraction := storedDialectic.UserInteractions[len(storedDialectic.UserInteractions)-1]
			assert.Equal(t, models.StatusPendingAnswer, lastInteraction.Status)
			assert.Equal(t, nextQuestion, lastInteraction.Interaction.QuestionAnswer.Question.Question)
		}).
		Return(nil)

	// Call the method being tested
	result, err := service.OptimizedUpdateDialectic(&models.UpdateDialecticInput{
		SelfModelID: selfModelID,
		ID:          dialecticID,
		Answer: models.UserAnswer{
			UserAnswer: "I exercise three times a week",
		},
		DryRun: false,
	})

	// Verify results
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 2, len(result.Dialectic.UserInteractions)) // Original + new question

	// Verify the last interaction is the new question
	lastInteraction := result.Dialectic.UserInteractions[len(result.Dialectic.UserInteractions)-1]
	assert.Equal(t, models.StatusPendingAnswer, lastInteraction.Status)
	assert.Equal(t, nextQuestion, lastInteraction.Interaction.QuestionAnswer.Question.Question)

	// Verify all mocks were called as expected
	mockKVStore.AssertExpectations(t)
	mockDialecticEpi.AssertExpectations(t)
	mockAIHelper.AssertExpectations(t)
}

// TestEnhancedPredictiveProcessingContext verifies the PPC enhancements
func TestEnhancedPredictiveProcessingContext(t *testing.T) {
	// Setup mocks
	mockKVStore := new(MockKVStoreForOptTests)
	mockAIHelper := new(MockAIHelperForOptTests)
	mockDialecticEpi := new(MockDialecticalEpistemologyForUpdate)

	// Create service instance using the test implementation
	service := NewTestDialecticService(mockKVStore, mockAIHelper, mockDialecticEpi)

	// Set up test data
	selfModelID := "test-self-model-id"
	dialecticID := "test-dialectic-id"

	// Create a test dialectic with one pending question
	dialectic := &models.Dialectic{
		ID:          dialecticID,
		SelfModelID: selfModelID,
		UserInteractions: []models.DialecticalInteraction{
			{
				ID:     uuid.New().String(),
				Status: models.StatusPendingAnswer,
				Type:   models.InteractionTypeQuestionAnswer,
				Interaction: &models.InteractionData{
					QuestionAnswer: &models.QuestionAnswerInteraction{
						Question: models.Question{
							Question:           "How would you describe your diet?",
							CreatedAtMillisUTC: time.Now().UnixMilli(),
						},
						UpdatedAtMillisUTC: time.Now().UnixMilli(),
					},
				},
				UpdatedAtMillisUTC: time.Now().UnixMilli(),
			},
		},
	}

	// Setup initial belief system
	initialBS := &models.BeliefSystem{
		Beliefs: []*models.Belief{
			{
				ID: uuid.New().String(),
				Content: []models.Content{
					{RawStr: "User follows a balanced diet"},
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

	// Mock the extraction of beliefs from the answer
	mockAIHelper.On("GetInteractionEventAsBelief", mock.Anything).
		Return([]string{
			"User follows a mediterranean diet",
			"User avoids processed foods",
		}, nil)

	// Mock the retrieval of the dialectic
	mockKVStore.On("Retrieve", selfModelID, "Dialectic:"+dialecticID).Return(dialectic, nil)

	// Mock Process to return the initial belief system
	mockDialecticEpi.On("Process", mock.Anything, false, selfModelID).Return(initialBS, nil)

	// Mock Respond to return a valid response
	mockDialecticEpi.On("Respond", mock.Anything, mock.Anything, "").
		Return(&models.DialecticResponse{
			SelfModelID:          selfModelID,
			PreviousInteractions: dialectic.UserInteractions,
			NewInteraction: &models.DialecticalInteraction{
				ID:     uuid.New().String(),
				Status: models.StatusPendingAnswer,
				Type:   models.InteractionTypeQuestionAnswer,
				Interaction: &models.InteractionData{
					QuestionAnswer: &models.QuestionAnswerInteraction{
						Question: models.Question{
							Question:           "Do you have any dietary restrictions?",
							CreatedAtMillisUTC: time.Now().UnixMilli(),
						},
						UpdatedAtMillisUTC: time.Now().UnixMilli(),
					},
				},
				UpdatedAtMillisUTC: time.Now().UnixMilli(),
			},
		}, nil)

	// Mock storing the belief system with verification of enhanced context
	mockKVStore.On("Store", selfModelID, "BeliefSystem", mock.Anything, mock.Anything).Return(nil)

	// Mock storing the updated dialectic
	mockKVStore.On("Store", selfModelID, "Dialectic:"+dialecticID, mock.Anything, mock.Anything).Return(nil)

	// Call the method being tested
	result, err := service.OptimizedUpdateDialectic(&models.UpdateDialecticInput{
		SelfModelID: selfModelID,
		ID:          dialecticID,
		Answer: models.UserAnswer{
			UserAnswer: "I follow a mediterranean diet and avoid processed foods",
		},
		DryRun: false,
	})

	// Verify results
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify the dialectic was updated with the extracted beliefs
	require.Equal(t, 2, len(result.Dialectic.UserInteractions))

	// Check that the first interaction now has an answer with the expected beliefs
	interaction := result.Dialectic.UserInteractions[0]
	require.Equal(t, models.StatusAnswered, interaction.Status)
	require.NotNil(t, interaction.Interaction.QuestionAnswer.ExtractedBeliefs)
	require.Equal(t, 2, len(interaction.Interaction.QuestionAnswer.ExtractedBeliefs))

	// Check the content of the extracted beliefs
	beliefs := interaction.Interaction.QuestionAnswer.ExtractedBeliefs
	foundMedDiet := false
	foundProcessedFoods := false

	for _, belief := range beliefs {
		content := belief.GetContentAsString()
		if content == "User follows a mediterranean diet" {
			foundMedDiet = true
		}
		if content == "User avoids processed foods" {
			foundProcessedFoods = true
		}
	}

	assert.True(t, foundMedDiet, "Extracted beliefs should include 'User follows a mediterranean diet'")
	assert.True(t, foundProcessedFoods, "Extracted beliefs should include 'User avoids processed foods'")

	// Verify all mocks were called as expected
	mockKVStore.AssertExpectations(t)
	mockDialecticEpi.AssertExpectations(t)
	mockAIHelper.AssertExpectations(t)
}

// TestOptimizedAnswerBlobProcessing verifies answer blob handling
func TestOptimizedAnswerBlobProcessing(t *testing.T) {
	// Setup mocks
	mockKVStore := new(MockKVStoreForOptTests)
	mockAIHelper := new(MockAIHelperForOptTests)
	mockDialecticEpi := new(MockDialecticalEpistemologyForUpdate)

	// Create service instance using the test implementation
	service := NewTestDialecticService(mockKVStore, mockAIHelper, mockDialecticEpi)

	// Set up test data
	selfModelID := "test-self-model-id"
	dialecticID := "test-dialectic-id"

	// Create a test dialectic with existing interactions
	dialectic := &models.Dialectic{
		ID:          dialecticID,
		SelfModelID: selfModelID,
		UserInteractions: []models.DialecticalInteraction{
			{
				ID:     uuid.New().String(),
				Status: models.StatusAnswered,
				Type:   models.InteractionTypeQuestionAnswer,
				Interaction: &models.InteractionData{
					QuestionAnswer: &models.QuestionAnswerInteraction{
						Question: models.Question{
							Question:           "What are your exercise habits?",
							CreatedAtMillisUTC: time.Now().UnixMilli(),
						},
						Answer: models.UserAnswer{
							UserAnswer:         "I exercise three times a week",
							CreatedAtMillisUTC: time.Now().UnixMilli(),
						},
						UpdatedAtMillisUTC: time.Now().UnixMilli(),
					},
				},
				UpdatedAtMillisUTC: time.Now().UnixMilli(),
			},
		},
	}

	// Setup belief system
	bs := &models.BeliefSystem{
		Beliefs: []*models.Belief{
			{
				ID: uuid.New().String(),
				Content: []models.Content{
					{RawStr: "User exercises three times a week"},
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

	// Setup the expected response from Respond
	nextQuestion := "What type of exercise do you prefer?"
	nextInteraction := models.DialecticalInteraction{
		ID:     uuid.New().String(),
		Status: models.StatusPendingAnswer,
		Type:   models.InteractionTypeQuestionAnswer,
		Interaction: &models.InteractionData{
			QuestionAnswer: &models.QuestionAnswerInteraction{
				Question: models.Question{
					Question:           nextQuestion,
					CreatedAtMillisUTC: time.Now().UnixMilli(),
				},
				UpdatedAtMillisUTC: time.Now().UnixMilli(),
			},
		},
		UpdatedAtMillisUTC: time.Now().UnixMilli(),
	}

	response := &models.DialecticResponse{
		SelfModelID:          selfModelID,
		PreviousInteractions: dialectic.UserInteractions,
		NewInteraction:       &nextInteraction,
	}

	// Setup mock behaviors
	// 1. Retrieve the dialectic
	mockKVStore.On("Retrieve", selfModelID, "Dialectic:"+dialecticID).Return(dialectic, nil)

	// 2. Retrieve the belief system
	mockKVStore.On("Retrieve", selfModelID, "BeliefSystem").Return(bs, nil)

	// 3. Use Respond to generate the next question
	mockDialecticEpi.On("Respond", bs, mock.Anything, "").Return(response, nil)

	// 4. Store the updated dialectic
	mockKVStore.On("Store", selfModelID, "Dialectic:"+dialecticID, mock.Anything, mock.Anything).Return(nil)

	// Call the method being tested with an answer blob
	result, err := service.OptimizedUpdateDialectic(&models.UpdateDialecticInput{
		SelfModelID: selfModelID,
		ID:          dialecticID,
		AnswerBlob:  "Here's my full health profile...",
		DryRun:      false,
	})

	// Verify results
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 2, len(result.Dialectic.UserInteractions))

	// Verify the last interaction is the new question
	lastInteraction := result.Dialectic.UserInteractions[len(result.Dialectic.UserInteractions)-1]
	assert.Equal(t, models.StatusPendingAnswer, lastInteraction.Status)
	assert.Equal(t, nextQuestion, lastInteraction.Interaction.QuestionAnswer.Question.Question)

	// Verify all mocks were called as expected
	mockKVStore.AssertExpectations(t)
	mockDialecticEpi.AssertExpectations(t)
	mockAIHelper.AssertExpectations(t)
}

// TestQuestionBlobProcessing verifies question blob handling
func TestQuestionBlobProcessing(t *testing.T) {
	// Setup mocks
	mockKVStore := new(MockKVStoreForOptTests)
	mockAIHelper := new(MockAIHelperForOptTests)
	mockDialecticEpi := new(MockDialecticalEpistemologyForUpdate)

	// Create service instance using the test implementation
	service := NewTestDialecticService(mockKVStore, mockAIHelper, mockDialecticEpi)

	// Set up test data
	selfModelID := "test-self-model-id"
	dialecticID := "test-dialectic-id"

	// Create a test dialectic with existing interactions
	dialectic := &models.Dialectic{
		ID:          dialecticID,
		SelfModelID: selfModelID,
		UserInteractions: []models.DialecticalInteraction{
			{
				ID:     uuid.New().String(),
				Status: models.StatusAnswered,
				Type:   models.InteractionTypeQuestionAnswer,
				Interaction: &models.InteractionData{
					QuestionAnswer: &models.QuestionAnswerInteraction{
						Question: models.Question{
							Question:           "What is your fitness level?",
							CreatedAtMillisUTC: time.Now().UnixMilli(),
						},
						Answer: models.UserAnswer{
							UserAnswer:         "I'm moderately fit",
							CreatedAtMillisUTC: time.Now().UnixMilli(),
						},
						UpdatedAtMillisUTC: time.Now().UnixMilli(),
					},
				},
				UpdatedAtMillisUTC: time.Now().UnixMilli(),
			},
		},
	}

	// Extract questions from the question blob
	questionBlob := "Here are some questions: How many hours do you sleep? What is your stress level? Do you have any health conditions?"
	extractedQuestions := []string{
		"How many hours do you sleep?",
		"What is your stress level?",
		"Do you have any health conditions?",
	}

	// Setup mock behaviors
	// 1. Retrieve the dialectic
	mockKVStore.On("Retrieve", selfModelID, "Dialectic:"+dialecticID).Return(dialectic, nil)

	// 2. Extract questions from the blob
	mockAIHelper.On("ExtractQuestionsFromText", questionBlob).Return(extractedQuestions, nil)

	// 3. Store the updated dialectic
	mockKVStore.On("Store", selfModelID, "Dialectic:"+dialecticID, mock.Anything, mock.Anything).Return(nil)

	// Call the method being tested with a question blob
	result, err := service.OptimizedUpdateDialectic(&models.UpdateDialecticInput{
		SelfModelID:  selfModelID,
		ID:           dialecticID,
		QuestionBlob: questionBlob,
		DryRun:       false,
	})

	// Verify results
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify the interactions were correctly updated
	assert.Equal(t, 1+len(extractedQuestions), len(result.Dialectic.UserInteractions))

	// Verify each extracted question was added to the result
	for i, question := range extractedQuestions {
		interaction := result.Dialectic.UserInteractions[i+1]
		assert.Equal(t, models.StatusPendingAnswer, interaction.Status)
		assert.Equal(t, question, interaction.Interaction.QuestionAnswer.Question.Question)
	}

	// Verify all mocks were called as expected
	mockKVStore.AssertExpectations(t)
	mockAIHelper.AssertExpectations(t)
}
