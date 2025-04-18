# Server Test Documentation: Preventing Regressions

## Overview

This document provides a comprehensive overview of the test infrastructure in the `Server/` directory. These tests are critical for maintaining system stability and preventing regressions as the codebase evolves. The test suite covers key aspects of the API server, SDK functionality, AI components, and their integrations.

## Test Categories

### 1. Integration Tests (`Server/tests/integration/`)

Tests in this directory validate end-to-end functionality of the API server and its components.

#### `belief_system_detail_test.go`
- **Purpose**: Validates belief system API functionality
- **Key Test Cases**:
  - Retrieval of belief systems with proper structure
  - Verification of epistemic contexts and their components
  - Validation of belief system object types
- **Regression Prevention**:
  - Ensures belief system data structures maintain expected shape
  - Prevents breaking changes to API response formats
  - Maintains consistency in data type handling

#### `integration_test.go`
- **Purpose**: Core integration test suite covering multiple API functionalities
- **Key Test Cases**:
  - API key management and authentication
  - Server initialization and teardown
  - CRUD operations for beliefs and dialectics
  - Store management (in-memory and persistent)
- **Regression Prevention**:
  - Ensures core API endpoints function correctly
  - Validates authentication mechanisms
  - Verifies data persistence across operations
  - Maintains proper initialization and cleanup sequences

#### `dashboard_test.go`
- **Purpose**: Tests dashboard-specific functionality
- **Regression Prevention**:
  - Ensures dashboard data retrieval and processing remains functional
  - Prevents regressions in dashboard-related endpoints

### 2. SDK Tests (`Server/tests/sdk/`)

These tests validate the SDK components and their interactions with the core system.

#### `dialectic_learning_test.go`
- **Purpose**: Tests dialectic learning processes
- **Key Test Cases**:
  - Creation of dialectics with learning objectives
  - Multi-round question-answer cycles
  - Belief extraction and learning progress tracking
  - Integration with self-models
- **Regression Prevention**:
  - Ensures dialectic learning algorithms function correctly
  - Validates belief extraction from interactions
  - Maintains learning progress tracking
  - Preserves integrity of multi-turn interactions

#### `self_model_integration_test.go`
- **Purpose**: Tests self-model functionality
- **Regression Prevention**:
  - Ensures creation and management of self-models
  - Prevents regressions in belief system integration with self-models
  - Maintains philosophy application to self-models

#### `survey_integration_test.go` & `chat_survey_integration_test.go`
- **Purpose**: Tests survey functionality and chat-based surveys
- **Regression Prevention**:
  - Ensures survey creation, processing, and response handling
  - Maintains chat-based survey interactions
  - Preserves survey data integrity

#### `dialectic_qa_test.go`
- **Purpose**: Tests question-answer dynamics in dialectics
- **Regression Prevention**:
  - Ensures proper question generation and answer processing
  - Maintains belief extraction from Q&A sessions
  - Prevents regressions in dialectic interaction flows

#### `preprocess_question_answer_integration_test.go`
- **Purpose**: Tests preprocessing of Q&A data
- **Regression Prevention**:
  - Ensures proper data preprocessing for effective belief extraction
  - Maintains integration between preprocessing and downstream components

#### `developer_user_integration_test.go`
- **Purpose**: Tests developer-user interactions
- **Regression Prevention**:
  - Ensures API key management for developers
  - Maintains separation of developer and user contexts
  - Prevents regressions in authentication flows

### 3. AI Tests (`Server/tests/ai/`)

These tests validate AI functionality essential to the system's operation.

#### `qa_preprocessing_test.go`
- **Purpose**: Tests AI preprocessing functionality for question-answer pairs
- **Key Test Cases**:
  - Handling of numbered answers
  - Matching answers to questions
  - Processing multi-paragraph responses
  - Semantic matching between questions and answers
  - Extraction and cleaning of questions
- **Regression Prevention**:
  - Ensures AI preprocessing algorithms function correctly
  - Maintains text processing quality
  - Prevents regressions in semantic matching
  - Preserves answer extraction accuracy

## Critical Areas to Monitor

1. **Belief System Management**
   - Any changes to `BeliefSystem` structure require careful testing
   - Modifications to belief creation/update logic need validation with `TestCreateBelief` and `TestUpdateBelief`
   - Changes to epistemic contexts must be verified with `TestBeliefSystemAPIIntegration`

2. **Dialectical Epistemology Processing**
   - Changes to `DialecticalEpistemology` service (particularly `Process` and `Respond` methods) must be validated with `TestDialecticLearningCycle`
   - Alterations to belief extraction from dialectics require testing with `TestDialecticLearning`
   - Updates to the belief update process in `Process` method (see `Server/svc/dialectical_epistemology_svc.go:63`) should be tested for regression

3. **AI Integration**
   - Updates to AI helper functionality must be verified with `TestMatchAnswersToQuestions` and related tests
   - Changes to question generation or answer processing require testing with QA preprocessing tests

4. **API Authentication**
   - Modifications to API key handling must be validated with integration tests
   - Changes to authentication flow require testing with `developer_user_integration_test.go`

## Test Dependencies

1. **Environmental Requirements**
   - Tests require `OPENAI_API_KEY` environment variable
   - Some tests create temporary files and expect appropriate filesystem permissions

2. **External Services**
   - AI tests depend on OpenAI API connectivity
   - Tests skip when required environment variables are missing

## How to Run Tests

Standard commands to run tests:

```bash
# Run all tests
go test ./...

# Run specific test category
go test ./tests/integration
go test ./tests/sdk
go test ./tests/ai

# Run specific test
go test ./tests/sdk -run TestDialecticLearning
```

## Maintaining Test Integrity

1. **When Adding Features**
   - Add corresponding test cases that validate new functionality
   - Ensure existing tests aren't broken by new features
   - Consider edge cases specific to the new feature

2. **When Fixing Bugs**
   - Add regression tests that would have caught the bug
   - Ensure the fix doesn't break existing tests
   - Document the regression scenario

3. **When Refactoring**
   - Run all tests before and after refactoring
   - Maintain test coverage percentage
   - Update tests if refactoring changes API signatures or behavior

## Conclusion

The test suite provides comprehensive coverage of the server's functionality. By keeping these tests passing and expanding them with new features, you can prevent regressions and maintain system stability. When making changes to the codebase, always ensure that relevant tests pass and consider adding new tests for new functionality or edge cases. 