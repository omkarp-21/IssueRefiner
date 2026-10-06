package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type GeminiProvider struct {
	baseURL    string
	httpClient *http.Client
}

func NewGeminiProvider(baseURL string) GeminiProvider {
	return GeminiProvider{baseURL: baseURL, httpClient: &http.Client{Timeout: 30 * time.Second}}
}

func (gemini GeminiProvider) Type() string {
	return ProviderGemini
}

func (gemini GeminiProvider) SupportsModel(model string) bool {
	// Gemini model availability is account-dependent and changes over time,
	// so this is only a syntactic guard. The provider_models rows, populated
	// from live discovery, are the authority on what may execute.
	return model != ""
}

type geminiErrorBody struct {
	Error struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	} `json:"error"`
}

const geminiMaxOutputTokens = 2000

type geminiGenerateRequest struct {
	Contents         []geminiContent  `json:"contents"`
	GenerationConfig geminiGeneration `json:"generationConfig"`
}

type geminiGeneration struct {
	MaxOutputTokens  int    `json:"maxOutputTokens"`
	ResponseMimeType string `json:"responseMimeType"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerateResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
}

type geminiModelList struct {
	Models []geminiModelEntry `json:"models"`
}

type geminiModelEntry struct {
	Name                       string   `json:"name"`
	SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
}

func (gemini GeminiProvider) DiscoverModels(ctx context.Context, apiKey string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, gemini.baseURL+"/models?pageSize=100", nil)
	if err != nil {
		return nil, NewProviderError(ErrProviderUnavailable, err)
	}
	gemini.setHeaders(req, apiKey)
	resp, err := gemini.httpClient.Do(req)
	if err != nil {
		return nil, gemini.requestError(ctx, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, gemini.statusError(resp)
	}
	var list geminiModelList
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, NewProviderError(ErrUnknownProvider, fmt.Errorf("decode models response"))
	}
	models := []string{}
	for _, entry := range list.Models {
		if !supportsGeneration(entry.SupportedGenerationMethods) {
			continue
		}
		if name := strings.TrimPrefix(entry.Name, "models/"); name != "" {
			models = append(models, name)
		}
	}
	return models, nil
}

func supportsGeneration(methods []string) bool {
	for _, method := range methods {
		if method == "generateContent" {
			return true
		}
	}
	return false
}

func (gemini GeminiProvider) ValidateCredentials(ctx context.Context, apiKey string, model string) error {
	if !gemini.SupportsModel(model) {
		return fmt.Errorf("%w: %s", ErrModelUnavailable, model)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, gemini.baseURL+"/models/"+model, nil)
	if err != nil {
		return NewProviderError(ErrProviderUnavailable, err)
	}
	gemini.setHeaders(req, apiKey)
	resp, err := gemini.httpClient.Do(req)
	if err != nil {
		return gemini.requestError(ctx, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return gemini.statusError(resp)
	}
	return nil
}

func (gemini GeminiProvider) Generate(ctx context.Context, apiKey string, model string, input Input) (Result, error) {
	if !gemini.SupportsModel(model) {
		return Result{}, fmt.Errorf("%w: %s", ErrModelUnavailable, model)
	}
	userText := fmt.Sprintf("Refine this GitHub issue into a clear summary with reproduction steps: %s/%s#%d %s\n%s", input.Owner, input.Repo, input.IssueNumber, input.Title, input.Body)
	if input.SystemPrompt != "" {
		userText = input.SystemPrompt + "\n\n" + userText
	}
	body, err := json.Marshal(geminiGenerateRequest{
		Contents:         []geminiContent{{Parts: []geminiPart{{Text: userText}}}},
		GenerationConfig: geminiGeneration{MaxOutputTokens: geminiMaxOutputTokens, ResponseMimeType: "application/json"},
	})
	if err != nil {
		return Result{}, NewProviderError(ErrRequestRejected, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, gemini.baseURL+"/models/"+model+":generateContent", bytes.NewReader(body))
	if err != nil {
		return Result{}, NewProviderError(ErrProviderUnavailable, err)
	}
	gemini.setHeaders(req, apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := gemini.httpClient.Do(req)
	if err != nil {
		return Result{}, gemini.requestError(ctx, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Result{}, gemini.statusError(resp)
	}
	var generated geminiGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&generated); err != nil {
		return Result{}, NewProviderError(ErrUnknownProvider, fmt.Errorf("decode generate response"))
	}
	if len(generated.Candidates) == 0 || len(generated.Candidates[0].Content.Parts) == 0 || generated.Candidates[0].Content.Parts[0].Text == "" {
		return Result{}, NewProviderError(ErrUnknownProvider, fmt.Errorf("empty generate response"))
	}
	return Result{
		Text:         generated.Candidates[0].Content.Parts[0].Text,
		InputTokens:  generated.UsageMetadata.PromptTokenCount,
		OutputTokens: generated.UsageMetadata.CandidatesTokenCount,
	}, nil
}

func (gemini GeminiProvider) setHeaders(req *http.Request, apiKey string) {
	req.Header.Set("x-goog-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")
}

func (gemini GeminiProvider) requestError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return NewProviderError(ErrProviderTimeout, ctx.Err())
	}
	var urlErr interface{ Timeout() bool }
	if errors.As(err, &urlErr) && urlErr.Timeout() {
		return NewProviderError(ErrProviderTimeout, err)
	}
	return NewProviderError(ErrProviderUnavailable, err)
}

func (gemini GeminiProvider) statusError(resp *http.Response) error {
	status, message := geminiErrorDetails(resp)
	switch {
	case resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		if status == "NOT_FOUND" {
			return NewProviderError(ErrModelUnavailable, fmt.Errorf("status %d", resp.StatusCode))
		}
		return NewProviderError(ErrInvalidCredentials, fmt.Errorf("status %d", resp.StatusCode))
	case resp.StatusCode == http.StatusNotFound:
		return NewProviderError(ErrModelUnavailable, fmt.Errorf("status %d", resp.StatusCode))
	case resp.StatusCode == http.StatusTooManyRequests:
		if strings.Contains(strings.ToLower(message), "quota") {
			return NewProviderError(ErrInsufficientQuota, fmt.Errorf("status %d", resp.StatusCode))
		}
		return NewRateLimitedError(fmt.Errorf("status %d", resp.StatusCode), retryAfter(resp))
	case resp.StatusCode >= 500:
		return NewProviderError(ErrProviderUnavailable, fmt.Errorf("status %d", resp.StatusCode))
	default:
		return NewProviderError(ErrUnknownProvider, fmt.Errorf("status %d", resp.StatusCode))
	}
}

func geminiErrorDetails(resp *http.Response) (string, string) {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return "", ""
	}
	var parsed geminiErrorBody
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", ""
	}
	return parsed.Error.Status, parsed.Error.Message
}
