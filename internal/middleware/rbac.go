package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RequireRole(allowed ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleValue, exists := c.Get("role")
		role, ok := roleValue.(string)
		if !exists || !ok || role == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "unauthenticated", "message": "Authentication required"}})
			return
		}
		for _, allowedRole := range allowed {
			if role == allowedRole {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "forbidden", "message": "Insufficient role"}})
	}
}
