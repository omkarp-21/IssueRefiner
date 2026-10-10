package jobs

import (
	"context"
	"fmt"
	"testing"

	"issue-refiner/internal/testdb"

	"github.com/google/uuid"
)

func TestListForIssueBounded(t *testing.T) {
	pool := testdb.Connect(t)
	ctx := context.Background()

	userID := uuid.New()
	if _, err := pool.Exec(ctx, "INSERT INTO users (id, github_id, username, display_name, avatar_url) VALUES ($1, 424242, 'bound-test-user', 'Bound', 'https://example.com')", userID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id=$1", userID)
	})
	orgID := uuid.New()
	if _, err := pool.Exec(ctx, "INSERT INTO organizations (id, name) VALUES ($1, 'bound-test-org')", orgID); err != nil {
		t.Fatalf("seed org: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM organizations WHERE id=$1", orgID)
	})
	if _, err := pool.Exec(ctx, "INSERT INTO organization_members (organization_id, user_id, role) VALUES ($1, $2, 'owner')", orgID, userID); err != nil {
		t.Fatalf("seed membership: %v", err)
	}
	repoID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO repositories (id, organization_id, github_id, owner, name, full_name, default_branch, html_url) VALUES ($1, $2, 4242, 'octo', 'demo', 'octo/demo', 'main', 'https://example.com')`, repoID, orgID); err != nil {
		t.Fatalf("seed repo: %v", err)
	}

	base := int64(1700000000)
	statuses := []string{"done", "failed", "queued"}
	for i := 0; i < 60; i++ {
		if _, err := pool.Exec(ctx, `INSERT INTO jobs (organization_id, repository_id, created_by, kind, issue_number, status, input_title, created_at) VALUES ($1, $2, $3, 'investigate', 7, $4, $5, to_timestamp($6))`, orgID, repoID, userID, statuses[i%3], fmt.Sprintf("title-%02d", i), float64(base)+float64(i)); err != nil {
			t.Fatalf("seed job %d: %v", i, err)
		}
	}
	// Noise that must not leak in: other org, other repo, other issue, refine kind.
	otherOrgID := uuid.New()
	if _, err := pool.Exec(ctx, "INSERT INTO organizations (id, name) VALUES ($1, 'bound-test-other')", otherOrgID); err != nil {
		t.Fatalf("seed other org: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM organizations WHERE id=$1", otherOrgID)
	})
	otherRepoID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO repositories (id, organization_id, github_id, owner, name, full_name, default_branch, html_url) VALUES ($1, $2, 4243, 'octo', 'other', 'octo/other', 'main', 'https://example.com')`, otherRepoID, orgID); err != nil {
		t.Fatalf("seed other repo: %v", err)
	}
	for _, seed := range []struct {
		org, repo uuid.UUID
		issue     int
		kind      string
	}{
		{otherOrgID, otherRepoID, 7, KindInvestigate},
		{orgID, otherRepoID, 7, KindInvestigate},
		{orgID, repoID, 8, KindInvestigate},
		{orgID, repoID, 7, KindRefine},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO jobs (organization_id, repository_id, created_by, kind, issue_number, status, input_title) VALUES ($1, $2, $3, $4, $5, 'done', 'noise')`, seed.org, seed.repo, userID, seed.kind, seed.issue); err != nil {
			t.Fatalf("seed noise: %v", err)
		}
	}

	repository := NewJobRepository(pool)
	got, err := repository.ListForIssue(ctx, orgID, repoID, 7)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 50 {
		t.Fatalf("expected exactly 50 investigations, got %d", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].CreatedAt.Before(got[i].CreatedAt) {
			t.Fatalf("expected newest-first ordering at index %d", i)
		}
	}
	if got[0].InputTitle != "title-59" || got[49].InputTitle != "title-10" {
		t.Fatalf("expected title-59..title-10, got %q..%q", got[0].InputTitle, got[49].InputTitle)
	}
	for _, job := range got {
		if job.Kind != KindInvestigate || job.IssueNumber != 7 || job.OrganizationID != orgID || job.RepositoryID == nil || *job.RepositoryID != repoID {
			t.Fatalf("foreign row leaked: %+v", job)
		}
	}

	empty, err := repository.ListForIssue(ctx, orgID, repoID, 999)
	if err != nil {
		t.Fatalf("empty list: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected empty result, got %d rows", len(empty))
	}
}
