package sessions

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	TokenHash      string     `json:"-" db:"token_hash"`
	UserID         uuid.UUID  `json:"user_id" db:"user_id"`
	OrganizationID *uuid.UUID `json:"organization_id" db:"organization_id"`
	ExpiresAt      time.Time  `json:"expires_at" db:"expires_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}
