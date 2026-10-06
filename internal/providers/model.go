package providers

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	ProviderOpenAI = "openai"
	ProviderGemini = "gemini"
	ProviderMock   = "mock"
)

const (
	StatusPending        = "pending"
	StatusHealthy        = "healthy"
	StatusInvalid        = "invalid"
	StatusRateLimited    = "rate_limited"
	StatusQuotaExhausted = "quota_exhausted"
	StatusUnavailable    = "unavailable"
	StatusDisabled       = "disabled"
)

type ProviderConfig struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	OrganizationID  uuid.UUID  `json:"organization_id" db:"organization_id"`
	ProviderType    string     `json:"provider_type" db:"provider_type"`
	DisplayName     string     `json:"display_name" db:"display_name"`
	Enabled         bool       `json:"enabled" db:"enabled"`
	Status          string     `json:"status" db:"status"`
	LastValidatedAt *time.Time `json:"last_validated_at" db:"last_validated_at"`
	LastError       *string    `json:"last_error" db:"last_error"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

type ProviderModel struct {
	ID               uuid.UUID `json:"id" db:"id"`
	ProviderConfigID uuid.UUID `json:"provider_config_id" db:"provider_config_id"`
	Model            string    `json:"model" db:"model"`
	Enabled          bool      `json:"enabled" db:"enabled"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

func ValidateProviderType(providerType string) error {
	if providerType != ProviderOpenAI && providerType != ProviderGemini && providerType != ProviderMock {
		return fmt.Errorf("%w: %s", ErrUnknownProvider, providerType)
	}
	return nil
}

func StatusForError(err error) string {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		return StatusInvalid
	case errors.Is(err, ErrInsufficientQuota):
		return StatusQuotaExhausted
	case errors.Is(err, ErrRateLimited):
		return StatusRateLimited
	case errors.Is(err, ErrModelUnavailable):
		return StatusInvalid
	case errors.Is(err, ErrRequestRejected):
		return StatusInvalid
	default:
		return StatusUnavailable
	}
}
