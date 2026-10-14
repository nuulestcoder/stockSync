package handler

import (
	"net/http"

	"github.com/adapolytech/stock-manager/pkg/response"
	"github.com/gin-gonic/gin"
)

func HealthCheck(c *gin.Context) {
	response.Success(c, http.StatusOK, gin.H{
		"status": "ok",
	}, "health check passed")
}
