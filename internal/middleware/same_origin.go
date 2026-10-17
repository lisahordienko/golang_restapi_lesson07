package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RequireSameOrigin() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		if origin == "" || origin != scheme+"://"+c.Request.Host {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}
