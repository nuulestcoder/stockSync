package middleware

import (
	"net/http"
	"strings"

	"github.com/adapolytech/stock-manager/pkg/jwt"
	"github.com/gin-gonic/gin"
)

const claimsKey = "claims"

func Auth(accessSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "missing bearer token"})
			return
		}

		claims, err := jwt.ParseAccessToken(strings.TrimSpace(strings.TrimPrefix(header, prefix)), accessSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "invalid or expired token"})
			return
		}

		c.Set(claimsKey, claims)
		c.Next()
	}
}

func Claims(c *gin.Context) (*jwt.Claims, bool) {
	value, exists := c.Get(claimsKey)
	if !exists {
		return nil, false
	}
	claims, ok := value.(*jwt.Claims)
	return claims, ok
}
