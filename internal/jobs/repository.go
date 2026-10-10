package jobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCreateJob   = errors.New("failed to create job")
	ErrFindJob     = errors.New("failed to find job")
	ErrUpdateJob   = errors.New("failed to update job")
	ErrClaimJob    = errors.New("failed to claim job")
	ErrJobNotFound = errors.New("job not found")
	ErrNotClaimed  = errors.New("job already claimed")
)

type JobRepository struct {
	dbPool *pgxpool.Pool
}

func NewJobRepository(dbPool *pgxpool.Pool) JobRepository {
	return JobRepository{dbPool: dbPool}
}

const jobColumns = "id, organization_id, repository_id, provider_config_id, provider_type, model, created_by, kind, issue_number, status, input_title, input_body, result, error, input_tokens, output_tokens, attempts, created_at, updated_at"

func (jobRepository *JobRepository) Create(ctx context.Context, organizationID uuid.UUID, repositoryID uuid.UUID, providerConfigID uuid.UUID, providerType string, model string, createdBy uuid.UUID, kind string, issueNumber int, title string, body string) (*Job, error) {
	var job Job
	err := jobRepository.dbPool.QueryRow(
		ctx,
		"INSERT INTO jobs (id, organization_id, repository_id, provider_config_id, provider_type, model, created_by, kind, issue_number, status, input_title, input_body) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING "+jobColumns,
		uuid.New(),
		organizationID,
		repositoryID,
		providerConfigID,
		providerType,
		model,
		createdBy,
		kind,
		issueNumber,
		StatusQueued,
		title,
		body,
	).Scan(&job.ID,
		&job.OrganizationID,
		&job.RepositoryID,
		&job.ProviderConfigID,
		&job.ProviderType,
		&job.Model,
		&job.CreatedBy,
		&job.Kind,
		&job.IssueNumber,
		&job.Status,
		&job.InputTitle,
		&job.InputBody,
		&job.Result,
		&job.Error,
		&job.InputTokens,
		&job.OutputTokens,
		&job.Attempts,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateJob, err)
	}
	return &job, nil
}

func (jobRepository *JobRepository) FindByID(ctx context.Context, id uuid.UUID) (*Job, error) {
	var job Job
	err := jobRepository.dbPool.QueryRow(
		ctx,
		"SELECT "+jobColumns+" FROM jobs WHERE id=$1",
		id,
	).Scan(&job.ID,
		&job.OrganizationID,
		&job.RepositoryID,
		&job.ProviderConfigID,
		&job.ProviderType,
		&job.Model,
		&job.CreatedBy,
		&job.Kind,
		&job.IssueNumber,
		&job.Status,
		&job.InputTitle,
		&job.InputBody,
		&job.Result,
		&job.Error,
		&job.InputTokens,
		&job.OutputTokens,
		&job.Attempts,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFindJob, err)
	}
	return &job, nil
}

func (jobRepository *JobRepository) ListForIssue(ctx context.Context, organizationID uuid.UUID, repositoryID uuid.UUID, issueNumber int) ([]Job, error) {
	rows, err := jobRepository.dbPool.Query(
		ctx,
		"SELECT "+jobColumns+" FROM jobs WHERE organization_id=$1 AND repository_id=$2 AND issue_number=$3 AND kind=$4 ORDER BY created_at DESC",
		organizationID,
		repositoryID,
		issueNumber,
		KindInvestigate,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFindJob, err)
	}
	defer rows.Close()
	investigations := []Job{}
	for rows.Next() {
		var job Job
		if err := rows.Scan(&job.ID,
			&job.OrganizationID,
			&job.RepositoryID,
			&job.ProviderConfigID,
			&job.ProviderType,
			&job.Model,
			&job.CreatedBy,
			&job.Kind,
			&job.IssueNumber,
			&job.Status,
			&job.InputTitle,
			&job.InputBody,
			&job.Result,
			&job.Error,
			&job.InputTokens,
			&job.OutputTokens,
			&job.Attempts,
			&job.CreatedAt,
			&job.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrFindJob, err)
		}
		investigations = append(investigations, job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFindJob, err)
	}
	return investigations, nil
}

func (jobRepository *JobRepository) MarkDone(ctx context.Context, id uuid.UUID, result string, inputTokens *int, outputTokens *int) (*Job, error) {
	var job Job
	err := jobRepository.dbPool.QueryRow(
		ctx,
		"UPDATE jobs SET status=$1, result=$2, error=NULL, input_tokens=$3, output_tokens=$4, updated_at=NOW() WHERE id=$5 AND status=$6 RETURNING "+jobColumns,
		StatusDone,
		result,
		inputTokens,
		outputTokens,
		id,
		StatusRunning,
	).Scan(&job.ID,
		&job.OrganizationID,
		&job.RepositoryID,
		&job.ProviderConfigID,
		&job.ProviderType,
		&job.Model,
		&job.CreatedBy,
		&job.Kind,
		&job.IssueNumber,
		&job.Status,
		&job.InputTitle,
		&job.InputBody,
		&job.Result,
		&job.Error,
		&job.InputTokens,
		&job.OutputTokens,
		&job.Attempts,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUpdateJob, err)
	}
	return &job, nil
}

func (jobRepository *JobRepository) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) (*Job, error) {
	var job Job
	err := jobRepository.dbPool.QueryRow(
		ctx,
		"UPDATE jobs SET status=$1, error=$2, updated_at=NOW() WHERE id=$3 AND (status=$4 OR status=$5) RETURNING "+jobColumns,
		StatusFailed,
		errMsg,
		id,
		StatusQueued,
		StatusRunning,
	).Scan(&job.ID,
		&job.OrganizationID,
		&job.RepositoryID,
		&job.ProviderConfigID,
		&job.ProviderType,
		&job.Model,
		&job.CreatedBy,
		&job.Kind,
		&job.IssueNumber,
		&job.Status,
		&job.InputTitle,
		&job.InputBody,
		&job.Result,
		&job.Error,
		&job.InputTokens,
		&job.OutputTokens,
		&job.Attempts,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUpdateJob, err)
	}
	return &job, nil
}

func (jobRepository *JobRepository) Claim(ctx context.Context, id uuid.UUID) (*Job, error) {
	var job Job
	err := jobRepository.dbPool.QueryRow(
		ctx,
		"UPDATE jobs SET status=$1, attempts=attempts+1, updated_at=NOW() WHERE id=$2 AND status=$3 RETURNING "+jobColumns,
		StatusRunning,
		id,
		StatusQueued,
	).Scan(&job.ID,
		&job.OrganizationID,
		&job.RepositoryID,
		&job.ProviderConfigID,
		&job.ProviderType,
		&job.Model,
		&job.CreatedBy,
		&job.Kind,
		&job.IssueNumber,
		&job.Status,
		&job.InputTitle,
		&job.InputBody,
		&job.Result,
		&job.Error,
		&job.InputTokens,
		&job.OutputTokens,
		&job.Attempts,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrClaimJob, err)
	}
	return &job, nil
}

const interruptedReason = "job interrupted before completion; marked failed on startup recovery"

func (jobRepository *JobRepository) FailStaleRunning(ctx context.Context, olderThan time.Duration) (int64, error) {
	tag, err := jobRepository.dbPool.Exec(
		ctx,
		"UPDATE jobs SET status=$1, error=$2, updated_at=NOW() WHERE status=$3 AND updated_at < NOW() - make_interval(secs => $4)",
		StatusFailed,
		interruptedReason,
		StatusRunning,
		olderThan.Seconds(),
	)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrUpdateJob, err)
	}
	return tag.RowsAffected(), nil
}
