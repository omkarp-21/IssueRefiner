package tokens

import (
	"time"

	"github.com/google/uuid"
)

type GitHubToken struct {
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	AccessToken string    `json:"-" db:"access_token"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}
