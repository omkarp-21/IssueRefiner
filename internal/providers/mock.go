package providers

import (
	"context"
	"fmt"
	"time"
)

const MockDefaultModel = "mock-test"

var mockModels = []string{"mock-test", "mock-slow", "mock-quota", "mock-rate", "mock-invalid", "mock-unavailable", "mock-investigate-ready", "mock-investigate-needs-info", "mock-investigate-malformed"}

type MockProvider struct{}

func NewMockProvider() MockProvider {
	return MockProvider{}
}

func (mock MockProvider) Type() string {
	return ProviderMock
}

func (mock MockProvider) SupportsModel(model string) bool {
	for _, supported := range mockModels {
		if model == supported {
			return true
		}
	}
	return false
}

func (mock MockProvider) ValidateCredentials(ctx context.Context, apiKey string, model string) error {
	if !mock.SupportsModel(model) {
		return fmt.Errorf("%w: %s", ErrModelUnavailable, model)
	}
	if model == "mock-invalid" {
		return NewProviderError(ErrInvalidCredentials, fmt.Errorf("mock invalid credentials"))
	}
	return nil
}

func (mock MockProvider) Generate(ctx context.Context, apiKey string, model string, input Input) (Result, error) {
	if !mock.SupportsModel(model) {
		return Result{}, fmt.Errorf("%w: %s", ErrModelUnavailable, model)
	}
	switch model {
	case "mock-quota":
		return Result{}, NewProviderError(ErrInsufficientQuota, fmt.Errorf("mock quota exhausted"))
	case "mock-rate":
		return Result{}, NewRateLimitedError(fmt.Errorf("mock rate limited"), time.Second)
	case "mock-invalid":
		return Result{}, NewProviderError(ErrInvalidCredentials, fmt.Errorf("mock invalid credentials"))
	case "mock-unavailable":
		return Result{}, NewProviderError(ErrProviderUnavailable, fmt.Errorf("mock unavailable"))
	case "mock-investigate-ready":
		return Result{Text: `{"summary":"Connection pool exhaustion under load","readiness":"READY","missing_information":[],"signals":["pq: too many clients already","pool size 5"],"hypotheses":[{"statement":"Pool is undersized for peak concurrency","confidence":"MEDIUM","reasoning":"Error appears only under load","evidence":["pq: too many clients already"]}]`}, nil
	case "mock-investigate-needs-info":
		return Result{Text: `{"summary":"Something is broken","readiness":"NEEDS_INFORMATION","missing_information":["reproduction steps","expected behavior"],"signals":[],"hypotheses":[]`}, nil
	case "mock-investigate-malformed":
		return Result{Text: `not json{{{`}, nil
	case "mock-slow":
		select {
		case <-time.After(2 * time.Second):
		case <-ctx.Done():
			return Result{}, NewProviderError(ErrProviderTimeout, ctx.Err())
		}
	}
	if err := ctx.Err(); err != nil {
		return Result{}, NewProviderError(ErrProviderTimeout, err)
	}
	body := input.Body
	if len(body) > 200 {
		body = body[:200]
	}
	return Result{Text: fmt.Sprintf("Refined: %s (%s/%s#%d)\nSummary: %s\nSteps:\n1. Reproduce the issue\n2. Narrow the cause\n3. Verify the fix",
		input.Title, input.Owner, input.Repo, input.IssueNumber, body)}, nil
}
