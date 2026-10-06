package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestValidateRefineInput(t *testing.T) {
	if err := ValidateRefineInput(7, "title"); err != nil {
		t.Fatalf("expected valid input, got %v", err)
	}
	cases := []struct {
		name  string
		num   int
		title string
	}{
		{"zero issue", 0, "title"},
		{"negative issue", -3, "title"},
		{"empty title", 7, ""},
	}
	for _, tc := range cases {
		if err := ValidateRefineInput(tc.num, tc.title); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("%s: expected ErrInvalidInput, got %v", tc.name, err)
		}
	}
}

func TestCheckOrg(t *testing.T) {
	organizationID := uuid.New()
	job := &Job{OrganizationID: organizationID}
	if err := checkOrg(job, organizationID); err != nil {
		t.Fatalf("expected same organization to pass, got %v", err)
	}
	if err := checkOrg(job, uuid.New()); !errors.Is(err, ErrJobNotFound) {
		t.Fatalf("expected ErrJobNotFound, got %v", err)
	}
}

func TestRetryTerminalSuccess(t *testing.T) {
	calls := 0
	err := retryTerminal(context.Background(), func(ctx context.Context) error {
		calls++
		return nil
	})
	if err != nil || calls != 1 {
		t.Fatalf("expected one successful call, got %d %v", calls, err)
	}
}

func TestRetryTerminalThenSuccess(t *testing.T) {
	calls := 0
	boom := errors.New("boom")
	err := retryTerminal(context.Background(), func(ctx context.Context) error {
		calls++
		if calls < 3 {
			return boom
		}
		return nil
	})
	if err != nil || calls != 3 {
		t.Fatalf("expected success on third call, got %d %v", calls, err)
	}
}

func TestRetryTerminalExhausted(t *testing.T) {
	calls := 0
	boom := errors.New("boom")
	err := retryTerminal(context.Background(), func(ctx context.Context) error {
		calls++
		return boom
	})
	if !errors.Is(err, boom) || calls != terminalAttempts {
		t.Fatalf("expected %d calls with last error, got %d %v", terminalAttempts, calls, err)
	}
}

func TestRetryTerminalNoRowsImmediate(t *testing.T) {
	calls := 0
	err := retryTerminal(context.Background(), func(ctx context.Context) error {
		calls++
		return pgx.ErrNoRows
	})
	if !errors.Is(err, pgx.ErrNoRows) || calls != 1 {
		t.Fatalf("expected single attempt for deterministic failure, got %d %v", calls, err)
	}
}

func TestRetryTerminalCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	err := retryTerminal(ctx, func(ctx context.Context) error {
		return errors.New("boom")
	})
	if err == nil || time.Since(start) > terminalRetryDelay {
		t.Fatalf("expected prompt return on cancelled context, got %v", err)
	}
}

func TestValidateInputSize(t *testing.T) {
	if err := ValidateInputSize("title", "body"); err != nil {
		t.Fatalf("expected small input to pass, got %v", err)
	}
	exact := make([]byte, MaxIssueBytes-len("title"))
	if err := ValidateInputSize("title", string(exact)); err != nil {
		t.Fatalf("expected input exactly at limit to pass, got %v", err)
	}
	over := make([]byte, MaxIssueBytes-len("title")+1)
	if err := ValidateInputSize("title", string(over)); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput above limit, got %v", err)
	}
}
