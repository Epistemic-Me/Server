package ai_helper

import (
	"context"
	"encoding/json"
	"epistemic-me-core/svc/models"
	"fmt"
	"log"
	"regexp"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

const DIALECTICAL_STRATEGY = `What is a Question?
In the BC dialectic context, a question creates a conceptual framework for learning. It acts as a container for prior beliefs, evidence collection, and answer verification.

It is an inference of the world based on prior beliefs that guides evidence collection and observation. A good question can shape knowledge that is valuable to our future experiences. The question provides the boundaries of an unambiguous explanation for a causal pattern.

If we ask the question, "What is a healthy day?" The first phase of our inquiry explores whether our prior beliefs create sufficient knowledge to predict that our next day can be healthy.

Let's assume that our epistemic emotions find a gap in the predictability of our prior beliefs and guide curiosity. Now, the question will guide us to collect more evidence to either justify our beliefs more deeply or update them. In both cases, our curiosity guides a more accurate prediction of how we will make tomorrow a healthy day.

In seeking evidence, we can look to the experiences of our ancestors or to the experiences of our contemporaries. In the former, we consider the synthesis of all people who have lived a healthy life and the "best practices" synthesized through our research and education systems. In the latter, we consider the direct experiences of others who share our beliefs.

Ultimately, we're seeking evidence that is valuable to our learning process to answer the question.
Who has beliefs that generate a predictive answer to this question?
What evidence justifies their beliefs?
Where are the beliefs predictive? (ie, in what observation context)
Why does evidence justify their beliefs?
When in their history have they applied these beliefs predictively?
How does the evidence justify their beliefs?

What is an Answer?
An answer is an explanatory narrative that describes a causal pattern within a conceptual framework. Specifically, the answer is consistent with the possibilities of the belief systems (generative) and probabilities of value systems (discriminant) that contextualize an inquiry.

What is a Belief?
Prior beliefs are stated as an expectation for an observable causal change; each belief carries a level of certainty. For instance, "I believe that quality sleep is required for high energy the following day" (ie, Sleep Belief). This belief might carry a 90% certainty of being predictive.

When beliefs are evidenced and certain above a certain threshold, they create knowledge. Beliefs can be evidenced by experience or by theory.

The user's Sleep Belief could be evidenced by their own experience with quality sleep and energy, the experience of others, theories, research, etc. Epistemologically, the user is seeking evidence that a belief is predictive.

The user can take action to seek out further evidence to 

What is Evidence?
Ladder of evidence 
Conceptual Framework of reproducibility: Theory tested in nothing (ie, theory only), cells, mice, humans
Type of reproducibility, how can the claim be reproduced:
All cells show behavior
All animals/mice/mammals
All Humans/types of humans
Organism model: ORT of homeostasis/allostasis
What homeostatic conserved?
Phenotype (homeostatic) Morphology for Syntopical Inquiry
Considering what we can know from evolution
adaptation capability passed on..
while maintaining allostasis in the organism
how does the preservation of an epigenetic possibility affect the population-level genetics?
`

type LLMModel string

// Define the constants
const (
	GPT_LATEST LLMModel = openai.GPT4oMini
)

type AIHelper struct {
	client *openai.Client
}

type InteractionEvent struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// Constructor for AIHelper
func NewAIHelper(apiKey string) *AIHelper {
	client := openai.NewClient(apiKey)
	return &AIHelper{
		client: client,
	}
}

func (aih *AIHelper) GenerateQuestion(beliefSystem string, previousEvents []InteractionEvent) (string, error) {
	systemContext := fmt.Sprintf("Given these definitions %s. Generate a single question to further understand the user's belief system.", DIALECTICAL_STRATEGY)
	if len(beliefSystem) > 0 {
		systemContext += fmt.Sprintf(" The user's current belief system is %s", beliefSystem)
	}
	if len(previousEvents) > 0 {
		events, err := json.Marshal(previousEvents)
		if err != nil {
			return "", err
		}
		systemContext += fmt.Sprintf(" Ask a single novel question given the existing questions asked: %s", events)
	}

	response, err := aih.client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: string(GPT_LATEST),
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: systemContext},
			{Role: "user", Content: "Please ask me a question to further inquire into my belief system, just respond with the question directly."},
		},
	})
	if err != nil {
		return "", err
	}

	return response.Choices[0].Message.Content, nil
}

func (aih *AIHelper) GenerateBeliefSystem(activeBeliefs []string) (string, error) {
	beliefs, err := json.Marshal(activeBeliefs)
	if err != nil {
		return "", err
	}

	response, err := aih.client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: string(GPT_LATEST),
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: fmt.Sprintf("Given these definitions %s. Construct a belief system based on these events", DIALECTICAL_STRATEGY)},
			{Role: "user", Content: fmt.Sprintf("Please respond curtly with just a concise representation of my belief system, %s", beliefs)},
		},
	})
	if err != nil {
		return "", err
	}

	return response.Choices[0].Message.Content, nil
}

func (aih *AIHelper) GetInteractionEventAsBelief(event InteractionEvent) (string, error) {
	eventJson, err := json.Marshal(event)
	if err != nil {
		return "", err
	}

	response, err := aih.client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: string(GPT_LATEST),
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: fmt.Sprintf(`Given these definitions %s. 
				Extract a belief from the user's response. 
				Return ONLY a JSON object with a "belief" field containing the belief statement.
				Example: {"belief": "I believe that quality sleep is essential for energy"}`, DIALECTICAL_STRATEGY)},
			{Role: "user", Content: fmt.Sprintf("Extract a belief from this interaction: %s", eventJson)},
		},
	})
	if err != nil {
		log.Printf("Error from AI: %v", err)
		return "", err
	}

	// Log the AI response
	log.Printf("AI response: %s", response.Choices[0].Message.Content)

	// Parse the JSON response
	var beliefResponse struct {
		Belief string `json:"belief"`
	}
	if err := json.Unmarshal([]byte(response.Choices[0].Message.Content), &beliefResponse); err != nil {
		return "", fmt.Errorf("failed to parse belief response: %w", err)
	}

	return beliefResponse.Belief, nil
}

func (aih *AIHelper) ExtractBeleifsFromResource(resource models.Resource) ([]string, error) {

	response, err := aih.client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: string(GPT_LATEST),
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: fmt.Sprintf(`Given these definitions %s. 
				Extract a series of beleifs from this document. 
				Return ONLY a JSON object with a "beliefs" field containing the array of belief statement.
				Example: {"beliefs": ["Quality sleep is essential for energy", "HRV is a biomarker for good health"]}`, DIALECTICAL_STRATEGY)},
			{Role: "user", Content: fmt.Sprintf("Extract a belief from this document: %s", resource.Content)},
		},
	})
	if err != nil {
		log.Printf("Error from AI: %v", err)
		return nil, err
	}

	// Log the AI response
	log.Printf("AI response: %s", response.Choices[0].Message.Content)

	// Parse the JSON response
	var beliefResponse struct {
		Beliefs []string `json:"beliefs"`
	}
	if err := json.Unmarshal([]byte(response.Choices[0].Message.Content), &beliefResponse); err != nil {
		return nil, fmt.Errorf("failed to parse belief response: %w", err)
	}

	return beliefResponse.Beliefs, nil
}

func (aih *AIHelper) DetermineBeliefValidity(oldBeliefs []*models.Belief, newBeliefs []*models.Belief) ([]string, []string, error) {
	// STEP 1: Prepare the system instruction.
	// We now request two arrays of IDs—kept vs. deleted—in a clearly specified JSON structure.
	systemInstruction := `
You are given two lists:
1. Old beliefs (each with an ID and content).
2. New beliefs.

Task: Determine which old beliefs are invalidated by the new beliefs and which remain valid.

Return a JSON object with two arrays of belief IDs:
{
  "kept_belief_ids": ["...","..."],
  "deleted_belief_ids": ["...","..."]
}

Important:
 - "kept_belief_ids" should be the IDs of old beliefs that remain valid.
 - "deleted_belief_ids" should be the IDs of old beliefs that are no longer valid.
 - Do NOT include the new beliefs in the returned IDs.
`

	// STEP 2: Marshal both old and new beliefs for the AI.
	oldBeliefsJSON, err := json.Marshal(oldBeliefs)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal oldBeliefs: %w", err)
	}

	newBeliefsJSON, err := json.Marshal(newBeliefs)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal newBeliefs: %w", err)
	}

	// STEP 3: Prompt: Provide old and new beliefs, request two ID lists.
	prompt := fmt.Sprintf(`
Old Beliefs (JSON):
%s

New Beliefs (JSON):
%s

Please return a JSON object like this:
{
  "kept_belief_ids": ["...","..."],
  "deleted_belief_ids": ["...","..."]
}
`, string(oldBeliefsJSON), string(newBeliefsJSON))

	// STEP 4: Call OpenAI
	response, err := aih.client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: string(GPT_LATEST),
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: systemInstruction},
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		log.Printf("Error from AI: %v", err)
		return nil, nil, err
	}

	// STEP 5: Extract the raw text returned by ChatGPT.
	aiContent := response.Choices[0].Message.Content
	log.Printf("AI response: %s", aiContent)

	// STEP 6: Unmarshal the JSON that ChatGPT returns.
	var parsedResponse struct {
		KeptBeliefIDs    []string `json:"kept_belief_ids"`
		DeletedBeliefIDs []string `json:"deleted_belief_ids"`
	}
	if err := json.Unmarshal([]byte(aiContent), &parsedResponse); err != nil {
		return nil, nil, fmt.Errorf("failed to parse JSON from AI: %w", err)
	}

	// Return the two lists: kept and deleted.
	return parsedResponse.KeptBeliefIDs, parsedResponse.DeletedBeliefIDs, nil
}

func (aih *AIHelper) UpdateBeliefWithInteractionEvent(event InteractionEvent, existingBeliefStr string) (bool, string, error) {
	eventJson, err := json.Marshal(event)
	if err != nil {
		return false, "", err
	}

	response, err := aih.client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: string(GPT_LATEST),
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: "Determine whether a user interaction and an existing belief have any relevance to each other or not."},
			{Role: "user", Content: fmt.Sprintf("Curtly respond with 'yes' or 'no' if %s has a meaningful relevance to %s", eventJson, existingBeliefStr)},
		},
	})
	if err != nil {
		return false, "", err
	}

	if response.Choices[0].Message.Content == "no" {
		return false, "", nil
	}

	response, err = aih.client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: string(GPT_LATEST),
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: fmt.Sprintf("Given these definitions %s. Construct a belief that underlies the information present in the user event", DIALECTICAL_STRATEGY)},
			{Role: "user", Content: fmt.Sprintf("Given the existing belief, %s, provide a curt summary of the new updated belief given the user interaction, %s", existingBeliefStr, eventJson)},
		},
	})
	if err != nil {
		return false, "", err
	}

	return true, response.Choices[0].Message.Content, nil
}

type DialecticStrategy int

const (
	StrategyDefault DialecticStrategy = iota
	StrategySleepDietExercise
	// Add more strategies as needed
)

func (h *AIHelper) GenerateAnalysisForStrategy(strategy DialecticStrategy, beliefSystem *models.BeliefSystem, userInteractions []models.DialecticalInteraction, interactionEvent InteractionEvent) (*models.BeliefAnalysis, error) {
	switch strategy {
	case StrategySleepDietExercise:
		return h.generateSleepDietExerciseAnalysis(beliefSystem, userInteractions, interactionEvent)
	default:
		return h.generateDefaultAnalysis(beliefSystem, userInteractions, interactionEvent)
	}
}

func (h *AIHelper) generateSleepDietExerciseAnalysis(beliefSystem *models.BeliefSystem, userInteractions []models.DialecticalInteraction, interactionEvent InteractionEvent) (*models.BeliefAnalysis, error) {
	systemPrompt := fmt.Sprintf(`Analyze the following belief system related to sleep, diet, and exercise:
%s

Consider the latest interaction:
Question: %s
Answer: %s

Provide an analysis focusing on:
1. Coherence of beliefs related to sleep, diet, and exercise
2. Consistency of beliefs with established health principles
3. Falsifiability of the beliefs
4. Overall understanding of the relationship between sleep, diet, exercise, metabolism, and energy

Respond ONLY with a JSON object in the following structure:
{
  "coherence": float,
  "consistency": float,
  "falsifiability": float,
  "overallScore": float,
  "feedback": string,
  "recommendations": [string],
  "verifiedBeliefs": [string]
}`, beliefSystemToString(beliefSystem), interactionEvent.Question, interactionEvent.Answer)

	response, err := h.getCompletionFromAI(systemPrompt)
	if err != nil {
		return nil, err
	}

	// Log the full response for debugging
	log.Printf("AI Response: %s", response)

	// Try to extract JSON from the response
	jsonStr := extractJSON(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("no valid JSON found in the response")
	}

	var analysis models.BeliefAnalysis
	err = json.Unmarshal([]byte(jsonStr), &analysis)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	return &analysis, nil
}

func extractJSON(s string) string {
	// Find the first occurrence of '{' and the last occurrence of '}'
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")

	if start == -1 || end == -1 || end <= start {
		return ""
	}

	// Extract the substring between '{' and '}'
	jsonCandidate := s[start : end+1]

	// Use a regular expression to validate the JSON structure
	re := regexp.MustCompile(`^\s*\{[\s\S]*\}\s*$`)
	if !re.MatchString(jsonCandidate) {
		return ""
	}

	return jsonCandidate
}

func (h *AIHelper) generateDefaultAnalysis(beliefSystem *models.BeliefSystem, userInteractions []models.DialecticalInteraction, interactionEvent InteractionEvent) (*models.BeliefAnalysis, error) {
	// Implementation similar to generateSleepDietExerciseAnalysis, but with a more general focus
	// ... (implement this method)
	return nil, nil
}

func (h *AIHelper) getCompletionFromAI(systemPrompt string) (string, error) {
	response, err := h.client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: string(GPT_LATEST),
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: "Please respond with the analysis in the specified JSON format."},
		},
	})
	if err != nil {
		return "", err
	}

	return response.Choices[0].Message.Content, nil
}

func beliefSystemToString(bs *models.BeliefSystem) string {
	// Convert BeliefSystem to a string representation
	// ... (implement this method)
	return ""
}

func (h *AIHelper) PredictAnswer(question string) (string, error) {
	prompt := fmt.Sprintf(`Given the question: "%s"
	Based on typical human responses and common belief systems,
	predict a likely answer to this question.
	Provide only the predicted answer, no explanation.`, question)

	return h.CompletePrompt(prompt)
}

// CompletePrompt sends a prompt to the AI model and returns the completion
func (h *AIHelper) CompletePrompt(prompt string) (string, error) {
	if h.client == nil {
		return "", fmt.Errorf("AI client is not initialized")
	}

	resp, err := h.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: string(GPT_LATEST),
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    "system",
					Content: "You are a helpful assistant.",
				},
				{
					Role:    "user",
					Content: prompt,
				},
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to complete prompt: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no completion choices returned")
	}

	return resp.Choices[0].Message.Content, nil
}

func (h *AIHelper) IsAnswerToQuestion(question, potentialAnswer string) (bool, error) {
	prompt := fmt.Sprintf(`Given this question: "%s"
	
Is this a direct answer to the question: "%s"

Reply with only "true" if it answers the question, or "false" if it does not.`,
		question, potentialAnswer)

	response, err := h.CompletePrompt(prompt)
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(strings.ToLower(response)) == "true", nil
}

func (h *AIHelper) ExtractQuestionsFromText(text string) ([]string, error) {
	prompt := fmt.Sprintf(`Extract all distinct questions from this text. Return only the questions, one per line, without any numbering or bullets:

Text: %s`, text)

	response, err := h.CompletePrompt(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to extract questions: %w", err)
	}

	// Split response into lines and clean up
	var questions []string
	for _, line := range strings.Split(strings.TrimSpace(response), "\n") {
		line = strings.TrimSpace(line)
		// Only include non-empty lines that end with a question mark
		if line != "" && strings.HasSuffix(line, "?") {
			questions = append(questions, line)
		}
	}

	if len(questions) == 0 {
		return nil, fmt.Errorf("no valid questions found in text")
	}

	return questions, nil
}

func (h *AIHelper) MatchAnswersToQuestions(answerBlob string, questions []string) ([]string, error) {
	prompt := fmt.Sprintf(`Given this text that may contain answers: "%s"

And these questions:
%s

For each question, extract the relevant answer from the text.
If no answer is found for a question, return an empty string.
Return answers in the same order as questions, one per line.`,
		answerBlob, strings.Join(questions, "\n"))

	response, err := h.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: string(GPT_LATEST),
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    "system",
					Content: prompt,
				},
				{
					Role:    "user",
					Content: "Please extract the answers.",
				},
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to complete prompt: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no completion choices returned")
	}

	// Get the message content from the response
	content := response.Choices[0].Message.Content

	// Split response into lines and clean up
	answers := strings.Split(strings.TrimSpace(content), "\n")
	for i := range answers {
		answers[i] = strings.TrimSpace(answers[i])
	}

	// Pad with empty strings if needed
	for len(answers) < len(questions) {
		answers = append(answers, "")
	}

	return answers[:len(questions)], nil
}
