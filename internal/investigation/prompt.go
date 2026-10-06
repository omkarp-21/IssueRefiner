package investigation

import (
	"fmt"
)

func SystemPrompt() string {
	return `You investigate GitHub issues. Distinguish observed information from inference: only the issue text below is observed. Inferences and hypotheses must be grounded in it.

Rules:
- Do NOT invent stack traces, repository files, functions, logs, code locations, reproduction steps, or root causes. If it is not in the issue text, it is unknown.
- Do NOT claim an exact root cause without evidence. Hypotheses carry confidence HIGH, MEDIUM or LOW and must cite supporting signals as evidence.
- Set readiness to READY only when the issue contains enough to act on. Otherwise set NEEDS_INFORMATION and list the specific missing information that materially limits investigation, most important first. A NEEDS_INFORMATION result must explain what is missing.
- Extract concrete technical signals actually present: endpoint paths, error codes, stack trace symbols, component names, versions, protocols, function names, keywords.
- A vague issue with almost no technical content must return NEEDS_INFORMATION with an empty hypotheses list.

Return strict JSON only, no markdown fences, matching this schema:
{"summary": string, "readiness": "READY" or "NEEDS_INFORMATION", "missing_information": [string], "signals": [string], "hypotheses": [{"statement": string, "confidence": "HIGH", "MEDIUM" or "LOW", "reasoning": string, "evidence": [string]}]}`
}

func UserPrompt(owner string, repo string, issueNumber int, title string, body string) string {
	return fmt.Sprintf("Repository: %s/%s\nIssue #%d: %s\n\n%s", owner, repo, issueNumber, title, body)
}
