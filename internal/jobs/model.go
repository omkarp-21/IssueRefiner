package jobs

import (
	"time"

	"github.com/google/uuid"
)

const (
	KindRefine      = "refine"
	KindInvestigate = "investigate"
	StatusQueued    = "queued"
	StatusRunning   = "running"
	StatusDone      = "done"
	StatusFailed    = "failed"
)

type Job struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	OrganizationID   uuid.UUID  `json:"organization_id" db:"organization_id"`
	RepositoryID     *uuid.UUID `json:"repository_id" db:"repository_id"`
	ProviderConfigID *uuid.UUID `json:"provider_config_id" db:"provider_config_id"`
	ProviderType     *string    `json:"provider_type" db:"provider_type"`
	Model            *string    `json:"model" db:"model"`
	CreatedBy        uuid.UUID  `json:"created_by" db:"created_by"`
	Kind             string     `json:"kind" db:"kind"`
	IssueNumber      int        `json:"issue_number" db:"issue_number"`
	Status           string     `json:"status" db:"status"`
	InputTitle       string     `json:"input_title" db:"input_title"`
	InputBody        string     `json:"input_body" db:"input_body"`
	Result           *string    `json:"result" db:"result"`
	Error            *string    `json:"error" db:"error"`
	InputTokens      *int       `json:"input_tokens" db:"input_tokens"`
	OutputTokens     *int       `json:"output_tokens" db:"output_tokens"`
	Attempts         int        `json:"attempts" db:"attempts"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}
