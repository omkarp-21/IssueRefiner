package investigation

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	SourceIssue   = "issue"
	SourceComment = "comment"

	MaxComments      = 10
	MaxCommentBytes  = 2 * 1024
	MaxEvidenceBytes = 20 * 1024

	truncateMarker = "[truncated]"
)

type EvidenceItem struct {
	ID     int64
	Source string
	Author string
	Body   string
}

func truncateTo(text string, max int) string {
	if len(text) <= max {
		return text
	}
	cut := max - len(truncateMarker)
	for cut > 0 && !utf8.ValidString(text[:cut]) {
		cut--
	}
	return text[:cut] + truncateMarker
}

func BuildBody(body string, comments []EvidenceItem) string {
	rendered := truncateTo(body, MaxEvidenceBytes)
	remaining := MaxEvidenceBytes - len(rendered)
	count := 0
	for _, comment := range comments {
		if count >= MaxComments {
			break
		}
		banner := fmt.Sprintf("\n--- Comment by @%s (%s) ---\n", comment.Author, ReferenceID(SourceComment, comment.ID))
		if len(banner) >= remaining {
			break
		}
		budget := remaining - len(banner)
		if budget > MaxCommentBytes {
			budget = MaxCommentBytes
		}
		part := truncateTo(comment.Body, budget)
		rendered += banner + part
		remaining -= len(banner) + len(part)
		count++
	}
	return rendered
}

func ReferenceID(kind string, id int64) string {
	return fmt.Sprintf("%s:%d", kind, id)
}

func EvidenceIDs(rendered string) map[string]bool {
	ids := map[string]bool{SourceIssue: true}
	rest := rendered
	for {
		start := strings.Index(rest, "(comment:")
		if start < 0 {
			break
		}
		rest = rest[start+len("(comment:"):]
		end := strings.Index(rest, ")")
		if end < 0 {
			break
		}
		id := rest[:end]
		if id != "" {
			ids["comment:"+id] = true
		}
		rest = rest[end+1:]
	}
	return ids
}
