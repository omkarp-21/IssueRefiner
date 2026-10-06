package providers

import (
	"errors"
	"testing"
	"time"
)

func TestValidateProviderType(t *testing.T) {
	for _, providerType := range []string{ProviderOpenAI, ProviderGemini, ProviderMock} {
		if err := ValidateProviderType(providerType); err != nil {
			t.Fatalf("expected %q to be valid, got %v", providerType, err)
		}
	}
	for _, providerType := range []string{"", "anthropic", "OPENAI", "open ai"} {
		if err := ValidateProviderType(providerType); !errors.Is(err, ErrUnknownProvider) {
			t.Fatalf("expected ErrUnknownProvider for %q, got %v", providerType, err)
		}
	}
}

func TestRetryable(t *testing.T) {
	retryable := []error{ErrRateLimited, ErrProviderUnavailable, ErrProviderTimeout}
	for _, err := range retryable {
		if !Retryable(NewProviderError(err, errors.New("cause"))) {
			t.Fatalf("expected %v to be retryable", err)
		}
	}
	permanent := []error{ErrInvalidCredentials, ErrInsufficientQuota, ErrModelUnavailable, ErrRequestRejected, ErrUnknownProvider}
	for _, err := range permanent {
		if Retryable(NewProviderError(err, errors.New("cause"))) {
			t.Fatalf("expected %v to be non-retryable", err)
		}
	}
}

func TestProviderErrorUnwrap(t *testing.T) {
	err := NewRateLimitedError(errors.New("boom"), time.Second)
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected errors.Is ErrRateLimited, got %v", err)
	}
	if err.RetryAfter != time.Second {
		t.Fatalf("expected retry after 1s, got %v", err.RetryAfter)
	}
	if got := err.Error(); got == "" {
		t.Fatalf("expected non-empty message")
	}
}

func TestStatusForError(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{ErrInvalidCredentials, StatusInvalid},
		{ErrInsufficientQuota, StatusQuotaExhausted},
		{ErrRateLimited, StatusRateLimited},
		{ErrModelUnavailable, StatusInvalid},
		{ErrRequestRejected, StatusInvalid},
		{ErrProviderUnavailable, StatusUnavailable},
		{ErrProviderTimeout, StatusUnavailable},
		{ErrUnknownProvider, StatusUnavailable},
	}
	for _, tc := range cases {
		if got := StatusForError(NewProviderError(tc.err, errors.New("cause"))); got != tc.want {
			t.Fatalf("expected %s, got %s", tc.want, got)
		}
	}
}

func TestRegistry(t *testing.T) {
	registry := NewRegistry(NewOpenAIProvider("http://openai"), NewGeminiProvider("http://gemini"), NewMockProvider())
	for _, providerType := range []string{ProviderOpenAI, ProviderGemini, ProviderMock} {
		provider, err := registry.Get(providerType)
		if err != nil {
			t.Fatalf("expected %q to resolve, got %v", providerType, err)
		}
		if provider.Type() != providerType {
			t.Fatalf("expected type %q, got %q", providerType, provider.Type())
		}
	}
	if _, err := registry.Get("anthropic"); !errors.Is(err, ErrUnknownProvider) {
		t.Fatalf("expected ErrUnknownProvider, got %v", err)
	}
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	var key [32]byte
	for i := range key {
		key[i] = byte(i)
	}
	encrypted, err := EncryptAPIKey("sk-secret-value", key)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	if encrypted == "sk-secret-value" {
		t.Fatalf("expected ciphertext to differ from plaintext")
	}
	decrypted, err := DecryptAPIKey(encrypted, key)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if decrypted != "sk-secret-value" {
		t.Fatalf("expected roundtrip, got %q", decrypted)
	}
}

func TestDecryptWrongKey(t *testing.T) {
	var key [32]byte
	for i := range key {
		key[i] = byte(i)
	}
	encrypted, err := EncryptAPIKey("sk-secret-value", key)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	var wrong [32]byte
	if _, err := DecryptAPIKey(encrypted, wrong); !errors.Is(err, ErrBadCiphertext) {
		t.Fatalf("expected ErrBadCiphertext, got %v", err)
	}
}

func TestDecryptTampered(t *testing.T) {
	var key [32]byte
	encrypted, err := EncryptAPIKey("sk-secret-value", key)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	tampered := encrypted[:len(encrypted)-2] + "ff"
	if _, err := DecryptAPIKey(tampered, key); !errors.Is(err, ErrBadCiphertext) {
		t.Fatalf("expected ErrBadCiphertext, got %v", err)
	}
	if _, err := DecryptAPIKey("not-hex!!", key); !errors.Is(err, ErrBadCiphertext) {
		t.Fatalf("expected ErrBadCiphertext, got %v", err)
	}
}
