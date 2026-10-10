package repos

import (
	"errors"
	"issue-refiner/internal/auth"
	"issue-refiner/internal/github"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RepositoryHandler struct {
	RepositoryService *RepositoryService
}

func NewRepositoryHandler(repositoryService *RepositoryService) RepositoryHandler {
	return RepositoryHandler{RepositoryService: repositoryService}
}

type importRequest struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
}

func parsePagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "30"))
	return NormalizePage(page, perPage)
}

func githubStatus(err error) int {
	switch {
	case errors.Is(err, github.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, github.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, github.ErrRateLimited):
		return http.StatusTooManyRequests
	case errors.Is(err, github.ErrGitHubServer):
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

func (repositoryHandler *RepositoryHandler) ImportRepo(c *gin.Context) {
	user := auth.CurrentUser(c)
	organization, _ := auth.CurrentOrg(c)
	if user == nil || organization == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req importRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	repository, err := repositoryHandler.RepositoryService.ImportRepo(c.Request.Context(), organization.ID, user.ID, req.Owner, req.Repo)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidRepo):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		case errors.Is(err, ErrGitHubToken):
			c.JSON(http.StatusForbidden, gin.H{"error": "GitHub account not connected"})
		case errors.Is(err, ErrTokenNeedsReauth):
			c.JSON(http.StatusForbidden, gin.H{"error": "GitHub access expired; reconnect GitHub and try again"})
		case errors.Is(err, ErrGitHubUpstream):
			c.JSON(githubStatus(err), gin.H{"error": "GitHub request failed"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to import repository"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"repository": repository})
}

func (repositoryHandler *RepositoryHandler) ListRepos(c *gin.Context) {
	organization, _ := auth.CurrentOrg(c)
	if organization == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	repositories, err := repositoryHandler.RepositoryService.ListRepos(c.Request.Context(), organization.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list repositories"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"repositories": repositories})
}

func (repositoryHandler *RepositoryHandler) GetRepo(c *gin.Context) {
	organization, _ := auth.CurrentOrg(c)
	if organization == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid repository id"})
		return
	}
	repository, err := repositoryHandler.RepositoryService.GetRepo(c.Request.Context(), organization.ID, id)
	if err != nil {
		switch {
		case errors.Is(err, ErrRepositoryNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Repository not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch repository"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"repository": repository})
}

func (repositoryHandler *RepositoryHandler) ListContributors(c *gin.Context) {
	user := auth.CurrentUser(c)
	organization, _ := auth.CurrentOrg(c)
	if user == nil || organization == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid repository id"})
		return
	}
	page, perPage := parsePagination(c)
	contributors, err := repositoryHandler.RepositoryService.ListContributors(c.Request.Context(), organization.ID, user.ID, id, page, perPage)
	if err != nil {
		switch {
		case errors.Is(err, ErrRepositoryNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Repository not found"})
		case errors.Is(err, ErrGitHubToken):
			c.JSON(http.StatusForbidden, gin.H{"error": "GitHub account not connected"})
		case errors.Is(err, ErrTokenNeedsReauth):
			c.JSON(http.StatusForbidden, gin.H{"error": "GitHub access expired; reconnect GitHub and try again"})
		case errors.Is(err, ErrGitHubUpstream):
			c.JSON(githubStatus(err), gin.H{"error": "GitHub request failed"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch contributors"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"contributors": contributors, "page": page, "per_page": perPage})
}

func (repositoryHandler *RepositoryHandler) ListIssues(c *gin.Context) {
	user := auth.CurrentUser(c)
	organization, _ := auth.CurrentOrg(c)
	if user == nil || organization == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid repository id"})
		return
	}
	state := c.DefaultQuery("state", "open")
	page, perPage := parsePagination(c)
	issues, err := repositoryHandler.RepositoryService.ListIssues(c.Request.Context(), organization.ID, user.ID, id, state, page, perPage)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidRepo):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state"})
		case errors.Is(err, ErrRepositoryNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Repository not found"})
		case errors.Is(err, ErrGitHubToken):
			c.JSON(http.StatusForbidden, gin.H{"error": "GitHub account not connected"})
		case errors.Is(err, ErrTokenNeedsReauth):
			c.JSON(http.StatusForbidden, gin.H{"error": "GitHub access expired; reconnect GitHub and try again"})
		case errors.Is(err, ErrGitHubUpstream):
			c.JSON(githubStatus(err), gin.H{"error": "GitHub request failed"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch issues"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"issues": issues, "page": page, "per_page": perPage})
}

func (repositoryHandler *RepositoryHandler) RegisterRoutes(router *gin.Engine, requireAuth gin.HandlerFunc) {
	api := router.Group("/api", requireAuth)
	api.POST("/repos/import", repositoryHandler.ImportRepo)
	api.GET("/repos", repositoryHandler.ListRepos)
	api.GET("/repos/:id", repositoryHandler.GetRepo)
	api.GET("/repos/:id/contributors", repositoryHandler.ListContributors)
	api.GET("/repos/:id/issues", repositoryHandler.ListIssues)
}
