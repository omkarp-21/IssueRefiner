package repos

import (
	"context"
	"errors"
	"testing"

	"issue-refiner/internal/github"

	"github.com/google/uuid"
)

type stubTokens struct {
	token string
	err   error
}

func (stub stubTokens) Get(ctx context.Context, userID uuid.UUID) (string, error) {
	return stub.token, stub.err
}

type stubGitHub struct {
	repository *github.Repository
	err        error
}

func (stub stubGitHub) GetRepository(ctx context.Context, token string, owner string, repo string) (*github.Repository, error) {
	return stub.repository, stub.err
}

func (stub stubGitHub) ListContributors(ctx context.Context, token string, owner string, repo string, page int, perPage int) ([]github.Contributor, error) {
	return nil, stub.err
}

func (stub stubGitHub) ListIssues(ctx context.Context, token string, owner string, repo string, state string, page int, perPage int) ([]github.Issue, error) {
	return nil, stub.err
}

func (stub stubGitHub) GetIssue(ctx context.Context, token string, owner string, repo string, issueNumber int) (*github.Issue, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	return &github.Issue{Number: issueNumber, Title: "T", Body: "B", State: "open"}, nil
}

func testService(tokens TokenStore, client GitHubClient) *RepositoryService {
	repository := NewRepositoryRepository(nil)
	service := NewRepositoryService(&repository, client, tokens)
	return &service
}

func TestNormalizePage(t *testing.T) {
	page, perPage := NormalizePage(2, 50)
	if page != 2 || perPage != 50 {
		t.Fatalf("expected 2/50, got %d/%d", page, perPage)
	}
	page, perPage = NormalizePage(0, 0)
	if page != 1 || perPage != 30 {
		t.Fatalf("expected defaults 1/30, got %d/%d", page, perPage)
	}
	_, perPage = NormalizePage(1, 500)
	if perPage != 100 {
		t.Fatalf("expected per_page clamped to 100, got %d", perPage)
	}
}

func TestValidateIssueState(t *testing.T) {
	for _, state := range []string{"open", "closed", "all"} {
		if err := ValidateIssueState(state); err != nil {
			t.Fatalf("expected %q to be valid, got %v", state, err)
		}
	}
	if err := ValidateIssueState("bogus"); !errors.Is(err, ErrInvalidRepo) {
		t.Fatalf("expected ErrInvalidRepo, got %v", err)
	}
}

func TestImportRepoValidation(t *testing.T) {
	service := testService(stubTokens{}, stubGitHub{})
	_, err := service.ImportRepo(context.Background(), uuid.New(), uuid.New(), "", "repo")
	if !errors.Is(err, ErrInvalidRepo) {
		t.Fatalf("expected ErrInvalidRepo, got %v", err)
	}
}

func TestImportRepoNoToken(t *testing.T) {
	service := testService(stubTokens{err: errors.New("no rows")}, stubGitHub{})
	_, err := service.ImportRepo(context.Background(), uuid.New(), uuid.New(), "octo", "demo")
	if !errors.Is(err, ErrGitHubToken) {
		t.Fatalf("expected ErrGitHubToken, got %v", err)
	}
}

func TestImportRepoUpstreamError(t *testing.T) {
	service := testService(stubTokens{token: "token"}, stubGitHub{err: github.ErrNotFound})
	_, err := service.ImportRepo(context.Background(), uuid.New(), uuid.New(), "octo", "demo")
	if !errors.Is(err, ErrGitHubUpstream) || !errors.Is(err, github.ErrNotFound) {
		t.Fatalf("expected wrapped ErrNotFound, got %v", err)
	}
}
