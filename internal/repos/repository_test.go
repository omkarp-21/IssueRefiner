package repos

import (
	"context"
	"fmt"
	"testing"

	"issue-refiner/internal/testdb"

	"github.com/google/uuid"
)

func TestListForOrgBounded(t *testing.T) {
	pool := testdb.Connect(t)
	ctx := context.Background()

	orgID := uuid.New()
	if _, err := pool.Exec(ctx, "INSERT INTO organizations (id, name) VALUES ($1, 'bound-test-org')", orgID); err != nil {
		t.Fatalf("seed org: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM organizations WHERE id=$1", orgID)
	})
	otherOrgID := uuid.New()
	if _, err := pool.Exec(ctx, "INSERT INTO organizations (id, name) VALUES ($1, 'bound-test-other')", otherOrgID); err != nil {
		t.Fatalf("seed other org: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM organizations WHERE id=$1", otherOrgID)
	})

	for i := 0; i < 60; i++ {
		name := fmt.Sprintf("repo-%02d", i)
		if _, err := pool.Exec(ctx, `INSERT INTO repositories (organization_id, github_id, owner, name, full_name, default_branch, html_url) VALUES ($1, $2, 'octo', $3, $4, 'main', 'https://example.com')`, orgID, 1000+i, name, "octo/"+name); err != nil {
			t.Fatalf("seed repo %d: %v", i, err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO repositories (organization_id, github_id, owner, name, full_name, default_branch, html_url) VALUES ($1, $2, 'octo', $3, $4, 'main', 'https://example.com')`, otherOrgID, 2000+i, name, "octo/"+name); err != nil {
			t.Fatalf("seed other repo %d: %v", i, err)
		}
	}

	repository := NewRepositoryRepository(pool)
	got, err := repository.ListForOrg(ctx, orgID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 50 {
		t.Fatalf("expected exactly 50 repositories, got %d", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].FullName > got[i].FullName {
			t.Fatalf("expected alphabetical order, got %q before %q", got[i-1].FullName, got[i].FullName)
		}
		if got[i].OrganizationID != orgID {
			t.Fatalf("foreign organization row leaked: %v", got[i].ID)
		}
	}
	if got[0].FullName != "octo/repo-00" || got[49].FullName != "octo/repo-49" {
		t.Fatalf("expected repo-00..repo-49, got %q..%q", got[0].FullName, got[49].FullName)
	}

	empty, err := repository.ListForOrg(ctx, uuid.New())
	if err != nil {
		t.Fatalf("empty list: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected empty result, got %d rows", len(empty))
	}
}
