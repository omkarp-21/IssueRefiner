package investigation

import (
	"errors"
	"strings"
	"testing"
)

const strongJSON = `{
  "summary": "Connection pool exhaustion under concurrent load",
  "readiness": "READY",
  "missing_information": ["exact pool size configuration"],
  "signals": ["pq: too many clients already", "POST /api/refine", "pool size 5", "Go 1.22"],
  "hypotheses": [
    {"statement": "Pool is undersized for peak concurrency", "confidence": "MEDIUM", "reasoning": "Error appears only under load per reporter", "evidence": ["pq: too many clients already", "pool size 5"]},
    {"statement": "Connections leak on failed requests", "confidence": "LOW", "reasoning": "No leak evidence in issue; alternative explanation", "evidence": ["pool size 5"]}
  ]
}`

const weakJSON = `{
  "summary": "Something is broken",
  "readiness": "NEEDS_INFORMATION",
  "missing_information": ["reproduction steps", "expected behavior", "environment and version"],
  "signals": ["login page"],
  "hypotheses": []
}`

func TestParseStrongIssue(t *testing.T) {
	result, err := ParseResult(strongJSON)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Readiness != ReadinessReady {
		t.Fatalf("expected READY, got %q", result.Readiness)
	}
	if len(result.Signals) == 0 {
		t.Fatalf("expected signals")
	}
	if len(result.Hypotheses) != 2 {
		t.Fatalf("expected 2 hypotheses, got %d", len(result.Hypotheses))
	}
	for _, hypothesis := range result.Hypotheses {
		if len(hypothesis.Evidence) == 0 {
			t.Fatalf("expected evidence on %+v", hypothesis)
		}
	}
}

func TestParseWeakIssue(t *testing.T) {
	result, err := ParseResult(weakJSON)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Readiness != ReadinessNeedsInfo {
		t.Fatalf("expected NEEDS_INFORMATION, got %q", result.Readiness)
	}
	if len(result.MissingInformation) != 3 {
		t.Fatalf("expected prioritized missing list, got %v", result.MissingInformation)
	}
	if len(result.Hypotheses) != 0 {
		t.Fatalf("expected no hypotheses, got %v", result.Hypotheses)
	}
}

func TestParseMalformed(t *testing.T) {
	for _, raw := range []string{"", "not json{{{", `{"summary":"x"}`, `{"summary":"x","readiness":"MAYBE","missing_information":[],"signals":[],"hypotheses":[]}`} {
		if _, err := ParseResult(raw); !errors.Is(err, ErrInvalidResult) {
			t.Fatalf("expected ErrInvalidResult for %q, got %v", raw, err)
		}
	}
}

func TestParseNeedsInfoRequiresMissing(t *testing.T) {
	raw := `{"summary":"vague","readiness":"NEEDS_INFORMATION","missing_information":[],"signals":[],"hypotheses":[]}`
	if _, err := ParseResult(raw); !errors.Is(err, ErrInvalidResult) {
		t.Fatalf("expected ErrInvalidResult, got %v", err)
	}
}

func TestParseHypothesisValidation(t *testing.T) {
	base := `{"summary":"s","readiness":"READY","missing_information":[],"signals":["x"],"hypotheses":[%s]}`
	cases := []string{
		`{"statement":"","confidence":"HIGH","reasoning":"r","evidence":["x"]}`,
		`{"statement":"s","confidence":"CERTAIN","reasoning":"r","evidence":["x"]}`,
		`{"statement":"s","confidence":"HIGH","reasoning":"","evidence":["x"]}`,
		`{"statement":"s","confidence":"HIGH","reasoning":"r","evidence":[]}`,
	}
	for _, hypothesis := range cases {
		raw := strings.Replace(base, "%s", hypothesis, 1)
		if _, err := ParseResult(raw); !errors.Is(err, ErrInvalidResult) {
			t.Fatalf("expected ErrInvalidResult for %s, got %v", hypothesis, err)
		}
	}
}

func TestParseStripsFences(t *testing.T) {
	fenced := "```json\n" + strongJSON + "\n```"
	result, err := ParseResult(fenced)
	if err != nil {
		t.Fatalf("expected fences tolerated, got %v", err)
	}
	if result.Readiness != ReadinessReady {
		t.Fatalf("expected READY, got %q", result.Readiness)
	}
}

func TestMarshalRoundtrip(t *testing.T) {
	result, err := ParseResult(strongJSON)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	raw, err := MarshalResult(result)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	again, err := ParseResult(raw)
	if err != nil {
		t.Fatalf("reparse failed: %v", err)
	}
	if again.Summary != result.Summary || len(again.Hypotheses) != 2 {
		t.Fatalf("roundtrip mismatch: %+v", again)
	}
}
