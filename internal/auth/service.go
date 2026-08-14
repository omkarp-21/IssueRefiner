package auth

import (
	"context"
	"errors"
	"fmt"
	"issue-refiner/internal/github"
	"issue-refiner/internal/users"
)

var (
	errAuthTimeout   = errors.New("token exchange timed out")
	errTokenFailed   = errors.New("token exchange failed")
	errStateMismatch = errors.New("state mismatch")
)

type AuthService struct {
	GitHubOAuth *github.GitHubOAuth
	UserService *users.UserService
}

func NewAuthService(githubOAuth *github.GitHubOAuth, userService *users.UserService) AuthService {
	return AuthService{GitHubOAuth: githubOAuth, UserService: userService}
}

func (authService *AuthService) AuthenticateGitHub(ctx context.Context, code string, state string) (*users.User, error) {
	if state != "randomstate" {
		return nil, errStateMismatch
	}

	token, err := authService.GitHubOAuth.GithubOAuthConfig.Exchange(ctx, code)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w: %w", errAuthTimeout, err)
		}
		return nil, fmt.Errorf("%w: %w", errTokenFailed, err)
	}

	githubUser, err := github.FetchGitHubUser(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", github.ErrDecodeGitHubUser, err)
	}
	user, err := authService.UserService.GetOrCreateFromGitHub(ctx, githubUser)

	return user, nil
}
