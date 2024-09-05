package ai_helper

import (
	"context"
	"encoding/json"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

const DIALECTICAL_STRATEGY = `What is a Question?
In the BC dialectic context, a question creates a conceptual framework for learning. It acts as a container for prior beliefs, evidence collection, and answer verification.

It is an inference of the world based on prior beliefs that guides evidence collection and observation. A good question can shape knowledge that is valuable to our future experiences. The question provides the boundaries of an unambiguous explanation for a causal pattern.

If we ask the question, “What is a healthy day?” The first phase of our inquiry explores whether our prior beliefs create sufficient knowledge to predict that our next day can be healthy.

Let's assume that our epistemic emotions find a gap in the predictability of our prior beliefs and guide curiosity. Now, the question will guide us to collect more evidence to either justify our beliefs more deeply or update them. In both cases, our curiosity guides a more accurate prediction of how we will make tomorrow a healthy day.

In seeking evidence, we can look to the experiences of our ancestors or to the experiences of our contemporaries. In the former, we consider the synthesis of all people who have lived a healthy life and the “best practices” synthesized through our research and education systems. In the latter, we consider the direct experiences of others who share our beliefs.

Ultimately, we’re seeking evidence that is valuable to our learning process to answer the question.
Who has beliefs that generate a predictive answer to this question?
What evidence justifies their beliefs?
Where are the beliefs predictive? (ie, in what observation context)
Why does evidence justify their beliefs?
When in their history have they applied these beliefs predictively?
How does the evidence justify their beliefs?

What is an Answer?
An answer is an explanatory narrative that describes a causal pattern within a conceptual framework. Specifically, the answer is consistent with the possibilities of the belief systems (generative) and probabilities of value systems (discriminant) that contextualize an inquiry.

What is a Belief?
Prior beliefs are stated as an expectation for an observable causal change; each belief carries a level of certainty. For instance, “I believe that quality sleep is required for high energy the following day” (ie, Sleep Belief). This belief might carry a 90% certainty of being predictive.

When beliefs are evidenced and certain above a certain threshold, they create knowledge. Beliefs can be evidenced by experience or by theory.

The user’s Sleep Belief could be evidenced by their own experience with quality sleep and energy, the experience of others, theories, research, etc. Epistemologically, the user is seeking evidence that a belief is predictive.

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
			{Role: "system", Content: fmt.Sprintf("Given these definitions %s. Construct a belief that underlies the information present in the user event", DIALECTICAL_STRATEGY)},
			{Role: "user", Content: fmt.Sprintf("Please just respond curtly with the belief summary, %s", eventJson)},
		},
	})
	if err != nil {
		return "", err
	}

	return response.Choices[0].Message.Content, nil
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
