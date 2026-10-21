package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/adapolytech/stock-manager/internal/domain"
	"github.com/adapolytech/stock-manager/internal/repository"
	"github.com/adapolytech/stock-manager/internal/service"
	"github.com/adapolytech/stock-manager/pkg/response"
	"github.com/gin-gonic/gin"
)

type SaleHandler struct {
	service *service.SaleService
}

func NewSaleHandler(saleService *service.SaleService) *SaleHandler {
	return &SaleHandler{service: saleService}
}

func (h *SaleHandler) Create(c *gin.Context) {
	companyID, ok := companyIDFromContext(c)
	if !ok {
		return
	}
	userID, ok := userIDFromContext(c)
	if !ok {
		return
	}
	var request domain.CreateSaleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid sale request")
		return
	}
	sale, err := h.service.Create(c.Request.Context(), companyID, userID, &request)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, http.StatusCreated, sale, "sale created successfully")
}

func (h *SaleHandler) List(c *gin.Context) {
	companyID, ok := companyIDFromContext(c)
	if !ok {
		return
	}
	result, err := h.service.List(c.Request.Context(), companyID, repository.SaleFilter{
		Status: domain.SaleStatus(c.Query("status")),
		Page:   parsePositiveInt(c.Query("page"), 1), Limit: parsePositiveInt(c.Query("limit"), 20),
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, http.StatusOK, gin.H{
		"items": result.Sales,
		"meta":  gin.H{"total": result.Total, "page": result.Page, "limit": result.Limit, "totalPages": result.TotalPages},
	}, "sales retrieved successfully")
}

func (h *SaleHandler) Get(c *gin.Context) {
	companyID, ok := companyIDFromContext(c)
	if !ok {
		return
	}
	saleID, ok := parseObjectID(c.Param("id"))
	if !ok {
		response.Error(c, http.StatusBadRequest, "invalid sale ID")
		return
	}
	sale, err := h.service.GetByID(c.Request.Context(), companyID, saleID)
	if err != nil {
		response.Error(c, http.StatusNotFound, err.Error())
		return
	}
	response.Success(c, http.StatusOK, sale, "sale retrieved successfully")
}

func (h *SaleHandler) Transition(c *gin.Context) {
	companyID, ok := companyIDFromContext(c)
	if !ok {
		return
	}
	userID, ok := userIDFromContext(c)
	if !ok {
		return
	}
	saleID, ok := parseObjectID(c.Param("id"))
	if !ok {
		response.Error(c, http.StatusBadRequest, "invalid sale ID")
		return
	}
	var request domain.UpdateSaleStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil || !request.Status.IsValid() {
		response.Error(c, http.StatusBadRequest, "invalid sale status request")
		return
	}
	sale, err := h.service.Transition(c.Request.Context(), companyID, saleID, userID, request.Status)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, http.StatusOK, sale, "sale status updated successfully")
}

func (h *SaleHandler) Dashboard(c *gin.Context) {
	companyID, ok := companyIDFromContext(c)
	if !ok {
		return
	}
	filter, err := saleReportFilter(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	dashboard, err := h.service.Dashboard(c.Request.Context(), companyID, filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, http.StatusOK, dashboard, "sales dashboard retrieved successfully")
}

func saleReportFilter(c *gin.Context) (repository.SaleReportFilter, error) {
	filter := repository.SaleReportFilter{}
	if value := c.Query("from"); value != "" {
		from, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return filter, fmt.Errorf("invalid from date, expected RFC3339")
		}
		filter.From = &from
	}
	if value := c.Query("to"); value != "" {
		to, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return filter, fmt.Errorf("invalid to date, expected RFC3339")
		}
		filter.To = &to
	}
	if filter.From != nil && filter.To != nil && !filter.From.Before(*filter.To) {
		return filter, fmt.Errorf("from date must be before to date")
	}
	return filter, nil
}
