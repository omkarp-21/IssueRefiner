package auth

import (
	"encoding/json"
	"issue-refiner/internal/orgs"
	"issue-refiner/internal/users"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func testAuthHandler() *AuthHandler {
	handler := NewAuthHandler(nil, nil, nil)
	return &handler
}

func TestGetMeOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	user := &users.User{ID: uuid.New(), Username: "octocat"}
	organization := &orgs.Organization{ID: uuid.New(), Name: "acme"}
	c.Set("user", user)
	c.Set("organization", organization)
	c.Set("role", orgs.RoleOwner)
	testAuthHandler().GetMe(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body struct {
		User         users.User        `json:"user"`
		Organization orgs.Organization `json:"organization"`
		Role         string            `json:"role"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON body, got %v", err)
	}
	if body.User.ID != user.ID || body.Organization.ID != organization.ID || body.Role != orgs.RoleOwner {
		t.Fatalf("unexpected body %+v", body)
	}
}

func TestGetMeMember(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user", &users.User{ID: uuid.New()})
	c.Set("organization", &orgs.Organization{ID: uuid.New()})
	c.Set("role", orgs.RoleMember)
	testAuthHandler().GetMe(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body struct {
		Role string `json:"role"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON body, got %v", err)
	}
	if body.Role != orgs.RoleMember {
		t.Fatalf("expected member role, got %q", body.Role)
	}
}

func TestGetMeUnauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	testAuthHandler().GetMe(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestCurrentSessionID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	if id := CurrentSessionID(c); id != uuid.Nil {
		t.Fatalf("expected nil UUID, got %v", id)
	}
	want := uuid.New()
	c.Set("session_id", want)
	if id := CurrentSessionID(c); id != want {
		t.Fatalf("expected %v, got %v", want, id)
	}
}

func TestLogoutWithoutSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user", &users.User{ID: uuid.New()})
	c.Set("organization", &orgs.Organization{ID: uuid.New()})
	c.Set("role", orgs.RoleMember)
	testAuthHandler().Logout(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	cookie := w.Header().Get("Set-Cookie")
	if !strings.Contains(cookie, "session_token=") || !strings.Contains(strings.ToLower(cookie), "httponly") {
		t.Fatalf("expected cleared HttpOnly session cookie, got %q", cookie)
	}
}
