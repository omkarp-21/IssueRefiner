package auth

import (
	"issue-refiner/internal/orgs"
	"issue-refiner/internal/users"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestCurrentOrg(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	organization := &orgs.Organization{ID: uuid.New(), Name: "acme"}
	c.Set("organization", organization)
	c.Set("role", orgs.RoleOwner)
	got, role := CurrentOrg(c)
	if got == nil || got.ID != organization.ID {
		t.Fatalf("expected organization context, got %v", got)
	}
	if role != orgs.RoleOwner {
		t.Fatalf("expected owner role, got %q", role)
	}
}

func TestCurrentOrgMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	got, role := CurrentOrg(c)
	if got != nil || role != "" {
		t.Fatalf("expected no organization context, got %v %q", got, role)
	}
}

func TestCurrentUserMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	if CurrentUser(c) != nil {
		t.Fatalf("expected no user")
	}
	c.Set("user", &users.User{ID: uuid.New()})
	if CurrentUser(c) == nil {
		t.Fatalf("expected user context")
	}
}

func TestRequireOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("organization", &orgs.Organization{ID: uuid.New()})
	c.Set("role", orgs.RoleOwner)
	RequireOwner()(c)
	if c.IsAborted() {
		t.Fatalf("expected owner to pass")
	}
}

func TestRequireOwnerForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, role := range []string{orgs.RoleMember, ""} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("organization", &orgs.Organization{ID: uuid.New()})
		c.Set("role", role)
		RequireOwner()(c)
		if !c.IsAborted() {
			t.Fatalf("expected role %q to be forbidden", role)
		}
		if w.Code != 403 {
			t.Fatalf("expected 403, got %d", w.Code)
		}
	}
}
