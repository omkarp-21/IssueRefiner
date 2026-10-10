package auth

import (
	"context"
	"issue-refiner/internal/orgs"
	"issue-refiner/internal/sessions"
	"issue-refiner/internal/users"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func livePool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := "postgres://postgres:postgres@localhost:5432/issue_refiner?sslmode=disable"
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("postgres unreachable: %v", err)
	}
	return pool
}

func TestLogoutRevokesSessionLive(t *testing.T) {
	pool := livePool(t)
	defer pool.Close()
	ctx := context.Background()

	userID := uuid.New()
	orgID := uuid.New()
	if _, err := pool.Exec(ctx, "INSERT INTO users (id, github_id, username, display_name, avatar_url) VALUES ($1, 999991, 'logout-test', 'Logout', 'https://example.com/x.png')", userID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	defer pool.Exec(context.Background(), "DELETE FROM users WHERE id=$1", userID)
	if _, err := pool.Exec(ctx, "INSERT INTO organizations (id, name) VALUES ($1, 'logout-test-org')", orgID); err != nil {
		t.Fatalf("seed org: %v", err)
	}
	defer pool.Exec(context.Background(), "DELETE FROM organizations WHERE id=$1", orgID)
	if _, err := pool.Exec(ctx, "INSERT INTO organization_members (organization_id, user_id, role) VALUES ($1, $2, 'member')", orgID, userID); err != nil {
		t.Fatalf("seed membership: %v", err)
	}

	sessionRepo := sessions.NewSessionRepository(pool)
	sessionService := sessions.NewSessionService(&sessionRepo)
	token, session, err := sessionService.Create(ctx, userID, orgID)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	userRepo := users.NewUserRepository(pool)
	userService := users.NewUserService(&userRepo)
	orgRepo := orgs.NewOrganizationRepository(pool)
	orgService := orgs.NewOrganizationService(&orgRepo)
	handler := NewAuthHandler(nil, &sessionService, &orgService)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	c.Request = req

	RequireAuth(&sessionService, &userService, &orgService)(c)
	if c.IsAborted() {
		t.Fatalf("expected middleware to pass, got %d", w.Code)
	}
	handler.Logout(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var remaining int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM sessions WHERE id=$1", session.ID).Scan(&remaining); err != nil {
		t.Fatalf("check session row: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("expected session row deleted")
	}

	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	req2 := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
	req2.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	c2.Request = req2
	RequireAuth(&sessionService, &userService, &orgService)(c2)
	if !c2.IsAborted() || w2.Code != http.StatusUnauthorized {
		t.Fatalf("expected revoked token to yield 401, got aborted=%v code=%d", c2.IsAborted(), w2.Code)
	}
}
