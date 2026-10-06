package investigation

import (
	"strings"
	"testing"
)

func TestSystemPromptRules(t *testing.T) {
	prompt := SystemPrompt()
	for _, want := range []string{"Do NOT invent", "NEEDS_INFORMATION", "READY", "HIGH", "missing_information", "hypotheses", "evidence"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("system prompt missing %q", want)
		}
	}
}

func TestUserPromptContainsIssue(t *testing.T) {
	prompt := UserPrompt("octo", "demo", 421, "Pool exhausted", "traceback here")
	for _, want := range []string{"octo/demo", "#421", "Pool exhausted", "traceback here"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("user prompt missing %q: %q", want, prompt)
		}
	}
}
