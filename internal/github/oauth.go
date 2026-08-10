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
		Scopes:       []string{},
	}
	return githubOAuth
}
