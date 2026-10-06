package repos

import (
	"context"
	"errors"
	"fmt"
	"issue-refiner/internal/github"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidRepo    = errors.New("invalid repository reference")
	ErrGitHubToken    = errors.New("github account not connected")
	ErrGitHubUpstream = errors.New("github request failed")
)

type GitHubClient interface {
	GetRepository(ctx context.Context, token string, owner string, repo string) (*github.Repository, error)
	GetIssue(ctx context.Context, token string, owner string, repo string, issueNumber int) (*github.Issue, error)
	ListContributors(ctx context.Context, token string, owner string, repo string, page int, perPage int) ([]github.Contributor, error)
	ListIssues(ctx context.Context, token string, owner string, repo string, state string, page int, perPage int) ([]github.Issue, error)
}

type TokenStore interface {
	Get(ctx context.Context, userID uuid.UUID) (string, error)
}

type RepositoryService struct {
	RepositoryRepository *RepositoryRepository
	GitHub               GitHubClient
	Tokens               TokenStore
}

func NewRepositoryService(repositoryRepository *RepositoryRepository, githubClient GitHubClient, tokens TokenStore) RepositoryService {
	return RepositoryService{RepositoryRepository: repositoryRepository, GitHub: githubClient, Tokens: tokens}
}

func NormalizePage(page int, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}

func ValidateIssueState(state string) error {
	if state != "open" && state != "closed" && state != "all" {
		return fmt.Errorf("%w: state must be open, closed or all", ErrInvalidRepo)
	}
	return nil
}

func (repositoryService *RepositoryService) ImportRepo(ctx context.Context, organizationID uuid.UUID, userID uuid.UUID, owner string, repo string) (*Repository, error) {
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("%w: owner and repo are required", ErrInvalidRepo)
	}
	token, err := repositoryService.Tokens.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGitHubToken, err)
	}
	remote, err := repositoryService.GitHub.GetRepository(ctx, token, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGitHubUpstream, err)
	}
	return repositoryService.RepositoryRepository.Upsert(ctx, organizationID, remote)
}

func (repositoryService *RepositoryService) ListRepos(ctx context.Context, organizationID uuid.UUID) ([]Repository, error) {
	return repositoryService.RepositoryRepository.ListForOrg(ctx, organizationID)
}

func (repositoryService *RepositoryService) GetRepo(ctx context.Context, organizationID uuid.UUID, id uuid.UUID) (*Repository, error) {
	repository, err := repositoryService.RepositoryRepository.FindForOrg(ctx, organizationID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w", ErrRepositoryNotFound)
		}
		return nil, err
	}
	return repository, nil
}

func (repositoryService *RepositoryService) GetByID(ctx context.Context, id uuid.UUID) (*Repository, error) {
	repository, err := repositoryService.RepositoryRepository.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w", ErrRepositoryNotFound)
		}
		return nil, err
	}
	return repository, nil
}

func (repositoryService *RepositoryService) ListContributors(ctx context.Context, organizationID uuid.UUID, userID uuid.UUID, id uuid.UUID, page int, perPage int) ([]github.Contributor, error) {
	repository, err := repositoryService.GetRepo(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}
	token, err := repositoryService.Tokens.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGitHubToken, err)
	}
	page, perPage = NormalizePage(page, perPage)
	contributors, err := repositoryService.GitHub.ListContributors(ctx, token, repository.Owner, repository.Name, page, perPage)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGitHubUpstream, err)
	}
	return contributors, nil
}

func (repositoryService *RepositoryService) ListIssues(ctx context.Context, organizationID uuid.UUID, userID uuid.UUID, id uuid.UUID, state string, page int, perPage int) ([]github.Issue, error) {
	if err := ValidateIssueState(state); err != nil {
		return nil, err
	}
	repository, err := repositoryService.GetRepo(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}
	token, err := repositoryService.Tokens.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGitHubToken, err)
	}
	page, perPage = NormalizePage(page, perPage)
	issues, err := repositoryService.GitHub.ListIssues(ctx, token, repository.Owner, repository.Name, state, page, perPage)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGitHubUpstream, err)
	}
	return issues, nil
}

func (repositoryService *RepositoryService) GetIssue(ctx context.Context, organizationID uuid.UUID, userID uuid.UUID, id uuid.UUID, issueNumber int) (*github.Issue, error) {
	if issueNumber <= 0 {
		return nil, fmt.Errorf("%w: issue number is required", ErrInvalidRepo)
	}
	repository, err := repositoryService.GetRepo(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}
	token, err := repositoryService.Tokens.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGitHubToken, err)
	}
	issue, err := repositoryService.GitHub.GetIssue(ctx, token, repository.Owner, repository.Name, issueNumber)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGitHubUpstream, err)
	}
	return issue, nil
}
