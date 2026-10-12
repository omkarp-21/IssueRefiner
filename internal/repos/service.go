package repos

import (
	"context"
	"errors"
	"fmt"
	"issue-refiner/internal/github"
	"issue-refiner/internal/tokens"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidRepo      = errors.New("invalid repository reference")
	ErrGitHubToken      = errors.New("github account not connected")
	ErrTokenNeedsReauth = errors.New("github token needs reauthorization")
	ErrGitHubUpstream   = errors.New("github request failed")
)

type GitHubClient interface {
	GetRepository(ctx context.Context, token string, owner string, repo string) (*github.Repository, error)
	GetIssue(ctx context.Context, token string, owner string, repo string, issueNumber int) (*github.Issue, error)
	ListIssueComments(ctx context.Context, token string, owner string, repo string, issueNumber int) ([]github.IssueComment, error)
	CreateIssueComment(ctx context.Context, token string, owner string, repo string, issueNumber int, body string) (*github.CreatedIssueComment, error)
	ListContributors(ctx context.Context, token string, owner string, repo string, page int, perPage int) ([]github.Contributor, error)
	ListIssues(ctx context.Context, token string, owner string, repo string, state string, page int, perPage int, query string) ([]github.Issue, bool, error)
}

type TokenStore interface {
	Get(ctx context.Context, userID uuid.UUID) (string, error)
}

func userToken(store TokenStore, ctx context.Context, userID uuid.UUID) (string, error) {
	token, err := store.Get(ctx, userID)
	if err != nil {
		if errors.Is(err, tokens.ErrLegacyToken) || errors.Is(err, tokens.ErrTokenCorrupt) {
			return "", fmt.Errorf("%w: %w", ErrTokenNeedsReauth, err)
		}
		return "", fmt.Errorf("%w: %w", ErrGitHubToken, err)
	}
	return token, nil
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
	return NormalizePageWithDefault(page, perPage, 30)
}

// NormalizePageWithDefault clamps page to >= 1 and perPage to 1..100,
// substituting def when perPage is missing or non-positive. The issues
// listing uses IssueDefaultPerPage; other listings keep their own default.
func NormalizePageWithDefault(page int, perPage int, def int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = def
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}

// IssueDefaultPerPage is the issues-listing page size: the collector fills
// each page with this many real issues, skipping pull requests.
const IssueDefaultPerPage = 7

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
	token, err := userToken(repositoryService.Tokens, ctx, userID)
	if err != nil {
		return nil, err
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
	token, err := userToken(repositoryService.Tokens, ctx, userID)
	if err != nil {
		return nil, err
	}
	page, perPage = NormalizePage(page, perPage)
	contributors, err := repositoryService.GitHub.ListContributors(ctx, token, repository.Owner, repository.Name, page, perPage)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGitHubUpstream, err)
	}
	return contributors, nil
}

func (repositoryService *RepositoryService) ListIssues(ctx context.Context, organizationID uuid.UUID, userID uuid.UUID, id uuid.UUID, state string, page int, perPage int, query string) ([]github.Issue, bool, error) {
	if err := ValidateIssueState(state); err != nil {
		return nil, false, err
	}
	repository, err := repositoryService.GetRepo(ctx, organizationID, id)
	if err != nil {
		return nil, false, err
	}
	token, err := userToken(repositoryService.Tokens, ctx, userID)
	if err != nil {
		return nil, false, err
	}
	page, perPage = NormalizePageWithDefault(page, perPage, IssueDefaultPerPage)
	issues, hasMore, err := repositoryService.GitHub.ListIssues(ctx, token, repository.Owner, repository.Name, state, page, perPage, query)
	if err != nil {
		return nil, false, fmt.Errorf("%w: %w", ErrGitHubUpstream, err)
	}
	return issues, hasMore, nil
}

func (repositoryService *RepositoryService) GetIssue(ctx context.Context, organizationID uuid.UUID, userID uuid.UUID, id uuid.UUID, issueNumber int) (*github.Issue, error) {
	if issueNumber <= 0 {
		return nil, fmt.Errorf("%w: issue number is required", ErrInvalidRepo)
	}
	repository, err := repositoryService.GetRepo(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}
	token, err := userToken(repositoryService.Tokens, ctx, userID)
	if err != nil {
		return nil, err
	}
	issue, err := repositoryService.GitHub.GetIssue(ctx, token, repository.Owner, repository.Name, issueNumber)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGitHubUpstream, err)
	}
	return issue, nil
}

func (repositoryService *RepositoryService) GetIssueComments(ctx context.Context, organizationID uuid.UUID, userID uuid.UUID, id uuid.UUID, issueNumber int) ([]github.IssueComment, error) {
	if issueNumber <= 0 {
		return nil, fmt.Errorf("%w: issue number is required", ErrInvalidRepo)
	}
	repository, err := repositoryService.GetRepo(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}
	token, err := userToken(repositoryService.Tokens, ctx, userID)
	if err != nil {
		return nil, err
	}
	comments, err := repositoryService.GitHub.ListIssueComments(ctx, token, repository.Owner, repository.Name, issueNumber)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGitHubUpstream, err)
	}
	return comments, nil
}

func (repositoryService *RepositoryService) PostIssueComment(ctx context.Context, organizationID uuid.UUID, userID uuid.UUID, id uuid.UUID, issueNumber int, body string) (*github.CreatedIssueComment, error) {
	if issueNumber <= 0 {
		return nil, fmt.Errorf("%w: issue number is required", ErrInvalidRepo)
	}
	repository, err := repositoryService.GetRepo(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}
	token, err := userToken(repositoryService.Tokens, ctx, userID)
	if err != nil {
		return nil, err
	}
	comment, err := repositoryService.GitHub.CreateIssueComment(ctx, token, repository.Owner, repository.Name, issueNumber, body)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGitHubUpstream, err)
	}
	return comment, nil
}
