package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

var (
	ErrUnauthorized = errors.New("github authentication failed")
	ErrNotFound     = errors.New("github resource not found")
	ErrRateLimited  = errors.New("github rate limit exceeded")
	ErrGitHubServer = errors.New("github api error")
	ErrDecodeBody   = errors.New("failed to decode github response")
)

type Repository struct {
	ID            int64
	Owner         string
	Name          string
	FullName      string
	Description   string
	DefaultBranch string
	HTMLURL       string
	Private       bool
}

type Contributor struct {
	Login         string `json:"login"`
	AvatarURL     string `json:"avatar_url"`
	Contributions int    `json:"contributions"`
}

type Issue struct {
	Number    int
	Title     string
	Body      string
	State     string
	Author    string
	Labels    []string
	CreatedAt time.Time
	UpdatedAt time.Time
	HTMLURL   string
}

type IssueComment struct {
	ID        int64
	Author    string
	Body      string
	CreatedAt time.Time
}

type repositoryResponse struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Description   string `json:"description"`
	DefaultBranch string `json:"default_branch"`
	HTMLURL       string `json:"html_url"`
	Private       bool   `json:"private"`
	Owner         struct {
		Login string `json:"login"`
	} `json:"owner"`
}

type issueResponse struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	HTMLURL     string    `json:"html_url"`
	PullRequest *struct {
		URL string `json:"url"`
	} `json:"pull_request"`
	User struct {
		Login string `json:"login"`
	} `json:"user"`
}

type issueCommentResponse struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) Client {
	return Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (client *Client) GetRepository(ctx context.Context, token string, owner string, repo string) (*Repository, error) {
	var raw repositoryResponse
	if err := client.get(ctx, token, "/repos/"+owner+"/"+repo, map[string]string{}, &raw); err != nil {
		return nil, err
	}
	return &Repository{
		ID:            raw.ID,
		Owner:         raw.Owner.Login,
		Name:          raw.Name,
		FullName:      raw.FullName,
		Description:   raw.Description,
		DefaultBranch: raw.DefaultBranch,
		HTMLURL:       raw.HTMLURL,
		Private:       raw.Private,
	}, nil
}

func (client *Client) ListContributors(ctx context.Context, token string, owner string, repo string, page int, perPage int) ([]Contributor, error) {
	var contributors []Contributor
	err := client.get(ctx, token, "/repos/"+owner+"/"+repo+"/contributors", map[string]string{
		"page":     strconv.Itoa(page),
		"per_page": strconv.Itoa(perPage),
	}, &contributors)
	if err != nil {
		return nil, err
	}
	return contributors, nil
}

func (client *Client) ListIssues(ctx context.Context, token string, owner string, repo string, state string, page int, perPage int) ([]Issue, error) {
	var raw []issueResponse
	err := client.get(ctx, token, "/repos/"+owner+"/"+repo+"/issues", map[string]string{
		"state":    state,
		"page":     strconv.Itoa(page),
		"per_page": strconv.Itoa(perPage),
	}, &raw)
	if err != nil {
		return nil, err
	}
	issues := make([]Issue, 0, len(raw))
	for _, item := range raw {
		if item.PullRequest != nil {
			continue
		}
		labels := make([]string, 0, len(item.Labels))
		for _, label := range item.Labels {
			labels = append(labels, label.Name)
		}
		issues = append(issues, Issue{
			Number:    item.Number,
			Title:     item.Title,
			Body:      item.Body,
			State:     item.State,
			Author:    item.User.Login,
			Labels:    labels,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
			HTMLURL:   item.HTMLURL,
		})
	}
	return issues, nil
}

func (client *Client) GetIssue(ctx context.Context, token string, owner string, repo string, issueNumber int) (*Issue, error) {
	var raw issueResponse
	if err := client.get(ctx, token, "/repos/"+owner+"/"+repo+"/issues/"+strconv.Itoa(issueNumber), map[string]string{}, &raw); err != nil {
		return nil, err
	}
	if raw.PullRequest != nil {
		return nil, fmt.Errorf("%w: #%d is a pull request", ErrNotFound, issueNumber)
	}
	labels := make([]string, 0, len(raw.Labels))
	for _, label := range raw.Labels {
		labels = append(labels, label.Name)
	}
	return &Issue{
		Number:    raw.Number,
		Title:     raw.Title,
		Body:      raw.Body,
		State:     raw.State,
		Author:    raw.User.Login,
		Labels:    labels,
		CreatedAt: raw.CreatedAt,
		UpdatedAt: raw.UpdatedAt,
		HTMLURL:   raw.HTMLURL,
	}, nil
}

func (client *Client) ListIssueComments(ctx context.Context, token string, owner string, repo string, issueNumber int) ([]IssueComment, error) {
	var raw []issueCommentResponse
	err := client.get(ctx, token, "/repos/"+owner+"/"+repo+"/issues/"+strconv.Itoa(issueNumber)+"/comments", map[string]string{
		"per_page":  "10",
		"sort":      "created",
		"direction": "asc",
	}, &raw)
	if err != nil {
		return nil, err
	}
	comments := make([]IssueComment, 0, len(raw))
	for _, item := range raw {
		comments = append(comments, IssueComment{
			ID:        item.ID,
			Author:    item.User.Login,
			Body:      item.Body,
			CreatedAt: item.CreatedAt,
		})
	}
	return comments, nil
}

type CreatedIssueComment struct {
	ID      int64
	HTMLURL string
	Body    string
}

type createdIssueCommentResponse struct {
	ID      int64  `json:"id"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
}

func (client *Client) CreateIssueComment(ctx context.Context, token string, owner string, repo string, issueNumber int, body string) (*CreatedIssueComment, error) {
	payload, err := json.Marshal(map[string]string{"body": body})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUserRetrieval, err)
	}
	var raw createdIssueCommentResponse
	if err := client.post(ctx, token, "/repos/"+owner+"/"+repo+"/issues/"+strconv.Itoa(issueNumber)+"/comments", payload, &raw); err != nil {
		return nil, err
	}
	if raw.ID == 0 {
		return nil, fmt.Errorf("%w: missing comment id", ErrDecodeBody)
	}
	return &CreatedIssueComment{
		ID:      raw.ID,
		HTMLURL: raw.HTMLURL,
		Body:    raw.Body,
	}, nil
}

func (client *Client) get(ctx context.Context, token string, path string, query map[string]string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, client.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUserRetrieval, err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "IssueRefiner-App")
	values := req.URL.Query()
	for key, value := range query {
		values.Set(key, value)
	}
	req.URL.RawQuery = values.Encode()

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUserRetrieval, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return statusError(resp)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("%w: %w", ErrDecodeBody, err)
	}
	return nil
}

func (client *Client) post(ctx context.Context, token string, path string, payload []byte, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUserRetrieval, err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "IssueRefiner-App")
	resp, err := client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUserRetrieval, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return statusError(resp)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("%w: %w", ErrDecodeBody, err)
	}
	return nil
}

func statusError(resp *http.Response) error {
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return fmt.Errorf("%w: status %d", ErrUnauthorized, resp.StatusCode)
	case resp.StatusCode == http.StatusForbidden && resp.Header.Get("X-RateLimit-Remaining") == "0":
		return fmt.Errorf("%w: status %d", ErrRateLimited, resp.StatusCode)
	case resp.StatusCode == http.StatusForbidden:
		return fmt.Errorf("%w: status %d", ErrUnauthorized, resp.StatusCode)
	case resp.StatusCode == http.StatusNotFound:
		return fmt.Errorf("%w: status %d", ErrNotFound, resp.StatusCode)
	case resp.StatusCode == http.StatusTooManyRequests:
		return fmt.Errorf("%w: status %d", ErrRateLimited, resp.StatusCode)
	case resp.StatusCode >= 500:
		return fmt.Errorf("%w: status %d", ErrGitHubServer, resp.StatusCode)
	default:
		return fmt.Errorf("%w: %d", ErrDiffStatus, resp.StatusCode)
	}
}
