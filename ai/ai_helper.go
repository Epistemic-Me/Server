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

func (aih *AIHelper) GetInteractionEventAsBelief(event InteractionEvent) ([]string, error) {
	eventJson, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}

	response, err := aih.client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: string(GPT_LATEST),
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: fmt.Sprintf(`Given these definitions %s. 
				Extract all beliefs from the user's response. 
				Return ONLY a JSON object with a "beliefs" array containing all belief statements.
				Example: {"beliefs": [
					"I believe that quality sleep is essential for energy",
					"I believe that maintaining a consistent sleep schedule improves sleep quality"
				]}`, DIALECTICAL_STRATEGY)},
			{Role: "user", Content: fmt.Sprintf("Extract beliefs from this interaction: %s", eventJson)},
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

func (aih *AIHelper) ExtractBeliefsFromResource(resource models.Resource) ([]string, error) {

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

	// Extract JSON from the response
	jsonStr := extractJSON(response.Choices[0].Message.Content)
	if jsonStr == "" {
		return nil, fmt.Errorf("failed to extract JSON from response")
	}

	// Parse the JSON response
	var beliefResponse struct {
		Beliefs []string `json:"beliefs"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &beliefResponse); err != nil {
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

Return ONLY a JSON object with two arrays of belief IDs:
{
  "kept_belief_ids": ["...","..."],
  "deleted_belief_ids": ["...","..."]
}

Important:
 - "kept_belief_ids" should be the IDs of old beliefs that remain valid.
 - "deleted_belief_ids" should be the IDs of old beliefs that are no longer valid.
 - Do NOT include the new beliefs in the returned IDs.
 - Do NOT include any markdown formatting or backticks in your response.
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

Please return ONLY a JSON object with kept_belief_ids and deleted_belief_ids arrays.
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
	log.Printf("Old Beliefs: %s", oldBeliefsJSON)
	log.Printf("AI response: %s", aiContent)

	// Extract JSON from the response if needed
	jsonStr := extractJSON(aiContent)
	if jsonStr == "" {
		return nil, nil, fmt.Errorf("no valid JSON found in response")
	}

	// STEP 6: Unmarshal the JSON that ChatGPT returns.
	var parsedResponse struct {
		KeptBeliefIDs    []string `json:"kept_belief_ids"`
		DeletedBeliefIDs []string `json:"deleted_belief_ids"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &parsedResponse); err != nil {
		return nil, nil, fmt.Errorf("failed to parse JSON from AI: %w", err)
	}

	// Return the two lists: kept and deleted.
	return parsedResponse.KeptBeliefIDs, parsedResponse.DeletedBeliefIDs, nil
}

// ProvidePerspectiveOnQuestionAndAnswer generates
// a perspective on how a specified belief system would interpret the
// question and the provided answer.
func (aih *AIHelper) ProvidePerspectiveOnQuestionAndAnswer(
	question, answer, beliefSystem string,
) (string, error) {

	// Create a prompt that asks for a concise perspective from the specified belief system
	perspectivePrompt := fmt.Sprintf(
		"From the perspective of the belief system '%s', how would this belief system interpret the question '%s' and the answer '%s'? Provide a concise viewpoint.",
		beliefSystem, question, answer,
	)

	// Make a single API call to retrieve the perspective
	perspectiveResponse, err := aih.client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: string(GPT_LATEST),
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "Provide a concise perspective on how the specified belief system would interpret the given question and answer.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: perspectivePrompt,
			},
		},
	})
	if err != nil {
		return "", err
	}

	return perspectiveResponse.Choices[0].Message.Content, nil
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
	prompt := fmt.Sprintf(`Given this text containing answers:
"%s"

And these questions:
%s

Task: Extract ONE DISTINCT answer for EACH question from the text.

Rules for Semantic Matching:
1. Each question MUST have its own separate answer
2. Do not combine answers for different questions
3. Look for answers across the ENTIRE text
4. Match answers that address the question's intent
5. If multiple answers exist, use the most specific one
6. Each answer should be self-contained
7. Do not include answers to other questions
8. If no relevant answer exists, return "No answer provided"

Example Output Format:
Q1: "How do you make your food choices?"
A1: "My food choices are based on nutrition, hunger and enjoyment of food. I manage energy, and biomarkers in my blood to make healthy food choices."

Q2: "What do you eat for breakfast?"
A2: "I start my day with a shake that contains MCT oil powder for energy."

Begin extracting answers, one per question:`,
		answerBlob, strings.Join(questions, "\n"))

	response, err := h.CompletePrompt(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to match answers: %w", err)
	}

	// Parse Q&A format response
	var answers []string
	lines := strings.Split(strings.TrimSpace(response), "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		// Skip question lines (starting with Q)
		if strings.HasPrefix(line, "Q") {
			continue
		}
		// Process answer lines (starting with A)
		if strings.HasPrefix(line, "A") {
			// Extract just the answer part (after the colon)
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				answer := strings.TrimSpace(parts[1])
				// Remove surrounding quotes if present
				answer = strings.Trim(answer, `"`)
				answers = append(answers, answer)
			}
		}
	}

	// Ensure we have an answer for each question
	for len(answers) < len(questions) {
		answers = append(answers, "No answer provided")
	}

	return answers[:len(questions)], nil
}

func (h *AIHelper) ExtractAndCleanQuestions(content string) []string {
	// Extract actual questions (lines ending with question marks)
	var questions []string
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "?") {
			// Remove markdown formatting
			line = regexp.MustCompile("`[^`]*`").ReplaceAllString(line, "")
			// Remove bold markers
			line = regexp.MustCompile(`\*\*`).ReplaceAllString(line, "")
			// Remove bullet points and numbering
			line = regexp.MustCompile(`^[-*•]|\d+\.\s*`).ReplaceAllString(line, "")
			// Clean up whitespace
			line = strings.TrimSpace(line)
			// Extract the question part if it's embedded in other text
			if idx := strings.Index(line, "?"); idx >= 0 {
				question := strings.TrimSpace(line[:idx+1])
				if question != "" {
					questions = append(questions, question)
				}
			}
		}
	}

	return questions
}

func (h *AIHelper) CleanAnswerText(content string) string {
	// Remove markdown formatting
	content = regexp.MustCompile("`[^`]*`").ReplaceAllString(content, "")

	// Add space after numbers followed immediately by letters
	content = regexp.MustCompile(`(\d+)([a-zA-Z])`).ReplaceAllString(content, "$1 $2")

	// Remove trailing commas and clean up whitespace
	content = strings.TrimSpace(content)
	content = strings.TrimSuffix(content, ",")

	return content
}

func (h *AIHelper) GenerateQuestionForLearningObjective(objective *models.LearningObjective, interactions []models.DialecticalInteraction) (string, error) {
	// For the initial question (no interactions yet), start with a general question about all topics
	if len(interactions) == 0 {
		prompt := fmt.Sprintf(`Given a learning objective to understand beliefs about: %s
Generate an initial question that covers multiple topics (%s).
The question should encourage detailed responses about personal beliefs and experiences.
Return only the question, with no additional text.`,
			objective.Description,
			strings.Join(objective.Topics, ", "))

		return h.CompletePrompt(prompt)
	}

	// For subsequent questions, analyze topic coverage
	var result struct {
		TopicCoverage map[string]struct {
			Percentage float32 `json:"percentage"`
		} `json:"topic_coverage"`
	}

	// Get current completion analysis to determine which topic needs attention
	completion, err := h.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4,
			Messages: []openai.ChatCompletionMessage{
				{Role: "system", Content: `You are a JSON-only response bot. Return EXACTLY this JSON structure with no other text:
{
    "topic_coverage": {
        "topic1": {"percentage": 50.0},
        "topic2": {"percentage": 75.0}
    }
}
Replace topic1, topic2 with the actual topics from the input, and calculate real coverage percentages based on the beliefs.`},
				{Role: "user", Content: fmt.Sprintf(`Analyze these topics: %v

For each topic, calculate a coverage percentage (0-100) based on these beliefs:
%s

Return the coverage percentages in the specified JSON format.`,
					strings.Join(objective.Topics, ", "),
					formatInteractionBeliefs(interactions))},
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to analyze topic coverage: %w", err)
	}

	// Extract JSON from response if needed
	responseContent := completion.Choices[0].Message.Content
	jsonStr := extractJSON(responseContent)
	if jsonStr == "" {
		return "", fmt.Errorf("no valid JSON found in response: %s", responseContent)
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return "", fmt.Errorf("failed to parse topic coverage: %w", err)
	}

	// Find the topic with lowest coverage
	var lowestTopic string
	var lowestCoverage float32 = 100.0
	for topic, coverage := range result.TopicCoverage {
		if coverage.Percentage < lowestCoverage {
			lowestTopic = topic
			lowestCoverage = coverage.Percentage
		}
	}

	// Generate a focused question for the topic with lowest coverage
	prompt := fmt.Sprintf(`Given a learning objective to understand beliefs about: %s
We are currently focusing on the topic: %s (current coverage: %.1f%%)

Generate a focused question to gather more detailed beliefs about %s.
If previous questions were general, ask about specific aspects or habits.
If previous questions covered basics, ask about influences and experiences.

The question should encourage detailed responses about personal beliefs and experiences.
Return only the question, with no additional text.`,
		objective.Description,
		lowestTopic,
		lowestCoverage,
		lowestTopic)

	return h.CompletePrompt(prompt)
}

// Helper function to format beliefs from interactions
func formatInteractionBeliefs(interactions []models.DialecticalInteraction) string {
	var beliefs []string
	for _, interaction := range interactions {
		if interaction.Status == models.StatusAnswered && interaction.Interaction != nil &&
			interaction.Interaction.QuestionAnswer != nil && len(interaction.Interaction.QuestionAnswer.ExtractedBeliefs) > 0 {
			for _, belief := range interaction.Interaction.QuestionAnswer.ExtractedBeliefs {
				beliefs = append(beliefs, belief.GetContentAsString())
			}
		}
	}
	return strings.Join(beliefs, "\n")
}

// CheckLearningObjectiveCompletion determines how complete our learning objective is based on collected beliefs
func (h *AIHelper) CheckLearningObjectiveCompletion(lo *models.LearningObjective, selfModel *models.SelfModel) (float32, error) {
	// Extract all beliefs from the current belief system
	var beliefs []string
	for _, belief := range selfModel.BeliefSystem.Beliefs {
		beliefs = append(beliefs, belief.GetContentAsString())
	}

	log.Printf("Total beliefs in belief system: %d", len(beliefs))
	log.Printf("Topics to cover: %v", lo.Topics)

	// Prepare the system prompt for analyzing current belief system
	systemPrompt := `You are an AI assistant helping to determine the completion percentage of a learning objective focused on belief statements.

For each topic, analyze the current belief system in these categories:
1. Foundational Beliefs (25%)
   - Basic understanding of the topic
   - Core principles and values
2. Practice-Based Beliefs (25%)
   - Specific habits and routines
   - Personal methods and approaches
3. Cause-Effect Beliefs (25%)
   - Understanding of impacts and consequences
   - Connections between actions and results
4. Experience-Based Beliefs (25%)
   - Personal experiences and observations
   - Learned insights and adaptations

Calculate topic coverage based on:
1. Presence - Each category above contributes 25% to the topic's completion
2. Quality - For each category, evaluate:
   - Clear statement of belief ("I believe...") (40%)
   - Supporting details or examples (30%)
   - Personal context or reasoning (30%)
3. Progression - The overall percentage should:
   - Start at 25% when foundational beliefs are present
   - Increase as beliefs become more detailed and personal
   - Reach 100% when all categories have quality beliefs

Important rules for analysis:
1. Consider the entire belief system as a whole
2. Each topic's coverage should reflect the depth and breadth of beliefs about that topic
3. The completion percentage represents how well the current belief system covers the learning objective

You MUST respond with a valid JSON object in EXACTLY this format:
{
    "completion_percentage": 45.5,
    "topic_coverage": {
        "sleep": {
            "percentage": 80.0,
            "covered_categories": ["foundational", "practice-based"],
            "missing_categories": ["cause-effect", "experience-based"],
            "belief_quality": {
                "foundational": 0.9,
                "practice_based": 0.8,
                "cause_effect": 0.0,
                "experience_based": 0.0
            }
        }
    },
    "explanation": "Brief explanation of the score"
}

Do not include any other text before or after the JSON object.`

	// Prepare the user message with the learning objective and current belief system
	userMsg := fmt.Sprintf(`Learning Objective: %s
Topics to explore: %v

Current Belief System:
%s`, lo.Description, lo.Topics, strings.Join(beliefs, "\n"))

	// Get completion analysis from OpenAI
	completion, err := h.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4,
			Messages: []openai.ChatCompletionMessage{
				{Role: "system", Content: systemPrompt},
				{Role: "user", Content: userMsg},
			},
		},
	)
	if err != nil {
		return 0, fmt.Errorf("failed to get completion analysis: %w", err)
	}

	// Parse the response
	var result struct {
		CompletionPercentage float32 `json:"completion_percentage"`
		TopicCoverage        map[string]struct {
			Percentage        float32            `json:"percentage"`
			CoveredCategories []string           `json:"covered_categories"`
			MissingCategories []string           `json:"missing_categories"`
			BeliefQuality     map[string]float32 `json:"belief_quality"`
		} `json:"topic_coverage"`
		Explanation string `json:"explanation"`
	}

	if err := json.Unmarshal([]byte(completion.Choices[0].Message.Content), &result); err != nil {
		return 0, fmt.Errorf("failed to parse completion analysis: %w", err)
	}

	// Log detailed analysis for each topic
	log.Printf("\n=== Learning Objective Completion Analysis ===")
	log.Printf("Overall Completion: %.1f%%", result.CompletionPercentage)
	log.Printf("Explanation: %s\n", result.Explanation)

	for topic, coverage := range result.TopicCoverage {
		log.Printf("\nTopic: %s", topic)
		log.Printf("Coverage: %.1f%%", coverage.Percentage)
		log.Printf("Covered Categories: %v", coverage.CoveredCategories)
		log.Printf("Missing Categories: %v", coverage.MissingCategories)
		log.Printf("Belief Quality by Category:")
		for category, quality := range coverage.BeliefQuality {
			log.Printf("  - %s: %.2f", category, quality)
		}
	}
	log.Printf("\n=== End Analysis ===\n")

	return result.CompletionPercentage, nil
}

// GenerateAnswerFromBeliefSystem generates an answer to a question based on the user's belief system and philosophy
func (aih *AIHelper) GenerateAnswerFromBeliefSystem(question string, beliefSystem *models.BeliefSystem, philosophies []string) (string, error) {
	// Convert beliefs to strings for the prompt
	beliefStrings := make([]string, len(beliefSystem.Beliefs))
	for i, belief := range beliefSystem.Beliefs {
		beliefStrings[i] = belief.GetContentAsString()
	}

	// Create the system prompt
	systemPrompt := fmt.Sprintf(`You are roleplaying as a user with the following belief system and philosophies:

Beliefs:
%s

Life Philosophy:
%s

When answering questions, maintain consistency with these beliefs and philosophies. 
Respond in first person ("I believe...") and be specific about your beliefs and how they influence your daily habits.
Keep responses concise but informative, focusing on your personal beliefs and experiences.`,
		strings.Join(beliefStrings, "\n"),
		strings.Join(philosophies, "\n"))

	response, err := aih.client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: string(GPT_LATEST),
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: fmt.Sprintf("Please answer this question about your beliefs: %s", question)},
		},
	})
	if err != nil {
		return "", err
	}

	return response.Choices[0].Message.Content, nil
}
