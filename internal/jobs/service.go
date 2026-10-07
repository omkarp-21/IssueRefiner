package jobs

import (
	"context"
	"errors"
	"fmt"
	"issue-refiner/internal/investigation"
	"issue-refiner/internal/providers"
	"issue-refiner/internal/repos"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidInput = errors.New("invalid refine input")
	ErrRefineFailed = errors.New("refinement failed")
)

// MaxIssueBytes bounds the total GitHub issue content sent to a provider
// in a single job. Without it one oversized issue body could consume an
// unreasonable share of the user's provider quota.
const MaxIssueBytes = 24 * 1024

const (
	terminalAttempts   = 3
	terminalRetryDelay = 200 * time.Millisecond
)

func retryTerminal(ctx context.Context, op func(context.Context) error) error {
	var err error
	for attempt := 0; attempt < terminalAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(terminalRetryDelay):
			}
		}
		if err = op(ctx); err == nil {
			return nil
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return err
		}
	}
	return err
}

type JobService struct {
	JobRepository *JobRepository
	Repositories  *repos.RepositoryService
	Providers     *providers.ProviderService
}

func NewJobService(jobRepository *JobRepository, repositoryService *repos.RepositoryService, providerService *providers.ProviderService) JobService {
	return JobService{JobRepository: jobRepository, Repositories: repositoryService, Providers: providerService}
}

func ValidateRefineInput(issueNumber int, title string) error {
	if issueNumber <= 0 || title == "" {
		return fmt.Errorf("%w: issue_number and title are required", ErrInvalidInput)
	}
	return nil
}

func ValidateInputSize(title string, body string) error {
	if len(title)+len(body) > MaxIssueBytes {
		return fmt.Errorf("%w: issue content exceeds %d bytes", ErrInvalidInput, MaxIssueBytes)
	}
	return nil
}

func checkOrg(job *Job, organizationID uuid.UUID) error {
	if job.OrganizationID != organizationID {
		return fmt.Errorf("%w", ErrJobNotFound)
	}
	return nil
}

func (jobService *JobService) EnqueueIssue(ctx context.Context, organizationID uuid.UUID, repositoryID uuid.UUID, providerConfigID uuid.UUID, model string, createdBy uuid.UUID, issueNumber int, title string, body string) (*Job, error) {
	if repositoryID == uuid.Nil {
		return nil, fmt.Errorf("%w: repository is required", ErrInvalidInput)
	}
	if providerConfigID == uuid.Nil {
		return nil, fmt.Errorf("%w: provider is required", ErrInvalidInput)
	}
	if model == "" {
		return nil, fmt.Errorf("%w: model is required", ErrInvalidInput)
	}
	if err := ValidateRefineInput(issueNumber, title); err != nil {
		return nil, err
	}
	if err := ValidateInputSize(title, body); err != nil {
		return nil, err
	}
	config, err := jobService.Providers.ResolveForJob(ctx, organizationID, providerConfigID, model)
	if err != nil {
		return nil, err
	}
	if _, err := jobService.Repositories.GetRepo(ctx, organizationID, repositoryID); err != nil {
		return nil, err
	}
	return jobService.JobRepository.Create(ctx, organizationID, repositoryID, config.ID, config.ProviderType, model, createdBy, KindRefine, issueNumber, title, body)
}

func (jobService *JobService) EnqueueInvestigation(ctx context.Context, organizationID uuid.UUID, repositoryID uuid.UUID, providerConfigID uuid.UUID, model string, createdBy uuid.UUID, issueNumber int) (*Job, error) {
	if repositoryID == uuid.Nil {
		return nil, fmt.Errorf("%w: repository is required", ErrInvalidInput)
	}
	if providerConfigID == uuid.Nil {
		return nil, fmt.Errorf("%w: provider is required", ErrInvalidInput)
	}
	if model == "" {
		return nil, fmt.Errorf("%w: model is required", ErrInvalidInput)
	}
	if issueNumber <= 0 {
		return nil, fmt.Errorf("%w: issue number is required", ErrInvalidInput)
	}
	config, err := jobService.Providers.ResolveForJob(ctx, organizationID, providerConfigID, model)
	if err != nil {
		return nil, err
	}
	issue, err := jobService.Repositories.GetIssue(ctx, organizationID, createdBy, repositoryID, issueNumber)
	if err != nil {
		return nil, err
	}
	if err := ValidateInputSize(issue.Title, issue.Body); err != nil {
		return nil, err
	}
	comments, err := jobService.Repositories.GetIssueComments(ctx, organizationID, createdBy, repositoryID, issueNumber)
	if err != nil {
		return nil, err
	}
	evidence := make([]investigation.EvidenceItem, 0, len(comments))
	for _, comment := range comments {
		evidence = append(evidence, investigation.EvidenceItem{
			Source: investigation.SourceComment,
			Author: comment.Author,
			Body:   comment.Body,
		})
	}
	body := investigation.BuildBody(issue.Body, evidence)
	if err := ValidateInputSize(issue.Title, body); err != nil {
		return nil, err
	}
	return jobService.JobRepository.Create(ctx, organizationID, repositoryID, config.ID, config.ProviderType, model, createdBy, KindInvestigate, issue.Number, issue.Title, body)
}

func (jobService *JobService) FailJob(ctx context.Context, id uuid.UUID, errMsg string) (*Job, error) {
	return jobService.JobRepository.MarkFailed(ctx, id, errMsg)
}

func (jobService *JobService) ProcessJob(ctx context.Context, id uuid.UUID) error {
	job, err := jobService.JobRepository.Claim(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: %w", ErrNotClaimed, err)
		}
		return err
	}
	// Terminal writes use their own bounded context so a cancelled
	// job context cannot leave the job stuck in running.
	persistCtx, persistCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer persistCancel()
	fail := func(msg string, err error) error {
		if markErr := retryTerminal(persistCtx, func(ctx context.Context) error {
			_, markErr := jobService.JobRepository.MarkFailed(ctx, job.ID, msg)
			return markErr
		}); markErr != nil {
			return markErr
		}
		return fmt.Errorf("%w: %w", ErrRefineFailed, err)
	}
	if job.RepositoryID == nil {
		return fail("job is not linked to a repository", ErrRefineFailed)
	}
	repository, err := jobService.Repositories.GetByID(ctx, *job.RepositoryID)
	if err != nil {
		return fail("repository not found", err)
	}
	if repository.OrganizationID != job.OrganizationID {
		return fail("repository does not belong to job organization", ErrRefineFailed)
	}
	providerType, model, configID := providers.ProviderMock, providers.MockDefaultModel, (*uuid.UUID)(nil)
	if job.ProviderConfigID != nil && job.ProviderType != nil && job.Model != nil {
		providerType, model, configID = *job.ProviderType, *job.Model, job.ProviderConfigID
	}
	input := providers.Input{
		Owner:       repository.Owner,
		Repo:        repository.Name,
		IssueNumber: job.IssueNumber,
		Title:       job.InputTitle,
		Body:        job.InputBody,
	}
	if job.Kind == KindInvestigate {
		input.SystemPrompt = investigation.SystemPrompt()
	}
	generated, err := jobService.Providers.GenerateWithConfig(ctx, configID, providerType, model, input)
	if err != nil {
		return fail(err.Error(), err)
	}
	text := generated.Text
	if job.Kind == KindInvestigate {
		investigationResult, err := investigation.ParseResult(generated.Text)
		if err != nil {
			return fail("provider returned invalid investigation output", err)
		}
		text, err = investigation.MarshalResult(investigationResult)
		if err != nil {
			return fail("failed to store investigation output", err)
		}
	}
	err = retryTerminal(persistCtx, func(ctx context.Context) error {
		inputTokens, outputTokens := usagePointers(generated.InputTokens, generated.OutputTokens)
		_, err := jobService.JobRepository.MarkDone(ctx, job.ID, text, inputTokens, outputTokens)
		return err
	})
	return err
}

func usagePointers(inputTokens int, outputTokens int) (*int, *int) {
	if inputTokens == 0 && outputTokens == 0 {
		return nil, nil
	}
	return &inputTokens, &outputTokens
}

func (jobService *JobService) GetForOrg(ctx context.Context, organizationID uuid.UUID, id uuid.UUID) (*Job, *repos.Repository, error) {
	job, err := jobService.JobRepository.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, fmt.Errorf("%w", ErrJobNotFound)
		}
		return nil, nil, err
	}
	if err := checkOrg(job, organizationID); err != nil {
		return nil, nil, err
	}
	if job.RepositoryID == nil {
		return job, nil, nil
	}
	repository, err := jobService.Repositories.GetRepo(ctx, organizationID, *job.RepositoryID)
	if err != nil {
		if errors.Is(err, repos.ErrRepositoryNotFound) {
			return job, nil, nil
		}
		return nil, nil, err
	}
	return job, repository, nil
}
