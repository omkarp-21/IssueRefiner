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

func groundedResult() *InvestigationResult {
	return &InvestigationResult{
		Summary:            "Pool exhaustion",
		Readiness:          ReadinessReady,
		MissingInformation: []string{},
		Signals:            []string{"pq: too many clients"},
		Hypotheses: []Hypothesis{
			{Statement: "Pool undersized", Confidence: ConfidenceMedium, Reasoning: "error under load", Evidence: []string{"issue", "comment:123"}},
		},
	}
}

func TestValidateGroundingValid(t *testing.T) {
	valid := map[string]bool{"issue": true, "comment:123": true}
	if err := ValidateGrounding(groundedResult(), valid); err != nil {
		t.Fatalf("expected grounded result to pass, got %v", err)
	}
}

func TestValidateGroundingInvalid(t *testing.T) {
	valid := map[string]bool{"issue": true, "comment:123": true}
	cases := []struct {
		name     string
		evidence []string
	}{
		{"unknown comment", []string{"comment:99999"}},
		{"fabricated file", []string{"file:core/mcp.py:123"}},
		{"fabricated log", []string{"log:production"}},
		{"other source", []string{"github:some-other-source"}},
		{"free-form text", []string{"the database was overloaded"}},
		{"old signal phrase", []string{"pq: too many clients"}},
		{"empty", []string{""}},
		{"malformed prefix", []string{"comment:"}},
		{"malformed id", []string{"comment:abc"}},
		{"malformed no colon", []string{"comment"}},
	}
	for _, tc := range cases {
		result := groundedResult()
		result.Hypotheses[0].Evidence = tc.evidence
		if err := ValidateGrounding(result, valid); !errors.Is(err, ErrInvalidResult) {
			t.Fatalf("%s: expected ErrInvalidResult, got %v", tc.name, err)
		}
	}
}

func TestValidateGroundingMultipleHypotheses(t *testing.T) {
	result := groundedResult()
	result.Hypotheses = append(result.Hypotheses, Hypothesis{
		Statement: "Second", Confidence: ConfidenceLow, Reasoning: "r", Evidence: []string{"comment:456"},
	})
	valid := map[string]bool{"issue": true, "comment:123": true, "comment:456": true}
	if err := ValidateGrounding(result, valid); err != nil {
		t.Fatalf("expected different evidence per hypothesis to pass, got %v", err)
	}
}

func TestValidateGroundingNeedsInfo(t *testing.T) {
	result := &InvestigationResult{
		Summary:            "Vague",
		Readiness:          ReadinessNeedsInfo,
		MissingInformation: []string{"reproduction steps"},
		Signals:            []string{},
		Hypotheses:         []Hypothesis{},
	}
	if err := ValidateGrounding(result, map[string]bool{"issue": true}); err != nil {
		t.Fatalf("expected NEEDS_INFORMATION without hypotheses to pass, got %v", err)
	}
}

func TestEvidenceIDs(t *testing.T) {
	rendered := BuildBody("body", []EvidenceItem{
		{ID: 111, Source: SourceComment, Author: "alice", Body: "one"},
		{ID: 111, Source: SourceComment, Author: "alice", Body: "duplicate"},
		{ID: 222, Source: SourceComment, Author: "bob", Body: "two"},
	})
	ids := EvidenceIDs(rendered)
	if !ids["issue"] || !ids["comment:111"] || !ids["comment:222"] {
		t.Fatalf("expected issue + both comment IDs, got %v", ids)
	}
	if ids["comment:999"] {
		t.Fatalf("dropped/unknown ID must not validate: %v", ids)
	}
	if got := len(ids); got != 3 {
		t.Fatalf("expected deduped set of 3, got %v", ids)
	}
}

func TestEvidenceIDsNoComments(t *testing.T) {
	ids := EvidenceIDs("just the issue")
	if !ids["issue"] || len(ids) != 1 {
		t.Fatalf("expected only issue, got %v", ids)
	}
}

func TestParseRefinedFields(t *testing.T) {
	raw := `{"summary":"Crash on login","readiness":"READY","missing_information":[],"signals":["500 on POST /login"],"observed_behavior":"Server returns 500","expected_behavior":"Server returns 200","reproduction_steps":["open login page","submit form"],"environment":"Go 1.22, Postgres 17","hypotheses":[]}`
	result, err := ParseResult(raw)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.ObservedBehavior != "Server returns 500" {
		t.Fatalf("unexpected observed_behavior %q", result.ObservedBehavior)
	}
	if result.ExpectedBehavior != "Server returns 200" {
		t.Fatalf("unexpected expected_behavior %q", result.ExpectedBehavior)
	}
	if len(result.ReproductionSteps) != 2 || result.ReproductionSteps[0] != "open login page" {
		t.Fatalf("unexpected reproduction_steps %v", result.ReproductionSteps)
	}
	if result.Environment != "Go 1.22, Postgres 17" {
		t.Fatalf("unexpected environment %q", result.Environment)
	}
}

func TestParseLegacyWithoutRefinedFields(t *testing.T) {
	result, err := ParseResult(strongJSON)
	if err != nil {
		t.Fatalf("expected legacy result to parse, got %v", err)
	}
	if result.ObservedBehavior != "" || result.ExpectedBehavior != "" || result.Environment != "" {
		t.Fatalf("expected empty refined strings, got %+v", result)
	}
	if result.ReproductionSteps == nil || len(result.ReproductionSteps) != 0 {
		t.Fatalf("expected normalized empty reproduction_steps, got %v", result.ReproductionSteps)
	}
}
