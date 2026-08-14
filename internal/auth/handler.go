package auth

import (
	"context"
	"errors"
	"issue-refiner/internal/github"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	AuthService *AuthService
}

func NewAuthHandler(authService *AuthService) AuthHandler {
	return AuthHandler{AuthService: authService}
}

func (authHandler *AuthHandler) StartGitHubLogin(c *gin.Context) {
	url := authHandler.AuthService.GitHubOAuth.GithubOAuthConfig.AuthCodeURL("randomstate")
	c.Redirect(http.StatusSeeOther, url)
}

func (authHandler *AuthHandler) GitHubCallback(c *gin.Context) {
	state := c.Query("state")
	code := c.Query("code")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	user, err := authHandler.AuthService.AuthenticateGitHub(ctx, code, state)
	if err != nil {
		switch {
		case errors.Is(err, errAuthTimeout):
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": "GitHub authentication timed out"})

		case errors.Is(err, errTokenFailed):
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed token exchange", "details": err.Error()})

		case errors.Is(err, errStateMismatch):
			c.JSON(http.StatusBadRequest, gin.H{"error": "State Mismatch"})

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
	c.JSON(http.StatusAccepted, gin.H{
		"status":  "success",
		"message": "Token obtained successfully",
		"user":    user,
	})

}

func (authHandler *AuthHandler) RegisterRoutes(router *gin.Engine) {
	router.GET("/auth/github", authHandler.StartGitHubLogin)
	router.GET("/auth/github/callback", authHandler.GitHubCallback)
}
