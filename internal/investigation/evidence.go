package investigation

import (
	"fmt"
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
		banner := fmt.Sprintf("\n--- Comment by @%s ---\n", comment.Author)
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
