package providers

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInsufficientQuota   = errors.New("insufficient quota")
	ErrRateLimited         = errors.New("rate limited")
	ErrModelUnavailable    = errors.New("model unavailable")
	ErrProviderUnavailable = errors.New("provider unavailable")
	ErrRequestRejected     = errors.New("request rejected")
	ErrProviderTimeout     = errors.New("provider timeout")
	ErrUnknownProvider     = errors.New("unknown provider")
)

type ProviderError struct {
	Category   error
	RetryAfter time.Duration
	cause      error
}

func NewProviderError(category error, cause error) *ProviderError {
	return &ProviderError{Category: category, cause: cause}
}

func NewRateLimitedError(cause error, retryAfter time.Duration) *ProviderError {
	return &ProviderError{Category: ErrRateLimited, RetryAfter: retryAfter, cause: cause}
}

func (providerError *ProviderError) Error() string {
	if providerError.cause != nil {
		return fmt.Sprintf("%s: %s", providerError.Category.Error(), providerError.cause.Error())
	}
	return providerError.Category.Error()
}

func (providerError *ProviderError) Unwrap() []error {
	if providerError.cause != nil {
		return []error{providerError.Category, providerError.cause}
	}
	return []error{providerError.Category}
}

func Retryable(err error) bool {
	switch {
	case errors.Is(err, ErrRateLimited):
		return true
	case errors.Is(err, ErrProviderUnavailable):
		return true
	case errors.Is(err, ErrProviderTimeout):
		return true
	default:
		return false
	}
}
