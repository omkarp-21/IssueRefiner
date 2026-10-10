package jobs

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"issue-refiner/internal/investigation"

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

func needsInfoResult() string {
	return `{"summary":"Pool exhaustion","readiness":"NEEDS_INFORMATION","missing_information":["repro steps"],"signals":["x"],"observed_behavior":"","expected_behavior":"","reproduction_steps":[],"environment":"","hypotheses":[]}`
}

func readyResult() string {
	return `{"summary":"Pool exhaustion","readiness":"READY","missing_information":[],"signals":["x"],"observed_behavior":"o","expected_behavior":"e","reproduction_steps":[],"environment":"","hypotheses":[]}`
}

func gateSource() *Job {
	orgID := uuid.New()
	repoID := uuid.New()
	result := needsInfoResult()
	return &Job{
		ID:             uuid.New(),
		OrganizationID: orgID,
		RepositoryID:   &repoID,
		CreatedBy:      uuid.New(),
		Kind:           KindInvestigate,
		IssueNumber:    7,
		Status:         StatusDone,
		Result:         &result,
	}
}

func TestValidateReinvestigateSourceValid(t *testing.T) {
	job := gateSource()
	if err := validateReinvestigateSource(job, job.OrganizationID, *job.RepositoryID, job.IssueNumber); err != nil {
		t.Fatalf("expected valid source, got %v", err)
	}
}

func TestValidateReinvestigateSourceMatrix(t *testing.T) {
	base := gateSource()
	ready := readyResult()
	cases := []struct {
		name   string
		mutate func(*Job)
	}{
		{"ready", func(j *Job) { j.Result = &ready }},
		{"failed status", func(j *Job) { j.Status = StatusFailed }},
		{"queued status", func(j *Job) { j.Status = StatusQueued }},
		{"running status", func(j *Job) { j.Status = StatusRunning }},
		{"refine kind", func(j *Job) { j.Kind = KindRefine }},
		{"nil repository", func(j *Job) { j.RepositoryID = nil }},
		{"wrong repository", func(j *Job) { id := uuid.New(); j.RepositoryID = &id }},
		{"wrong issue", func(j *Job) { j.IssueNumber = 8 }},
		{"nil result", func(j *Job) { j.Result = nil }},
		{"malformed result", func(j *Job) { bad := "not json"; j.Result = &bad }},
	}
	for _, tc := range cases {
		job := gateSource()
		job.OrganizationID = base.OrganizationID
		job.RepositoryID = base.RepositoryID
		tc.mutate(job)
		if err := validateReinvestigateSource(job, base.OrganizationID, *base.RepositoryID, 7); !errors.Is(err, ErrInvalidReinvestigation) {
			t.Fatalf("%s: expected ErrInvalidReinvestigation, got %v", tc.name, err)
		}
	}
}

func TestValidateReinvestigateSourceWrongOrg(t *testing.T) {
	job := gateSource()
	if err := validateReinvestigateSource(job, uuid.New(), *job.RepositoryID, job.IssueNumber); !errors.Is(err, ErrInvalidReinvestigation) {
		t.Fatalf("expected ErrInvalidReinvestigation, got %v", err)
	}
}

func TestPostInformationRequestEmptyBody(t *testing.T) {
	service := &JobService{}
	for _, body := range []string{"", "   ", "\n\t "} {
		_, err := service.PostInformationRequest(context.Background(), uuid.New(), uuid.New(), uuid.New(), 7, uuid.New(), body)
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput for %q, got %v", body, err)
		}
	}
}

func TestPostInformationRequestBodySizeBound(t *testing.T) {
	atLimit := strings.Repeat("a", investigation.MaxInformationRequestBytes)
	if err := validateCommentBody(atLimit); err != nil {
		t.Fatalf("expected body exactly at limit to pass validation, got %v", err)
	}
	over := strings.Repeat("a", investigation.MaxInformationRequestBytes+1)
	if err := validateCommentBody(over); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput one byte over, got %v", err)
	}
	multibyte := strings.Repeat("é", investigation.MaxInformationRequestBytes/2+1)
	if err := validateCommentBody(multibyte); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected multibyte body measured in bytes to be rejected, got %v", err)
	}
	padded := strings.Repeat(" ", 100) + strings.Repeat("b", investigation.MaxInformationRequestBytes+1)
	if err := validateCommentBody(padded); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected whitespace padding not to bypass the bound, got %v", err)
	}
}

func TestPostInformationRequestOversizedHasNoSideEffects(t *testing.T) {
	service := &JobService{}
	over := strings.Repeat("a", investigation.MaxInformationRequestBytes+1)
	if _, err := service.PostInformationRequest(context.Background(), uuid.New(), uuid.New(), uuid.New(), 7, uuid.New(), over); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
