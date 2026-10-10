package investigation

import (
	"strings"
	"testing"
)

func TestBuildInformationRequest(t *testing.T) {
	draft := BuildInformationRequest([]string{"Exact error message", "Steps to reproduce"})
	if !strings.Contains(draft, "- Exact error message") {
		t.Fatalf("expected bullet items, got %q", draft)
	}
	if !strings.Contains(draft, "- Steps to reproduce") {
		t.Fatalf("expected bullet items, got %q", draft)
	}
	if !strings.Contains(draft, "Issue Refiner") {
		t.Fatalf("expected attribution, got %q", draft)
	}
	if strings.Contains(draft, "```") {
		t.Fatalf("draft must not use code fences, got %q", draft)
	}
}

func TestBuildInformationRequestDeterministic(t *testing.T) {
	missing := []string{"Version", "Logs"}
	if first, second := BuildInformationRequest(missing), BuildInformationRequest(missing); first != second {
		t.Fatalf("expected deterministic output")
	}
}

func TestBuildInformationRequestEmpty(t *testing.T) {
	draft := BuildInformationRequest(nil)
	if !strings.Contains(draft, "Issue Refiner") {
		t.Fatalf("expected attribution even with no items, got %q", draft)
	}
	if strings.Contains(draft, "- ") {
		t.Fatalf("expected no bullets for empty input, got %q", draft)
	}
}

func TestBuildInformationRequestSkipsBlank(t *testing.T) {
	draft := BuildInformationRequest([]string{"  ", "", "Version"})
	if strings.Count(draft, "- ") != 1 {
		t.Fatalf("expected blank items skipped, got %q", draft)
	}
}

func TestBuildInformationRequestSanitizesMentions(t *testing.T) {
	draft := BuildInformationRequest([]string{"Ask @maintainer for logs"})
	if strings.Contains(draft, "@maintainer") {
		t.Fatalf("expected @mention neutralized, got %q", draft)
	}
	if !strings.Contains(draft, "maintainer") {
		t.Fatalf("expected item text preserved, got %q", draft)
	}
}

func TestBuildInformationRequestSizeBound(t *testing.T) {
	var missing []string
	for i := 0; i < 200; i++ {
		missing = append(missing, strings.Repeat("x", 500))
	}
	draft := BuildInformationRequest(missing)
	if len(draft) > MaxInformationRequestBytes {
		t.Fatalf("expected draft within %d bytes, got %d", MaxInformationRequestBytes, len(draft))
	}
}
