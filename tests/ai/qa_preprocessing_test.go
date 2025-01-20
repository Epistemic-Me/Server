package ai_tests

import (
	"os"
	"strings"
	"testing"

	ai "epistemic-me-core/ai"

	"github.com/stretchr/testify/require"
)

func TestMatchAnswersToQuestions(t *testing.T) {
	skipIfNoAPIKey(t)
	helper := ai.NewAIHelper(getTestAPIKey())

	tests := []struct {
		name      string
		answers   string
		questions []string
		validate  func(t *testing.T, answers []string)
	}{
		{
			name: "handles numbered answers",
			answers: `1. I sleep 8 hours
2. I exercise daily
3. I eat healthy food`,
			questions: []string{
				"How much do you sleep?",
				"What's your exercise routine?",
				"What's your diet like?",
			},
			validate: func(t *testing.T, answers []string) {
				require.Len(t, answers, 3)
				require.Equal(t, "I sleep 8 hours", answers[0], "Should remove number prefix")
				require.Equal(t, "I exercise daily", answers[1], "Should remove number prefix")
				require.Equal(t, "I eat healthy food", answers[2], "Should remove number prefix")
			},
		},
		{
			name: "handles missing answers",
			answers: `I sleep 8 hours.
I exercise daily.`,
			questions: []string{
				"How much do you sleep?",
				"What's your exercise routine?",
				"What's your diet like?",
			},
			validate: func(t *testing.T, answers []string) {
				require.Len(t, answers, 3)
				require.NotEqual(t, "", answers[2], "Should use 'No answer provided' for missing answers")
				require.Equal(t, "No answer provided", answers[2])
			},
		},
		{
			name: "maintains original text wording",
			answers: `My sleep routine includes taking supplements and using a white noise machine.
I exercise 5-6 times per week with a mix of cardio and weights.`,
			questions: []string{
				"What's your sleep routine?",
				"How often do you exercise?",
			},
			validate: func(t *testing.T, answers []string) {
				require.Len(t, answers, 2)
				require.Contains(t, answers[0], "supplements")
				require.Contains(t, answers[0], "white noise machine")
				require.Contains(t, answers[1], "5-6 times per week")
			},
		},
		{
			name: "avoids cross-referencing answers",
			answers: `I sleep 8 hours and exercise daily.
My exercise routine includes running and weights.`,
			questions: []string{
				"How much do you sleep?",
				"What's your exercise routine?",
			},
			validate: func(t *testing.T, answers []string) {
				require.Len(t, answers, 2)
				require.Contains(t, answers[0], "8 hours")
				require.NotContains(t, answers[0], "running")
				require.Contains(t, answers[1], "running")
				require.Contains(t, answers[1], "weights")
			},
		},
		{
			name: "handles multi-paragraph answers",
			answers: `I get 8 hours of sleep each night. My bedtime routine is to wind-down, 
take my sleep supplements, and then set up my sleep environment. This includes a white 
noise machine, sleep mask, weighted blanket and a cooling mattress.

I believe that if I sleep well each night, and consistently from night to night, 
it will help me manage my energy through circadian rhythm. I also believe that good 
sleep helps manage my nervous system as measured by my morning HRV.`,
			questions: []string{
				"How many hours do you sleep?",
				"What's your bedtime routine?",
				"What are your beliefs about sleep?",
			},
			validate: func(t *testing.T, answers []string) {
				require.Len(t, answers, 3)
				require.Contains(t, answers[0], "8 hours")
				require.Contains(t, answers[1], "wind-down")
				require.Contains(t, answers[1], "sleep supplements")
				require.Contains(t, answers[2], "circadian rhythm")
				require.Contains(t, answers[2], "nervous system")
			},
		},
		{
			name: "handles semantic matching",
			answers: `I start my day with a protein shake and take supplements.
I track my biomarkers and adjust my diet based on blood test results.
I focus on getting diverse nutrients from natural food sources.`,
			questions: []string{
				"What's your breakfast routine?",
				"How do you monitor your health?",
				"What guides your food choices?",
				"Do you eat processed foods?", // Implicit answer
			},
			validate: func(t *testing.T, answers []string) {
				require.Len(t, answers, 4)
				require.Contains(t, answers[0], "protein shake")
				require.Contains(t, answers[1], "biomarkers")
				require.Contains(t, answers[1], "blood test")
				require.Contains(t, answers[2], "diverse nutrients")
				require.Contains(t, answers[2], "natural food")
				// Should infer from "natural food sources" that processed foods are avoided
				require.Contains(t, answers[3], "natural")
			},
		},
		{
			name: "combines related information",
			answers: `I exercise 5-6 times a week.
Later in the text: I do cycling, running, and weights.
And even later: My workouts usually last about an hour.`,
			questions: []string{
				"Describe your exercise routine.",
			},
			validate: func(t *testing.T, answers []string) {
				require.Len(t, answers, 1)
				answer := answers[0]
				require.Contains(t, answer, "5-6 times")
				require.Contains(t, answer, "cycling")
				require.Contains(t, answer, "running")
				require.Contains(t, answer, "weights")
				require.Contains(t, answer, "hour")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			answers, err := helper.MatchAnswersToQuestions(tt.answers, tt.questions)
			require.NoError(t, err)
			tt.validate(t, answers)
		})
	}
}

func TestExtractAndCleanQuestions(t *testing.T) {
	helper := ai.NewAIHelper(getTestAPIKey())

	input := `### Step 1: Sleep Habits
- **How many hours do you sleep?**
- What's your bedtime routine?
Some other text...
• Do you have any sleep beliefs?`

	cleaned := helper.ExtractAndCleanQuestions(input)
	questions := strings.Split(cleaned, "\n")

	require.Len(t, questions, 3)
	require.Equal(t, "How many hours do you sleep?", questions[0])
	require.Equal(t, "What's your bedtime routine?", questions[1])
	require.Equal(t, "Do you have any sleep beliefs?", questions[2])
}

func TestCleanAnswerText(t *testing.T) {
	helper := ai.NewAIHelper(getTestAPIKey())

	input := "I sleep 8hrs per night, **use supplements**, and have a `good routine`,\n"
	cleaned := helper.CleanAnswerText(input)

	require.Equal(t, "I sleep 8 hrs per night, use supplements, and have a good routine", cleaned)
}

func getTestAPIKey() string {
	return os.Getenv("OPENAI_API_KEY")
}
