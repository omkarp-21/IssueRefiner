package providers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func openAITestServer(response string, status int, headers map[string]string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for key, value := range headers {
			w.Header().Set(key, value)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	}))
}

func testInput() Input {
	return Input{Owner: "octo", Repo: "demo", IssueNumber: 7, Title: "Crash", Body: "body"}
}

const geminiModelListFixture = `{"models":[
	{"name":"models/gemini-2.5-flash","supportedGenerationMethods":["generateContent","countTokens"]},
	{"name":"models/gemini-9-flash","supportedGenerationMethods":["generateContent"]},
	{"name":"models/gemini-embedding-001","supportedGenerationMethods":["embedContent"]},
	{"name":"models/gemini-2.5-flash-image","supportedGenerationMethods":["generateContent"]},
	{"name":"models/veo-3.1-generate-preview","supportedGenerationMethods":["generateVideo"]},
	{"name":"models/mystery-model","supportedGenerationMethods":[]},
	{"name":"models/unnamed-model"}
]}`

func TestGeminiDiscoverModels(t *testing.T) {
	server := openAITestServer(geminiModelListFixture, http.StatusOK, nil)
	defer server.Close()
	provider := NewGeminiProvider(server.URL)
	models, err := provider.DiscoverModels(context.Background(), "key")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	want := map[string]bool{"gemini-2.5-flash": true, "gemini-9-flash": true, "gemini-2.5-flash-image": true}
	if len(models) != len(want) {
		t.Fatalf("expected %v, got %v", want, models)
	}
	for _, model := range models {
		if !want[model] {
			t.Fatalf("unexpected model %q in %v", model, models)
		}
	}
}

func TestGeminiDiscoverUnknownModelAccepted(t *testing.T) {
	server := openAITestServer(`{"models":[{"name":"models/gemini-9-flash","supportedGenerationMethods":["generateContent"]}]}`, http.StatusOK, nil)
	defer server.Close()
	provider := NewGeminiProvider(server.URL)
	models, err := provider.DiscoverModels(context.Background(), "key")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(models) != 1 || models[0] != "gemini-9-flash" {
		t.Fatalf("expected unknown model accepted without code change, got %v", models)
	}
}

func TestGeminiDiscoverErrors(t *testing.T) {
	cases := []struct {
		name     string
		status   int
		response string
		want     error
	}{
		{"invalid key", http.StatusUnauthorized, `{}`, ErrInvalidCredentials},
		{"rate limited", http.StatusTooManyRequests, `{}`, ErrRateLimited},
		{"server", http.StatusInternalServerError, `{}`, ErrProviderUnavailable},
	}
	for _, tc := range cases {
		server := openAITestServer(tc.response, tc.status, nil)
		provider := NewGeminiProvider(server.URL)
		_, err := provider.DiscoverModels(context.Background(), "key")
		server.Close()
		if !errors.Is(err, tc.want) {
			t.Fatalf("%s: expected %v, got %v", tc.name, tc.want, err)
		}
	}
}

func TestGeminiDiscoverMalformed(t *testing.T) {
	server := openAITestServer(`not json{{{`, http.StatusOK, nil)
	defer server.Close()
	provider := NewGeminiProvider(server.URL)
	if _, err := provider.DiscoverModels(context.Background(), "key"); err == nil {
		t.Fatalf("expected decode error")
	}
}

func TestGeminiSupportsModelPermissive(t *testing.T) {
	provider := NewGeminiProvider("http://example.com")
	if !provider.SupportsModel("gemini-9-flash") {
		t.Fatalf("expected unknown model name to pass the syntactic guard")
	}
	if provider.SupportsModel("") {
		t.Fatalf("expected empty model to fail the syntactic guard")
	}
}

func TestOpenAIValidateSuccess(t *testing.T) {
	server := openAITestServer(`{"data":[{"id":"gpt-4o-mini"},{"id":"gpt-4o"}]}`, http.StatusOK, nil)
	defer server.Close()
	provider := NewOpenAIProvider(server.URL)
	if err := provider.ValidateCredentials(context.Background(), "key", "gpt-4o-mini"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestOpenAIValidateModelNotListed(t *testing.T) {
	server := openAITestServer(`{"data":[{"id":"gpt-4o"}]}`, http.StatusOK, nil)
	defer server.Close()
	provider := NewOpenAIProvider(server.URL)
	if err := provider.ValidateCredentials(context.Background(), "key", "gpt-4o-mini"); !errors.Is(err, ErrModelUnavailable) {
		t.Fatalf("expected ErrModelUnavailable, got %v", err)
	}
}

func TestOpenAIValidateErrors(t *testing.T) {
	cases := []struct {
		name     string
		status   int
		response string
		want     error
	}{
		{"invalid key", http.StatusUnauthorized, `{}`, ErrInvalidCredentials},
		{"quota", http.StatusTooManyRequests, `{"error":{"code":"insufficient_quota"}}`, ErrInsufficientQuota},
		{"rate limited", http.StatusTooManyRequests, `{"error":{"code":"rate_limit_exceeded"}}`, ErrRateLimited},
		{"server", http.StatusInternalServerError, `{}`, ErrProviderUnavailable},
		{"model missing", http.StatusNotFound, `{}`, ErrModelUnavailable},
	}
	for _, tc := range cases {
		server := openAITestServer(tc.response, tc.status, nil)
		provider := NewOpenAIProvider(server.URL)
		err := provider.ValidateCredentials(context.Background(), "key", "gpt-4o-mini")
		server.Close()
		if !errors.Is(err, tc.want) {
			t.Fatalf("%s: expected %v, got %v", tc.name, tc.want, err)
		}
	}
}

func TestOpenAIGenerateSuccess(t *testing.T) {
	var request map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&request)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"refined text"}}],"usage":{"prompt_tokens":120,"completion_tokens":45}}`))
	}))
	defer server.Close()
	provider := NewOpenAIProvider(server.URL)
	result, err := provider.Generate(context.Background(), "key", "gpt-4o-mini", testInput())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Text != "refined text" {
		t.Fatalf("unexpected result %q", result.Text)
	}
	if tokens, ok := request["max_completion_tokens"].(float64); !ok || tokens != 2000 {
		t.Fatalf("expected max_completion_tokens 2000, got %v", request["max_completion_tokens"])
	}
	if result.InputTokens != 120 || result.OutputTokens != 45 {
		t.Fatalf("expected usage 120/45, got %d/%d", result.InputTokens, result.OutputTokens)
	}
}

func TestOpenAIGenerateNoUsage(t *testing.T) {
	server := openAITestServer(`{"choices":[{"message":{"content":"refined text"}}]}`, http.StatusOK, nil)
	defer server.Close()
	provider := NewOpenAIProvider(server.URL)
	result, err := provider.Generate(context.Background(), "key", "gpt-4o-mini", testInput())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.InputTokens != 0 || result.OutputTokens != 0 {
		t.Fatalf("expected zero usage when omitted, got %d/%d", result.InputTokens, result.OutputTokens)
	}
}

func TestOpenAIGenerateMalformed(t *testing.T) {
	server := openAITestServer(`{"choices":[]}`, http.StatusOK, nil)
	defer server.Close()
	provider := NewOpenAIProvider(server.URL)
	if _, err := provider.Generate(context.Background(), "key", "gpt-4o-mini", testInput()); !errors.Is(err, ErrUnknownProvider) {
		t.Fatalf("expected ErrUnknownProvider, got %v", err)
	}
}

func TestOpenAICancelled(t *testing.T) {
	server := openAITestServer(`{}`, http.StatusOK, nil)
	defer server.Close()
	provider := NewOpenAIProvider(server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := provider.Generate(ctx, "key", "gpt-4o-mini", testInput()); !errors.Is(err, ErrProviderTimeout) {
		t.Fatalf("expected ErrProviderTimeout, got %v", err)
	}
}

func TestGeminiValidateSuccess(t *testing.T) {
	server := openAITestServer(`{"name":"models/gemini-2.5-flash"}`, http.StatusOK, nil)
	defer server.Close()
	provider := NewGeminiProvider(server.URL)
	if err := provider.ValidateCredentials(context.Background(), "key", "gemini-2.5-flash"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestGeminiValidateErrors(t *testing.T) {
	cases := []struct {
		name     string
		status   int
		response string
		want     error
	}{
		{"bad key", http.StatusBadRequest, `{"error":{"status":"INVALID_ARGUMENT","message":"API key not valid"}}`, ErrInvalidCredentials},
		{"not found", http.StatusNotFound, `{}`, ErrModelUnavailable},
		{"quota", http.StatusTooManyRequests, `{"error":{"status":"RESOURCE_EXHAUSTED","message":"Quota exceeded"}}`, ErrInsufficientQuota},
		{"rate limited", http.StatusTooManyRequests, `{"error":{"status":"RESOURCE_EXHAUSTED","message":"Rate exceeded"}}`, ErrRateLimited},
		{"server", http.StatusBadGateway, `{}`, ErrProviderUnavailable},
	}
	for _, tc := range cases {
		server := openAITestServer(tc.response, tc.status, nil)
		provider := NewGeminiProvider(server.URL)
		err := provider.ValidateCredentials(context.Background(), "key", "gemini-2.5-flash")
		server.Close()
		if !errors.Is(err, tc.want) {
			t.Fatalf("%s: expected %v, got %v", tc.name, tc.want, err)
		}
	}
}

func TestGeminiGenerateSuccess(t *testing.T) {
	var request map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&request)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"refined by gemini"}]}}],"usageMetadata":{"promptTokenCount":200,"candidatesTokenCount":60}}`))
	}))
	defer server.Close()
	provider := NewGeminiProvider(server.URL)
	result, err := provider.Generate(context.Background(), "key", "gemini-2.5-flash", testInput())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Text != "refined by gemini" {
		t.Fatalf("unexpected result %q", result.Text)
	}
	config, ok := request["generationConfig"].(map[string]any)
	if !ok {
		t.Fatalf("expected generationConfig, got %v", request["generationConfig"])
	}
	if tokens, ok := config["maxOutputTokens"].(float64); !ok || tokens != 2000 {
		t.Fatalf("expected maxOutputTokens 2000, got %v", config["maxOutputTokens"])
	}
	if mime, ok := config["responseMimeType"].(string); !ok || mime != "application/json" {
		t.Fatalf("expected responseMimeType application/json, got %v", config["responseMimeType"])
	}
	if result.InputTokens != 200 || result.OutputTokens != 60 {
		t.Fatalf("expected usage 200/60, got %d/%d", result.InputTokens, result.OutputTokens)
	}
}

func TestGeminiGenerateNoUsage(t *testing.T) {
	server := openAITestServer(`{"candidates":[{"content":{"parts":[{"text":"refined by gemini"}]}}]}`, http.StatusOK, nil)
	defer server.Close()
	provider := NewGeminiProvider(server.URL)
	result, err := provider.Generate(context.Background(), "key", "gemini-2.5-flash", testInput())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.InputTokens != 0 || result.OutputTokens != 0 {
		t.Fatalf("expected zero usage when omitted, got %d/%d", result.InputTokens, result.OutputTokens)
	}
}

func TestGeminiGenerateMalformed(t *testing.T) {
	server := openAITestServer(`{"candidates":[]}`, http.StatusOK, nil)
	defer server.Close()
	provider := NewGeminiProvider(server.URL)
	if _, err := provider.Generate(context.Background(), "key", "gemini-2.5-flash", testInput()); !errors.Is(err, ErrUnknownProvider) {
		t.Fatalf("expected ErrUnknownProvider, got %v", err)
	}
}

func TestMockProvider(t *testing.T) {
	provider := NewMockProvider()
	result, err := provider.Generate(context.Background(), "", "mock-test", testInput())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Text == "" {
		t.Fatalf("expected non-empty result")
	}
	cases := []struct {
		model string
		want  error
	}{
		{"mock-quota", ErrInsufficientQuota},
		{"mock-rate", ErrRateLimited},
		{"mock-invalid", ErrInvalidCredentials},
		{"mock-unavailable", ErrProviderUnavailable},
		{"mock-bogus", ErrModelUnavailable},
	}
	for _, tc := range cases {
		if _, err := provider.Generate(context.Background(), "", tc.model, testInput()); !errors.Is(err, tc.want) {
			t.Fatalf("%s: expected %v, got %v", tc.model, tc.want, err)
		}
	}
	if err := provider.ValidateCredentials(context.Background(), "", "mock-test"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := provider.ValidateCredentials(context.Background(), "", "mock-invalid"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestMockSlowRespectsContext(t *testing.T) {
	provider := NewMockProvider()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := provider.Generate(ctx, "", "mock-slow", testInput()); !errors.Is(err, ErrProviderTimeout) {
		t.Fatalf("expected ErrProviderTimeout, got %v", err)
	}
}

func TestMockInvestigateModelsSupported(t *testing.T) {
	provider := NewMockProvider()
	for _, model := range []string{"mock-investigate-ready", "mock-investigate-needs-info", "mock-investigate-malformed"} {
		if !provider.SupportsModel(model) {
			t.Fatalf("expected %q to be supported", model)
		}
		if err := provider.ValidateCredentials(context.Background(), "", model); err != nil {
			t.Fatalf("expected %q to validate, got %v", model, err)
		}
	}
}
