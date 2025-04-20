# Belief System Structure in OptimizedDialecticService

This document provides a detailed explanation of the belief system structure that's created and modified during the `OptimizedUpdateDialectic` process.

## Overview

The `OptimizedDialecticService` enhances the belief system with additional context information through the `PredictiveProcessingContext` (PPC). This creates a rich data structure that tracks not just beliefs but also observation contexts, confidence ratings, and relationships between various elements.

## Core Components

### BeliefSystem

The `BeliefSystem` is the root structure that contains:

```go
type BeliefSystem struct {
    Beliefs           []*Belief           // Array of belief objects
    EpistemicContexts *EpistemicContexts  // Container for context information
}
```

### Belief

Each `Belief` represents a discrete piece of knowledge or opinion:

```go
type Belief struct {
    ID          string      // Unique identifier 
    SelfModelID string      // ID of the self model this belief belongs to
    Version     int32       // Version number for tracking changes
    Type        BeliefType  // Type classification (STATEMENT, FALSIFIABLE, CAUSAL)
    Content     []*Content  // The actual belief content
}
```

### EpistemicContexts

`EpistemicContexts` serves as a container for various types of epistemic contexts:

```go
type EpistemicContexts struct {
    EpistemicContexts []*EpistemicContext
}
```

### EpistemicContext

An `EpistemicContext` can be one of several types, with the PredictiveProcessingContext being the most important for the optimized service:

```go
type EpistemicContext struct {
    // Union field Context can be only one of the following:
    PredictiveProcessingContext *PredictiveProcessingContext
    BeliefTreeContext           *BeliefTreeContext
    ExploratoryContext          *ExploratoryContext
}
```

### PredictiveProcessingContext (PPC)

The `PredictiveProcessingContext` is the core enhancement added by the optimized service:

```go
type PredictiveProcessingContext struct {
    ObservationContexts []*ObservationContext  // Contexts for observations
    BeliefContexts      []*BeliefContext       // Contexts for beliefs
}
```

### ObservationContext

An `ObservationContext` represents a situation or domain in which observations are made:

```go
type ObservationContext struct {
    ID             string      // Unique identifier
    Name           string      // Human-readable name (e.g., "Response to 'How do you feel about X?'")
    ParentID       string      // Optional parent context
    PossibleStates []string    // Possible states this context can be in
}
```

### BeliefContext

A `BeliefContext` links a belief to an observation context and provides additional metadata:

```go
type BeliefContext struct {
    BeliefID             string                      // References a specific belief
    ObservationContextID string                      // References an observation context
    ConfidenceRatings    []*ConfidenceRating         // Confidence scores
    ConditionalProbs     map[string]float32          // Conditional probabilities
    EpistemicEmotion     EpistemicEmotion            // Associated emotion (CONFIRMATION, SURPRISE, etc.)
    EmotionIntensity     float32                     // Intensity of the emotion
}
```

The `BeliefContext` is a critical structure for implementing predictive processing in the system. It contains several important elements:

1. **References**: Links a specific belief to an observation context through unique identifiers
2. **Confidence Ratings**: Tracks how confident the system is about the belief
3. **Conditional Probabilities**: Maps possible states to probability values (see detailed explanation below)
4. **Epistemic Emotions**: Captures the emotional response to beliefs (e.g., surprise when beliefs are violated)
5. **Emotion Intensity**: Quantifies the strength of the epistemic emotion

#### Confidence Ratings

```go
type ConfidenceRating struct {
    ConfidenceScore float32     // A value between 0 and 1 representing confidence
    Default         bool        // Whether this is a default confidence rating
    AssessmentDate  int64       // When the confidence was assessed (Unix timestamp)
    Source          string      // Source of the confidence assessment (e.g. "user", "system")
}
```

Confidence ratings track how strongly the system believes in a specific belief. Multiple confidence ratings can be stored for a single belief, enabling tracking of confidence changes over time.

#### Conditional Probabilities

The conditional probability map is a critical component that captures the relationship between a belief and expected observations. Each entry maps:

- **Key**: A potential state of the observation context (e.g., "High energy levels", "Low energy levels")
- **Value**: The probability (0-1) that this state would occur if the belief is true

For example, if the belief is "Regular exercise improves energy levels" and the observation context is "Energy State", the conditional probabilities might include:
```
{
  "High energy levels": 0.8,
  "Low energy levels": 0.2
}
```

This structure enables the system to:
1. Make predictions about expected observations
2. Compare predictions with actual observations
3. Update confidence ratings based on observed discrepancies
4. Generate appropriate epistemic emotions in response to confirmations or violations

## Conditional Probabilities and Discrepancy Detection

Conditional probabilities play a crucial role in implementing predictive processing principles within the belief system. The system uses these probabilities to:

### 1. Prediction Generation

The belief system can generate predictions about expected observations based on the conditional probabilities attached to beliefs. For example, if a belief has a high conditional probability for a specific state, the system expects to observe that state.

### 2. Discrepancy Detection

When actual observations differ from predicted observations, the system detects discrepancies. This is calculated by comparing the most probable state (based on conditional probabilities) with the actual observed state.

The formula for discrepancy can be expressed as:

```
Discrepancy = 1 - P(observed_state | belief)
```

Where `P(observed_state | belief)` is the conditional probability of the observed state given the belief.

### 3. Belief Updating

Discrepancies drive belief updating through several mechanisms:

1. **Confidence Adjustment**: The confidence in a belief decreases when discrepancies are detected
2. **Epistemic Emotion Generation**: Discrepancies trigger epistemic emotions like surprise or confusion
3. **Revision Triggers**: Large discrepancies can trigger revision of the belief itself

### 4. Multi-level Processing

The hierarchical structure of observation contexts allows for multi-level predictive processing:

1. Higher-level beliefs generate predictions for lower-level contexts
2. Discrepancies at lower levels can propagate upward
3. Complex belief networks can encode sophisticated predictive models

### Example of Discrepancy Detection

Consider a user who believes "I sleep 8 hours per night" with these conditional probabilities:
```
{
  "High energy levels": 0.9,
  "Low energy levels": 0.1
}
```

If the user then reports "Low energy levels", the system calculates:
- Predicted state: "High energy levels" (0.9 probability)
- Actual state: "Low energy levels"
- Discrepancy: 1 - 0.1 = 0.9 (high discrepancy)

This high discrepancy would:
1. Reduce confidence in the original belief
2. Generate a SURPRISE epistemic emotion
3. Potentially trigger follow-up questions about sleep quality or other factors

## Creation Process in OptimizedUpdateDialectic

When `OptimizedUpdateDialectic` processes a user answer, it:

1. **Retrieves the dialectic** from the key-value store
2. **Processes the belief system** with the `dialecticEpiSvc.Process` method
3. **Extracts beliefs** from the latest answer using `aiHelper.GetInteractionEventAsBelief`
4. **Updates the PPC** with the new observation and belief contexts via `updatePredictiveProcessingContext`
5. **Stores the updated belief system** with both updated and new beliefs

### PPC Enhancement Flow

The function `updatePredictiveProcessingContext` creates:

1. **ObservationContext**: 
   - Creates a context named after the question (e.g., "Response to 'What foods do you eat?'")
   - Assigns possible states like "Positive", "Negative", "Neutral"

2. **BeliefContext for each new belief**:
   - Links to the newly created ObservationContext
   - Sets default confidence ratings (0.8)
   - Sets default conditional probabilities for possible states
   - Sets the default epistemic emotion to "Confirmation"
   - Sets default emotion intensity (0.5)

## Relationships Between Components

The relationships in the belief system form a complex network:

1. **Beliefs ⟷ BeliefContexts**: 
   - Each BeliefContext references a specific Belief via the BeliefID
   - A Belief can be referenced by multiple BeliefContexts

2. **ObservationContexts ⟷ BeliefContexts**:
   - BeliefContexts reference ObservationContexts via ObservationContextID
   - An ObservationContext can be referenced by multiple BeliefContexts

3. **Hierarchical ObservationContexts**:
   - ObservationContexts can have parent-child relationships via ParentID

4. **Conditional Relationships**:
   - BeliefContexts contain conditional probabilities that link beliefs to expected observations
   - These conditional probabilities enable prediction and discrepancy detection

## Differences from Standard Implementation

The optimized implementation enhances the belief system with:

1. **Enhanced PPC**: 
   - Creates richer observation contexts based on the conversation
   - Links beliefs to specific interactions and questions

2. **Optimized Belief Extraction**:
   - Extracts beliefs directly from the latest answer
   - Checks for duplicate beliefs before adding them

3. **Improved Confidence Tracking**:
   - Sets default confidence ratings for new beliefs
   - Provides an extensible structure for future confidence updates

4. **Conditional Probability Mapping**:
   - Maps beliefs to expected observations through conditional probabilities
   - Enables detection of discrepancies between expectations and **reality**

5. **Epistemic Emotion Generation**:
   - Captures emotional responses to belief confirmation or violation
   - Provides intensity values to quantify emotional responses

## Example Structure

Here's a simplified example of the resulting structure after processing a user answer "I exercise three times a week":

```
BeliefSystem
├── Beliefs
│   ├── Belief_1: "I believe exercise three times a week is beneficial for health"
│   └── Belief_2: "Regular exercise improves mental clarity"
│
└── EpistemicContexts
    └── EpistemicContext (PredictiveProcessingContext)
        ├── ObservationContexts
        │   └── ObservationContext_1
        │       ├── ID: "oc_123"
        │       ├── Name: "Beneficial for health"
        │       └── PossibleStates: ["Improved health", "No health change", "Decreased health"]
        │
        └── BeliefContexts
            ├── BeliefContext_1
            │   ├── BeliefID: Reference to Belief_1
            │   ├── ObservationContextID: "oc_123"
            │   ├── ConfidenceRatings: [0.8]
            │   ├── ConditionalProbs: {"Improved health": 0.9, "No health change": 0.08, "Decreased health": 0.02}
            │   ├── EpistemicEmotion: CONFIRMATION
            │   └── EmotionIntensity: 0.5
            │
            └── BeliefContext_2
                ├── BeliefID: Reference to Belief_2
                ├── ObservationContextID: "oc_123"
                ├── ConfidenceRatings: [0.8]
                ├── ConditionalProbs: {"Improved health": 0.85, "No health change": 0.13, "Decreased health": 0.02}
                ├── EpistemicEmotion: CONFIRMATION
                └── EmotionIntensity: 0.5
```

## API Response Structure

When returned via the API, the belief system includes all these components serialized as Protocol Buffers. The client can then use this rich structure to:

1. Display beliefs organized by context
2. Show confidence levels for different beliefs
3. Track the evolution of beliefs over time
4. Understand the emotional response to different beliefs
5. Identify discrepancies between beliefs and observations
6. Visualize predictive relationships between beliefs and expected outcomes

## Usage in Optimized Dialectic Process

This enhanced belief system structure enables:

1. **More contextual follow-up questions** by understanding which contexts have been explored
2. **Better confidence management** by tracking confidence in different beliefs
3. **Emotion-aware responses** based on the epistemic emotions associated with beliefs
4. **Efficient conflict detection** by identifying beliefs with contradictory confidence ratings
5. **Discrepancy-driven questioning** by focusing on areas where predictions and observations differ
6. **Belief revision guidance** by highlighting beliefs with low confidence or high discrepancy
7. **Personalized follow-up** based on the specific conditional probabilities associated with a user's beliefs 