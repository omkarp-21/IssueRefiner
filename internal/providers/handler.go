package providers

import (
	"errors"
	"issue-refiner/internal/auth"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProviderHandler struct {
	ProviderService *ProviderService
}

func NewProviderHandler(providerService *ProviderService) ProviderHandler {
	return ProviderHandler{ProviderService: providerService}
}

type createConfigRequest struct {
	ProviderType string   `json:"provider_type"`
	DisplayName  string   `json:"display_name"`
	APIKey       string   `json:"api_key"`
	Models       []string `json:"models"`
}

type updateConfigRequest struct {
	DisplayName *string  `json:"display_name"`
	Enabled     *bool    `json:"enabled"`
	APIKey      *string  `json:"api_key"`
	Models      []string `json:"models"`
}

func upstreamStatus(err error) int {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		return http.StatusUnauthorized
	case errors.Is(err, ErrRateLimited):
		return http.StatusTooManyRequests
	case errors.Is(err, ErrProviderUnavailable):
		return http.StatusBadGateway
	case errors.Is(err, ErrProviderTimeout):
		return http.StatusGatewayTimeout
	case errors.Is(err, ErrModelUnavailable):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func (providerHandler *ProviderHandler) CreateConfig(c *gin.Context) {
	organization, _ := auth.CurrentOrg(c)
	if organization == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req createConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	config, err := providerHandler.ProviderService.CreateConfig(c.Request.Context(), organization.ID, req.ProviderType, req.DisplayName, req.APIKey, req.Models)
	if err != nil {
		switch {
		case errors.Is(err, ErrUnknownProvider),
			errors.Is(err, ErrInvalidConfig),
			errors.Is(err, ErrModelUnavailable):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		case errors.Is(err, ErrRateLimited):
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Provider rate limited"})
		case errors.Is(err, ErrProviderUnavailable),
			errors.Is(err, ErrProviderTimeout):
			c.JSON(http.StatusBadGateway, gin.H{"error": "Provider unavailable"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create provider"})
		}
		return
	}
	c.JSON(http.StatusCreated, gin.H{"provider": config})
}

func (providerHandler *ProviderHandler) ListConfigs(c *gin.Context) {
	organization, _ := auth.CurrentOrg(c)
	if organization == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	configs, err := providerHandler.ProviderService.ListConfigs(c.Request.Context(), organization.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list providers"})
		return
	}
	items := make([]gin.H, 0, len(configs))
	for _, config := range configs {
		models, err := providerHandler.ProviderService.Configs.ListModels(c.Request.Context(), organization.ID, config.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list providers"})
			return
		}
		items = append(items, gin.H{"provider": config, "models": models})
	}
	c.JSON(http.StatusOK, gin.H{"providers": items})
}

func (providerHandler *ProviderHandler) GetConfig(c *gin.Context) {
	organization, _ := auth.CurrentOrg(c)
	if organization == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider id"})
		return
	}
	config, models, err := providerHandler.ProviderService.GetConfig(c.Request.Context(), organization.ID, id)
	if err != nil {
		switch {
		case errors.Is(err, ErrConfigNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch provider"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": config, "models": models})
}

func (providerHandler *ProviderHandler) UpdateConfig(c *gin.Context) {
	organization, _ := auth.CurrentOrg(c)
	if organization == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider id"})
		return
	}
	var req updateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	existing, _, err := providerHandler.ProviderService.GetConfig(c.Request.Context(), organization.ID, id)
	if err != nil {
		switch {
		case errors.Is(err, ErrConfigNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch provider"})
		}
		return
	}
	displayName := existing.DisplayName
	if req.DisplayName != nil {
		displayName = *req.DisplayName
	}
	enabled := existing.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	apiKey := ""
	if req.APIKey != nil {
		apiKey = *req.APIKey
	}
	config, err := providerHandler.ProviderService.UpdateConfig(c.Request.Context(), organization.ID, id, displayName, enabled, apiKey, req.Models, req.Models != nil)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidConfig),
			errors.Is(err, ErrModelUnavailable):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		case errors.Is(err, ErrRateLimited):
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Provider rate limited"})
		case errors.Is(err, ErrProviderUnavailable),
			errors.Is(err, ErrProviderTimeout):
			c.JSON(http.StatusBadGateway, gin.H{"error": "Provider unavailable"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update provider"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": config})
}

func (providerHandler *ProviderHandler) DeleteConfig(c *gin.Context) {
	organization, _ := auth.CurrentOrg(c)
	if organization == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider id"})
		return
	}
	if err := providerHandler.ProviderService.DeleteConfig(c.Request.Context(), organization.ID, id); err != nil {
		switch {
		case errors.Is(err, ErrConfigNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		case errors.Is(err, ErrConfigInUse):
			c.JSON(http.StatusConflict, gin.H{"error": "Provider is in use by jobs"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete provider"})
		}
		return
	}
	c.Status(http.StatusNoContent)
}

func (providerHandler *ProviderHandler) TestConfig(c *gin.Context) {
	organization, _ := auth.CurrentOrg(c)
	if organization == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider id"})
		return
	}
	config, err := providerHandler.ProviderService.TestConfig(c.Request.Context(), organization.ID, id)
	if err != nil {
		switch {
		case errors.Is(err, ErrConfigNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		case errors.Is(err, ErrInvalidConfig):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(upstreamStatus(err), gin.H{"error": safeProviderMessage(err)})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": config})
}

func safeProviderMessage(err error) string {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		return "Invalid credentials"
	case errors.Is(err, ErrInsufficientQuota):
		return "Provider quota exhausted"
	case errors.Is(err, ErrRateLimited):
		return "Provider rate limited"
	case errors.Is(err, ErrModelUnavailable):
		return "Model unavailable"
	case errors.Is(err, ErrProviderUnavailable):
		return "Provider unavailable"
	case errors.Is(err, ErrProviderTimeout):
		return "Provider timed out"
	case errors.Is(err, ErrRequestRejected):
		return "Provider rejected the request"
	default:
		return "Provider request failed"
	}
}

func (providerHandler *ProviderHandler) RegisterRoutes(router *gin.Engine, requireAuth gin.HandlerFunc, requireOwner gin.HandlerFunc) {
	api := router.Group("/api", requireAuth)
	api.GET("/providers", providerHandler.ListConfigs)
	api.GET("/providers/:id", providerHandler.GetConfig)
	admin := api.Group("", requireOwner)
	admin.POST("/providers", providerHandler.CreateConfig)
	admin.PATCH("/providers/:id", providerHandler.UpdateConfig)
	admin.DELETE("/providers/:id", providerHandler.DeleteConfig)
	admin.POST("/providers/:id/test", providerHandler.TestConfig)
}
