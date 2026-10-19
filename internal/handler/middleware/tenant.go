package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	companyIDKey = "companyID"
	userIDKey    = "userID"
	roleKey      = "role"
)

func Tenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := Claims(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "authentication claims are missing"})
			return
		}

		c.Set(companyIDKey, claims.CompanyID)
		c.Set(userIDKey, claims.UserID)
		c.Set(roleKey, claims.Role)
		c.Next()
	}
}
