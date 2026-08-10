package auth

import (
	"context"
	"errors"
	"io"
	"issue-refiner/internal/github"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	GitHubOAuth *github.GitHubOAuth
}

func NewHandler(githubOAuth *github.GitHubOAuth) AuthHandler {
	return AuthHandler{GitHubOAuth: githubOAuth}
}

func (authHandler *AuthHandler) StartGitHubLogin(c *gin.Context) {
	url := authHandler.GitHubOAuth.GithubOAuthConfig.AuthCodeURL("randomstate")
	c.Redirect(http.StatusSeeOther, url)
}

func (authHandler *AuthHandler) GitHubCallback(c *gin.Context) {
	state := c.Query("state")
	if state != "randomstate" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "State Mismatch"})
		return
	}
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Code parameter missing"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	token, err := authHandler.GitHubOAuth.GithubOAuthConfig.Exchange(ctx, code)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": "GitHub token exchange timed out"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed token exchange", "details": err.Error()})
		return
	}

	reqURL := "https://api.github.com/user"
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "IssueRefiner-App")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed user retrieval", "details": err.Error()})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GitHub api returned non-200 status", "details": resp.StatusCode})
		return
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"details": err.Error()})
		return
	}

	jsonString := string(bodyBytes)
	c.JSON(http.StatusAccepted, gin.H{
		"status":   "success",
		"message":  "Token obtained successfully",
		"token":    token,
		"response": jsonString,
	})
}

func (authHandler *AuthHandler) RegisterRoutes(router *gin.Engine) {
	router.GET("/auth/github", authHandler.StartGitHubLogin)
	router.GET("/auth/github/callback", authHandler.GitHubCallback)
}
