package jobs

import (
	"errors"
	"issue-refiner/internal/auth"
	"issue-refiner/internal/github"
	"issue-refiner/internal/providers"
	"issue-refiner/internal/repos"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type JobHandler struct {
	JobService *JobService
	Queue      Enqueuer
}

type Enqueuer interface {
	Enqueue(id uuid.UUID) error
}

func NewJobHandler(jobService *JobService, queue Enqueuer) JobHandler {
	return JobHandler{JobService: jobService, Queue: queue}
}

type refineRequest struct {
	ProviderConfigID uuid.UUID `json:"provider_config_id"`
	Model            string    `json:"model"`
	IssueNumber      int       `json:"issue_number"`
	Title            string    `json:"title"`
	Body             string    `json:"body"`
}

type investigateRequest struct {
	ProviderConfigID uuid.UUID `json:"provider_config_id"`
	Model            string    `json:"model"`
	IssueNumber      int       `json:"issue_number"`
}

func (jobHandler *JobHandler) CreateRepoRefine(c *gin.Context) {
	user := auth.CurrentUser(c)
	organization, _ := auth.CurrentOrg(c)
	if user == nil || organization == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	repositoryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid repository id"})
		return
	}
	var req refineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	job, err := jobHandler.JobService.EnqueueIssue(c.Request.Context(), organization.ID, repositoryID, req.ProviderConfigID, req.Model, user.ID, req.IssueNumber, req.Title, req.Body)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		case errors.Is(err, repos.ErrRepositoryNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Repository not found"})
		case errors.Is(err, providers.ErrConfigNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		case errors.Is(err, providers.ErrConfigDisabled),
			errors.Is(err, providers.ErrModelNotEnabled):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create job"})
		}
		return
	}
	if err := jobHandler.Queue.Enqueue(job.ID); err != nil {
		_, _ = jobHandler.JobService.FailJob(c.Request.Context(), job.ID, "server busy, try again later")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Server busy, try again later"})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"job": job})
}

func (jobHandler *JobHandler) GetJob(c *gin.Context) {
	organization, _ := auth.CurrentOrg(c)
	if organization == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job id"})
		return
	}
	job, repository, err := jobHandler.JobService.GetForOrg(c.Request.Context(), organization.ID, id)
	if err != nil {
		switch {
		case errors.Is(err, ErrJobNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch job"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"job": job, "repository": repository})
}

func (jobHandler *JobHandler) ListInvestigations(c *gin.Context) {
	organization, _ := auth.CurrentOrg(c)
	if organization == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	repositoryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid repository id"})
		return
	}
	issueNumber, err := strconv.Atoi(c.Param("number"))
	if err != nil || issueNumber <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid issue number"})
		return
	}
	investigations, err := jobHandler.JobService.ListInvestigations(c.Request.Context(), organization.ID, repositoryID, issueNumber)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		case errors.Is(err, repos.ErrRepositoryNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Repository not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list investigations"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"investigations": investigations})
}

func (jobHandler *JobHandler) RegisterRoutes(router *gin.Engine, requireAuth gin.HandlerFunc) {
	api := router.Group("/api", requireAuth)
	api.POST("/repos/:id/refine", jobHandler.CreateRepoRefine)
	api.POST("/repos/:id/investigate", jobHandler.StartInvestigation)
	api.POST("/repos/:id/issues/:number/investigations/:investigation_id/reinvestigate", jobHandler.Reinvestigate)
	api.GET("/repos/:id/issues/:number/investigations/:investigation_id/request-draft", jobHandler.GetInformationDraft)
	api.POST("/repos/:id/issues/:number/investigations/:investigation_id/request-information", jobHandler.PostInformationRequest)
	api.GET("/repos/:id/issues/:number/investigations", jobHandler.ListInvestigations)
	api.GET("/jobs/:id", jobHandler.GetJob)
}

func (jobHandler *JobHandler) StartInvestigation(c *gin.Context) {
	user := auth.CurrentUser(c)
	organization, _ := auth.CurrentOrg(c)
	if user == nil || organization == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	repositoryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid repository id"})
		return
	}
	var req investigateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	job, err := jobHandler.JobService.EnqueueInvestigation(c.Request.Context(), organization.ID, repositoryID, req.ProviderConfigID, req.Model, user.ID, req.IssueNumber)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		case errors.Is(err, repos.ErrRepositoryNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Repository not found"})
		case errors.Is(err, providers.ErrConfigNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		case errors.Is(err, providers.ErrConfigDisabled),
			errors.Is(err, providers.ErrModelNotEnabled):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, repos.ErrGitHubToken):
			c.JSON(http.StatusForbidden, gin.H{"error": "GitHub account not connected"})
		case errors.Is(err, github.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Issue not found"})
		case errors.Is(err, repos.ErrGitHubUpstream):
			c.JSON(http.StatusBadGateway, gin.H{"error": "GitHub request failed"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start investigation"})
		}
		return
	}
	if err := jobHandler.Queue.Enqueue(job.ID); err != nil {
		_, _ = jobHandler.JobService.FailJob(c.Request.Context(), job.ID, "server busy, try again later")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Server busy, try again later"})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"job": job})
}

func (jobHandler *JobHandler) Reinvestigate(c *gin.Context) {
	user := auth.CurrentUser(c)
	organization, _ := auth.CurrentOrg(c)
	if user == nil || organization == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	repositoryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid repository id"})
		return
	}
	issueNumber, err := strconv.Atoi(c.Param("number"))
	if err != nil || issueNumber <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid issue number"})
		return
	}
	sourceID, err := uuid.Parse(c.Param("investigation_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid investigation id"})
		return
	}
	var req investigateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	job, err := jobHandler.JobService.Reinvestigate(c.Request.Context(), organization.ID, repositoryID, issueNumber, sourceID, req.ProviderConfigID, req.Model, user.ID)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		case errors.Is(err, ErrJobNotFound),
			errors.Is(err, repos.ErrRepositoryNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		case errors.Is(err, ErrInvalidReinvestigation):
			c.JSON(http.StatusConflict, gin.H{"error": "Investigation cannot be reinvestigated"})
		case errors.Is(err, providers.ErrConfigNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		case errors.Is(err, providers.ErrConfigDisabled),
			errors.Is(err, providers.ErrModelNotEnabled):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, repos.ErrGitHubToken):
			c.JSON(http.StatusForbidden, gin.H{"error": "GitHub account not connected"})
		case errors.Is(err, github.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Issue not found"})
		case errors.Is(err, repos.ErrGitHubUpstream):
			c.JSON(http.StatusBadGateway, gin.H{"error": "GitHub request failed"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start investigation"})
		}
		return
	}
	if err := jobHandler.Queue.Enqueue(job.ID); err != nil {
		_, _ = jobHandler.JobService.FailJob(c.Request.Context(), job.ID, "server busy, try again later")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Server busy, try again later"})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"job": job})
}

type informationRequest struct {
	Comment string `json:"comment"`
}

func (jobHandler *JobHandler) parseInformationTarget(c *gin.Context) (uuid.UUID, int, uuid.UUID, bool) {
	repositoryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid repository id"})
		return uuid.UUID{}, 0, uuid.UUID{}, false
	}
	issueNumber, err := strconv.Atoi(c.Param("number"))
	if err != nil || issueNumber <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid issue number"})
		return uuid.UUID{}, 0, uuid.UUID{}, false
	}
	sourceID, err := uuid.Parse(c.Param("investigation_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid investigation id"})
		return uuid.UUID{}, 0, uuid.UUID{}, false
	}
	return repositoryID, issueNumber, sourceID, true
}

func (jobHandler *JobHandler) GetInformationDraft(c *gin.Context) {
	user := auth.CurrentUser(c)
	organization, _ := auth.CurrentOrg(c)
	if user == nil || organization == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	repositoryID, issueNumber, sourceID, ok := jobHandler.parseInformationTarget(c)
	if !ok {
		return
	}
	draft, err := jobHandler.JobService.GetInformationDraft(c.Request.Context(), organization.ID, repositoryID, issueNumber, sourceID)
	if err != nil {
		switch {
		case errors.Is(err, ErrJobNotFound),
			errors.Is(err, repos.ErrRepositoryNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		case errors.Is(err, ErrInvalidReinvestigation):
			c.JSON(http.StatusConflict, gin.H{"error": "Investigation cannot be used for an information request"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build information request"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"comment": draft})
}

func (jobHandler *JobHandler) PostInformationRequest(c *gin.Context) {
	user := auth.CurrentUser(c)
	organization, _ := auth.CurrentOrg(c)
	if user == nil || organization == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	repositoryID, issueNumber, sourceID, ok := jobHandler.parseInformationTarget(c)
	if !ok {
		return
	}
	var req informationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	comment, err := jobHandler.JobService.PostInformationRequest(c.Request.Context(), organization.ID, user.ID, repositoryID, issueNumber, sourceID, req.Comment)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		case errors.Is(err, ErrJobNotFound),
			errors.Is(err, repos.ErrRepositoryNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		case errors.Is(err, ErrInvalidReinvestigation):
			c.JSON(http.StatusConflict, gin.H{"error": "Investigation cannot be used for an information request"})
		case errors.Is(err, repos.ErrGitHubToken):
			c.JSON(http.StatusForbidden, gin.H{"error": "GitHub account not connected"})
		case errors.Is(err, github.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Issue not found or GitHub access revoked; reconnect GitHub and try again"})
		case errors.Is(err, repos.ErrGitHubUpstream):
			c.JSON(http.StatusBadGateway, gin.H{"error": "GitHub request failed"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to post information request"})
		}
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": comment.ID, "html_url": comment.HTMLURL, "body": comment.Body})
}
