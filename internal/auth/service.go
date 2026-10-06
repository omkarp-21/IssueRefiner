package auth

import (
	"context"
	"errors"
	"fmt"
	"issue-refiner/internal/github"
	"issue-refiner/internal/tokens"
	"issue-refiner/internal/users"
)

var (
	errAuthTimeout   = errors.New("token exchange timed out")
	errTokenFailed   = errors.New("token exchange failed")
	errStateMismatch = errors.New("state mismatch")
	errPersistUser   = errors.New("failed to persist user")
)

type AuthService struct {
	GitHubOAuth  *github.GitHubOAuth
	UserService  *users.UserService
	TokenService *tokens.TokenService
}

func NewAuthService(githubOAuth *github.GitHubOAuth, userService *users.UserService, tokenService *tokens.TokenService) AuthService {
	return AuthService{GitHubOAuth: githubOAuth, UserService: userService, TokenService: tokenService}
}

func (authService *AuthService) AuthenticateGitHub(ctx context.Context, code string, state string, expectedState string) (*users.User, error) {
	if state == "" || expectedState == "" || state != expectedState {
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
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errPersistUser, err)
	}
	if err := authService.TokenService.Save(ctx, user.ID, token.AccessToken); err != nil {
		return nil, fmt.Errorf("%w: %w", errPersistUser, err)
	}

	return user, nil
}
