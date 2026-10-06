package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"issue-refiner/internal/github"
	"issue-refiner/internal/orgs"
	"issue-refiner/internal/sessions"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	AuthService         *AuthService
	SessionService      *sessions.SessionService
	OrganizationService *orgs.OrganizationService
}

const oauthStateCookie = "oauth_state"

func NewAuthHandler(authService *AuthService, sessionService *sessions.SessionService, organizationService *orgs.OrganizationService) AuthHandler {
	return AuthHandler{AuthService: authService, SessionService: sessionService, OrganizationService: organizationService}
}

func newOAuthState() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}

func (authHandler *AuthHandler) StartGitHubLogin(c *gin.Context) {
	state, err := newOAuthState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start login"})
		return
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(oauthStateCookie, state, 300, "/", "", false, true)
	url := authHandler.AuthService.GitHubOAuth.GithubOAuthConfig.AuthCodeURL(state)
	c.Redirect(http.StatusSeeOther, url)
}

func (authHandler *AuthHandler) GitHubCallback(c *gin.Context) {
	state := c.Query("state")
	code := c.Query("code")
	expectedState, err := c.Cookie(oauthStateCookie)
	if err != nil || expectedState == "" || state != expectedState {
		c.JSON(http.StatusBadRequest, gin.H{"error": "State Mismatch"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	user, err := authHandler.AuthService.AuthenticateGitHub(ctx, code, state, expectedState)
	if err != nil {
		switch {
		case errors.Is(err, errAuthTimeout):
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": "GitHub authentication timed out"})

		case errors.Is(err, errTokenFailed):
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed token exchange", "details": err.Error()})

		case errors.Is(err, errStateMismatch):
			c.JSON(http.StatusBadRequest, gin.H{"error": "State Mismatch"})

		case errors.Is(err, errPersistUser):
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to persist user"})

		case errors.Is(err, github.ErrDecodeGitHubUser):
			_ = c.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to decode GitHub response",
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		return
	}
	organization, err := authHandler.OrganizationService.EnsurePersonalOrg(ctx, user.ID, user.Username+" / personal")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare organization"})
		return
	}
	token, _, err := authHandler.SessionService.Create(ctx, user.ID, organization.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(oauthStateCookie, "", -1, "/", "", false, true)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(sessionCookie, token, 7*24*3600, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{
		"status":       "success",
		"message":      "Token obtained successfully",
		"user":         user,
		"organization": organization,
	})

}

func (authHandler *AuthHandler) RegisterRoutes(router *gin.Engine) {
	router.GET("/auth/github", authHandler.StartGitHubLogin)
	router.GET("/auth/github/callback", authHandler.GitHubCallback)
}
