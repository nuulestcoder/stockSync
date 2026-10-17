package handler

import (
	"net/http"

	"github.com/adapolytech/stock-manager/internal/domain"
	"github.com/adapolytech/stock-manager/internal/service"
	"github.com/adapolytech/stock-manager/pkg/response"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var request domain.RegisterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid registration request")
		return
	}

	user, err := h.authService.Register(c.Request.Context(), &request)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, gin.H{
		"userId":    user.ID,
		"companyId": user.CompanyID,
		"email":     user.Email,
		"role":      user.Role,
	}, "registration successful")
}
