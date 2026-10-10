package github

import (
	"issue-refiner/internal/config"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type GitHubOAuth struct {
	GithubOAuthConfig oauth2.Config
}

func NewOAuth(cfg config.GitHubConfig) GitHubOAuth {
	var githubOAuth GitHubOAuth
	githubOAuth.GithubOAuthConfig = oauth2.Config{
		RedirectURL:  cfg.GitHubCallbackURL,
		ClientID:     cfg.GitHubClientID,
		ClientSecret: cfg.GitHubClientSecret,
		Endpoint:     github.Endpoint,
		// public_repo grants read/write access to public repositories,
		// which is the narrowest classic scope permitting issue comments.
		// Tokens issued before this scope was requested keep their old
		// scopes; those users must complete the OAuth flow again.
		Scopes: []string{"public_repo"},
	}
	return githubOAuth
}
