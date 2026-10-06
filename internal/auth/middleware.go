package auth

import (
	"issue-refiner/internal/orgs"
	"issue-refiner/internal/sessions"
	"issue-refiner/internal/users"
	"net/http"

	"github.com/gin-gonic/gin"
)

const sessionCookie = "session_token"

func RequireAuth(sessionService *sessions.SessionService, userService *users.UserService, organizationService *orgs.OrganizationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(sessionCookie)
		if err != nil || token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		session, err := sessionService.GetSession(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		user, err := userService.UserRepository.FindByID(c.Request.Context(), session.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		organization, role, err := organizationService.ResolveOrg(c.Request.Context(), user.ID, session.OrganizationID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		if session.OrganizationID == nil {
			_ = sessionService.SetOrganization(c.Request.Context(), session.ID, organization.ID)
		}
		c.Set("user", user)
		c.Set("organization", organization)
		c.Set("role", role)
		c.Next()
	}
}

func CurrentUser(c *gin.Context) *users.User {
	user, _ := c.Get("user")
	if user == nil {
		return nil
	}
	if u, ok := user.(*users.User); ok {
		return u
	}
	return nil
}

func CurrentOrg(c *gin.Context) (*orgs.Organization, string) {
	organization, _ := c.Get("organization")
	o, ok := organization.(*orgs.Organization)
	if !ok || o == nil {
		return nil, ""
	}
	role, _ := c.Get("role")
	r, _ := role.(string)
	return o, r
}

func RequireOwner() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, role := CurrentOrg(c)
		if role != orgs.RoleOwner {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
			c.Abort()
			return
		}
		c.Next()
	}
}
