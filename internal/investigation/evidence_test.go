package investigation

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestBuildBodyIncludesIssueAndComments(t *testing.T) {
	rendered := BuildBody("issue text", []EvidenceItem{
		{ID: 111, Source: SourceComment, Author: "alice", Body: "repro here"},
		{ID: 222, Source: SourceComment, Author: "bob", Body: "same here"},
	})
	for _, want := range []string{"issue text", "--- Comment by @alice (comment:111) ---", "repro here", "--- Comment by @bob (comment:222) ---", "same here"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected %q in %q", want, rendered)
		}
	}
}

func TestBuildBodyCommentCap(t *testing.T) {
	comments := make([]EvidenceItem, 0, 12)
	for i := 0; i < 12; i++ {
		comments = append(comments, EvidenceItem{ID: int64(1000 + i), Source: SourceComment, Author: "a", Body: "x"})
	}
	rendered := BuildBody("body", comments)
	if got := strings.Count(rendered, "--- Comment by @a (comment:"); got != MaxComments {
		t.Fatalf("expected %d banners, got %d", MaxComments, got)
	}
}

func TestBuildBodyCommentTruncation(t *testing.T) {
	big := strings.Repeat("z", MaxCommentBytes+500)
	rendered := BuildBody("b", []EvidenceItem{{ID: 7, Source: SourceComment, Author: "a", Body: big}})
	if !strings.HasSuffix(rendered, truncateMarker) {
		t.Fatalf("expected truncation marker, got tail %q", rendered[len(rendered)-20:])
	}
	if !utf8.ValidString(rendered) {
		t.Fatalf("expected valid UTF-8")
	}
	commentPart := rendered[strings.Index(rendered, "--- Comment by @a (comment:7) ---"):]
	if len(commentPart) > len("\n--- Comment by @a (comment:7) ---\n")+MaxCommentBytes {
		t.Fatalf("comment exceeded per-comment bound: %d", len(commentPart))
	}
}

func TestBuildBodyRuneBoundary(t *testing.T) {
	big := strings.Repeat("é", MaxCommentBytes+10)
	rendered := BuildBody("", []EvidenceItem{{ID: 7, Source: SourceComment, Author: "a", Body: big}})
	if !utf8.ValidString(rendered) {
		t.Fatalf("expected valid UTF-8 at rune boundary")
	}
	if !strings.HasSuffix(rendered, truncateMarker) {
		t.Fatalf("expected truncation marker")
	}
}

func TestBuildBodyTotalBound(t *testing.T) {
	big := strings.Repeat("y", MaxEvidenceBytes+100)
	comments := []EvidenceItem{{Source: SourceComment, Author: "a", Body: "should not appear"}}
	rendered := BuildBody(big, comments)
	if len(rendered) > MaxEvidenceBytes {
		t.Fatalf("exceeded total bound: %d", len(rendered))
	}
	if strings.Contains(rendered, "should not appear") {
		t.Fatalf("comment must not displace full issue body")
	}
	if !strings.HasSuffix(rendered, truncateMarker) {
		t.Fatalf("expected issue body truncation marker")
	}
}

func TestBuildBodyIssuePriority(t *testing.T) {
	body := strings.Repeat("b", MaxEvidenceBytes-100)
	rendered := BuildBody(body, []EvidenceItem{{ID: 9, Source: SourceComment, Author: "a", Body: "c"}})
	if !strings.Contains(rendered, body) {
		t.Fatalf("expected full issue body preserved")
	}
	if !strings.Contains(rendered, "--- Comment by @a (comment:9) ---") {
		t.Fatalf("expected comment in remaining budget")
	}
	if len(rendered) > MaxEvidenceBytes {
		t.Fatalf("exceeded total bound: %d", len(rendered))
	}
}

func TestBuildBodyDeterministic(t *testing.T) {
	comments := []EvidenceItem{{ID: 1, Source: SourceComment, Author: "a", Body: "one"}, {ID: 2, Source: SourceComment, Author: "b", Body: "two"}}
	if first, second := BuildBody("body", comments), BuildBody("body", comments); first != second {
		t.Fatalf("expected deterministic output")
	}
}

func TestBuildBodyNoComments(t *testing.T) {
	if rendered := BuildBody("just the issue", nil); rendered != "just the issue" {
		t.Fatalf("expected untouched body, got %q", rendered)
	}
}
