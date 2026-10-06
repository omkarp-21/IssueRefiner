package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func testServer(response string, status int, headers map[string]string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for key, value := range headers {
			w.Header().Set(key, value)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	}))
}

func TestGetRepository(t *testing.T) {
	server := testServer(`{"id":1296269,"name":"Hello-World","full_name":"octocat/Hello-World","owner":{"login":"octocat"},"description":"My first repo","default_branch":"main","html_url":"https://github.com/octocat/Hello-World","private":false}`, http.StatusOK, nil)
	defer server.Close()
	client := NewClient(server.URL)
	repository, err := client.GetRepository(context.Background(), "token", "octocat", "Hello-World")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repository.ID != 1296269 || repository.Owner != "octocat" || repository.Name != "Hello-World" {
		t.Fatalf("unexpected repository: %+v", repository)
	}
	if repository.FullName != "octocat/Hello-World" || repository.Description != "My first repo" {
		t.Fatalf("unexpected repository fields: %+v", repository)
	}
	if repository.DefaultBranch != "main" || repository.HTMLURL != "https://github.com/octocat/Hello-World" {
		t.Fatalf("unexpected repository fields: %+v", repository)
	}
	if repository.Private {
		t.Fatalf("expected public repository")
	}
}

func TestListContributors(t *testing.T) {
	server := testServer(`[{"login":"octocat","avatar_url":"https://avatars/x","contributions":143},{"login":"hubot","avatar_url":"https://avatars/y","contributions":97}]`, http.StatusOK, nil)
	defer server.Close()
	client := NewClient(server.URL)
	contributors, err := client.ListContributors(context.Background(), "token", "o", "r", 1, 30)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(contributors) != 2 || contributors[0].Login != "octocat" || contributors[0].Contributions != 143 {
		t.Fatalf("unexpected contributors: %+v", contributors)
	}
}

func TestListIssuesFiltersPullRequests(t *testing.T) {
	server := testServer(`[
		{"number":421,"title":"Real issue","body":"body","state":"open","labels":[{"name":"bug"}],"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-02T00:00:00Z","html_url":"https://github.com/o/r/issues/421","user":{"login":"alice"}},
		{"number":420,"title":"A PR","body":"","state":"open","labels":[],"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z","html_url":"https://github.com/o/r/pull/420","user":{"login":"bob"},"pull_request":{"url":"https://api.github.com/x"}}
	]`, http.StatusOK, nil)
	defer server.Close()
	client := NewClient(server.URL)
	issues, err := client.ListIssues(context.Background(), "token", "o", "r", "open", 1, 30)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected pull requests filtered, got %+v", issues)
	}
	if issues[0].Number != 421 || issues[0].Author != "alice" || len(issues[0].Labels) != 1 || issues[0].Labels[0] != "bug" {
		t.Fatalf("unexpected issue: %+v", issues[0])
	}
}

func TestClientStatusErrors(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		headers map[string]string
		want    error
	}{
		{"unauthorized", http.StatusUnauthorized, nil, ErrUnauthorized},
		{"forbidden", http.StatusForbidden, nil, ErrUnauthorized},
		{"rate limited forbidden", http.StatusForbidden, map[string]string{"X-RateLimit-Remaining": "0"}, ErrRateLimited},
		{"too many requests", http.StatusTooManyRequests, nil, ErrRateLimited},
		{"not found", http.StatusNotFound, nil, ErrNotFound},
		{"server error", http.StatusInternalServerError, nil, ErrGitHubServer},
	}
	for _, tc := range cases {
		server := testServer(`{}`, tc.status, tc.headers)
		client := NewClient(server.URL)
		_, err := client.GetRepository(context.Background(), "token", "o", "r")
		server.Close()
		if !errors.Is(err, tc.want) {
			t.Fatalf("%s: expected %v, got %v", tc.name, tc.want, err)
		}
	}
}

func TestClientCancelledContext(t *testing.T) {
	server := testServer(`{}`, http.StatusOK, nil)
	defer server.Close()
	client := NewClient(server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.GetRepository(ctx, "token", "o", "r")
	if err == nil {
		t.Fatalf("expected cancellation error")
	}
}

func TestClientTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	defer server.Close()
	client := NewClient(server.URL)
	client.httpClient.Timeout = 50 * time.Millisecond
	_, err := client.GetRepository(context.Background(), "token", "o", "r")
	if err == nil {
		t.Fatalf("expected timeout error")
	}
}
