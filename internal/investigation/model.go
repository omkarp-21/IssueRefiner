package investigation

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	ReadinessReady     = "READY"
	ReadinessNeedsInfo = "NEEDS_INFORMATION"
)

const (
	ConfidenceHigh   = "HIGH"
	ConfidenceMedium = "MEDIUM"
	ConfidenceLow    = "LOW"
)

var (
	ErrInvalidResult = errors.New("invalid investigation result")
)

type Hypothesis struct {
	Statement  string   `json:"statement"`
	Confidence string   `json:"confidence"`
	Reasoning  string   `json:"reasoning"`
	Evidence   []string `json:"evidence"`
}

type InvestigationResult struct {
	Summary            string       `json:"summary"`
	Readiness          string       `json:"readiness"`
	MissingInformation []string     `json:"missing_information"`
	Signals            []string     `json:"signals"`
	Hypotheses         []Hypothesis `json:"hypotheses"`
}

func ParseResult(raw string) (*InvestigationResult, error) {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(strings.TrimSpace(cleaned), "```")
	cleaned = strings.TrimSpace(cleaned)

	var result InvestigationResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, fmt.Errorf("%w: malformed JSON: %v", ErrInvalidResult, err)
	}
	if strings.TrimSpace(result.Summary) == "" {
		return nil, fmt.Errorf("%w: summary is required", ErrInvalidResult)
	}
	if result.Readiness != ReadinessReady && result.Readiness != ReadinessNeedsInfo {
		return nil, fmt.Errorf("%w: readiness must be READY or NEEDS_INFORMATION", ErrInvalidResult)
	}
	if result.MissingInformation == nil {
		result.MissingInformation = []string{}
	}
	if result.Signals == nil {
		result.Signals = []string{}
	}
	if result.Hypotheses == nil {
		result.Hypotheses = []Hypothesis{}
	}
	if result.Readiness == ReadinessNeedsInfo && len(result.MissingInformation) == 0 {
		return nil, fmt.Errorf("%w: NEEDS_INFORMATION requires missing_information", ErrInvalidResult)
	}
	for i, hypothesis := range result.Hypotheses {
		if strings.TrimSpace(hypothesis.Statement) == "" {
			return nil, fmt.Errorf("%w: hypothesis %d needs a statement", ErrInvalidResult, i)
		}
		if hypothesis.Confidence != ConfidenceHigh && hypothesis.Confidence != ConfidenceMedium && hypothesis.Confidence != ConfidenceLow {
			return nil, fmt.Errorf("%w: hypothesis %d confidence must be HIGH, MEDIUM or LOW", ErrInvalidResult, i)
		}
		if strings.TrimSpace(hypothesis.Reasoning) == "" {
			return nil, fmt.Errorf("%w: hypothesis %d needs reasoning", ErrInvalidResult, i)
		}
		if len(hypothesis.Evidence) == 0 {
			return nil, fmt.Errorf("%w: hypothesis %d needs evidence", ErrInvalidResult, i)
		}
	}
	return &result, nil
}

func MarshalResult(result *InvestigationResult) (string, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidResult, err)
	}
	return string(raw), nil
}
