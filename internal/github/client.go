package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

var (
	ErrUserRetrieval    = errors.New("failed user retrieval")
	ErrDiffStatus       = errors.New("github api returned non-200 status")
	ErrDecodeGitHubUser = errors.New("failed to decode github response")
)

type GitHubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

func FetchGitHubUser(ctx context.Context, token *oauth2.Token) (*GitHubUser, error) {
	reqURL := "https://api.github.com/user"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "IssueRefiner-App")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUserRetrieval, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %v", ErrDiffStatus, resp.StatusCode)
	}
	var githubUser GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&githubUser); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDecodeGitHubUser, err)
	}
	return &githubUser, nil
}
