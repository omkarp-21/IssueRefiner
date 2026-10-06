package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var openAIModels = []string{"gpt-4o-mini", "gpt-4o", "gpt-4.1-mini"}

const openAIMaxOutputTokens = 2000

type OpenAIProvider struct {
	baseURL    string
	httpClient *http.Client
}

func NewOpenAIProvider(baseURL string) OpenAIProvider {
	return OpenAIProvider{baseURL: baseURL, httpClient: &http.Client{Timeout: 30 * time.Second}}
}

func (openAI OpenAIProvider) Type() string {
	return ProviderOpenAI
}

func (openAI OpenAIProvider) SupportsModel(model string) bool {
	for _, supported := range openAIModels {
		if model == supported {
			return true
		}
	}
	return false
}

type openAIModelList struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

type openAIErrorBody struct {
	Error struct {
		Code string `json:"code"`
	} `json:"error"`
}

func (openAI OpenAIProvider) ValidateCredentials(ctx context.Context, apiKey string, model string) error {
	if !openAI.SupportsModel(model) {
		return fmt.Errorf("%w: %s", ErrModelUnavailable, model)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openAI.baseURL+"/models", nil)
	if err != nil {
		return NewProviderError(ErrProviderUnavailable, err)
	}
	openAI.setHeaders(req, apiKey)
	resp, err := openAI.httpClient.Do(req)
	if err != nil {
		return openAI.requestError(ctx, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return openAI.statusError(resp)
	}
	var list openAIModelList
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return NewProviderError(ErrUnknownProvider, fmt.Errorf("decode models response"))
	}
	for _, item := range list.Data {
		if item.ID == model {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrModelUnavailable, model)
}

type openAIChatRequest struct {
	Model               string              `json:"model"`
	MaxCompletionTokens int                 `json:"max_completion_tokens"`
	Messages            []openAIChatMessage `json:"messages"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func (openAI OpenAIProvider) Generate(ctx context.Context, apiKey string, model string, input Input) (Result, error) {
	if !openAI.SupportsModel(model) {
		return Result{}, fmt.Errorf("%w: %s", ErrModelUnavailable, model)
	}
	system := "Refine the GitHub issue into a clear summary with reproduction steps."
	if input.SystemPrompt != "" {
		system = input.SystemPrompt
	}
	body, err := json.Marshal(openAIChatRequest{
		Model:               model,
		MaxCompletionTokens: openAIMaxOutputTokens,
		Messages: []openAIChatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: fmt.Sprintf("%s/%s#%d %s\n%s", input.Owner, input.Repo, input.IssueNumber, input.Title, input.Body)},
		},
	})
	if err != nil {
		return Result{}, NewProviderError(ErrRequestRejected, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAI.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Result{}, NewProviderError(ErrProviderUnavailable, err)
	}
	openAI.setHeaders(req, apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := openAI.httpClient.Do(req)
	if err != nil {
		return Result{}, openAI.requestError(ctx, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Result{}, openAI.statusError(resp)
	}
	var chat openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chat); err != nil {
		return Result{}, NewProviderError(ErrUnknownProvider, fmt.Errorf("decode chat response"))
	}
	if len(chat.Choices) == 0 || chat.Choices[0].Message.Content == "" {
		return Result{}, NewProviderError(ErrUnknownProvider, fmt.Errorf("empty chat response"))
	}
	return Result{
		Text:         chat.Choices[0].Message.Content,
		InputTokens:  chat.Usage.PromptTokens,
		OutputTokens: chat.Usage.CompletionTokens,
	}, nil
}

func (openAI OpenAIProvider) setHeaders(req *http.Request, apiKey string) {
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
}

func (openAI OpenAIProvider) requestError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return NewProviderError(ErrProviderTimeout, ctx.Err())
	}
	var urlErr interface{ Timeout() bool }
	if errors.As(err, &urlErr) && urlErr.Timeout() {
		return NewProviderError(ErrProviderTimeout, err)
	}
	return NewProviderError(ErrProviderUnavailable, err)
}

func (openAI OpenAIProvider) statusError(resp *http.Response) error {
	code := openAIErrorCode(resp)
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return NewProviderError(ErrInvalidCredentials, fmt.Errorf("status %d", resp.StatusCode))
	case resp.StatusCode == http.StatusTooManyRequests && code == "insufficient_quota":
		return NewProviderError(ErrInsufficientQuota, fmt.Errorf("status %d", resp.StatusCode))
	case resp.StatusCode == http.StatusTooManyRequests:
		return NewRateLimitedError(fmt.Errorf("status %d", resp.StatusCode), retryAfter(resp))
	case resp.StatusCode == http.StatusNotFound || code == "model_not_found":
		return NewProviderError(ErrModelUnavailable, fmt.Errorf("status %d", resp.StatusCode))
	case resp.StatusCode == http.StatusBadRequest:
		return NewProviderError(ErrRequestRejected, fmt.Errorf("status %d", resp.StatusCode))
	case resp.StatusCode >= 500:
		return NewProviderError(ErrProviderUnavailable, fmt.Errorf("status %d", resp.StatusCode))
	default:
		return NewProviderError(ErrUnknownProvider, fmt.Errorf("status %d", resp.StatusCode))
	}
}

func openAIErrorCode(resp *http.Response) string {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return ""
	}
	var parsed openAIErrorBody
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ""
	}
	return parsed.Error.Code
}

func retryAfter(resp *http.Response) time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(resp.Header.Get("Retry-After")))
	if err != nil || seconds < 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}
