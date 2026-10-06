package repos

import (
	"time"

	"github.com/google/uuid"
)

type Repository struct {
	ID             uuid.UUID `json:"id" db:"id"`
	OrganizationID uuid.UUID `json:"organization_id" db:"organization_id"`
	GitHubID       int64     `json:"github_id" db:"github_id"`
	Owner          string    `json:"owner" db:"owner"`
	Name           string    `json:"name" db:"name"`
	FullName       string    `json:"full_name" db:"full_name"`
	Description    *string   `json:"description" db:"description"`
	DefaultBranch  string    `json:"default_branch" db:"default_branch"`
	HTMLURL        string    `json:"html_url" db:"html_url"`
	Private        bool      `json:"private" db:"private"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
	SyncedAt       time.Time `json:"synced_at" db:"synced_at"`
}
