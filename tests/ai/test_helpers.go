package ai_tests

import (
	"os"
	"testing"
)

func skipIfNoAPIKey(t *testing.T) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("Skipping test: OPENAI_API_KEY not set")
	}
}
