package health

import "github.com/gin-gonic/gin"

func displayStatus(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "ok",
	})
}

func RegisterRoutes(router *gin.Engine) {
	router.GET("/health", displayStatus)
}
