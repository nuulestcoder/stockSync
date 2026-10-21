package handler

import (
	"context"
	"net/http"

	"github.com/adapolytech/stock-manager/internal/domain"
	"github.com/adapolytech/stock-manager/internal/repository"
	"github.com/adapolytech/stock-manager/internal/service"
	"github.com/adapolytech/stock-manager/pkg/response"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type StockHandler struct {
	service *service.StockService
}

func NewStockHandler(stockService *service.StockService) *StockHandler {
	return &StockHandler{service: stockService}
}

func (h *StockHandler) List(c *gin.Context) {
	companyID, ok := companyIDFromContext(c)
	if !ok {
		return
	}
	result, err := h.service.List(c.Request.Context(), companyID, domain.StockFilter{
		Page: parsePositiveInt(c.Query("page"), 1), Limit: parsePositiveInt(c.Query("limit"), 20),
		Location: c.Query("location"),
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, http.StatusOK, result, "stock retrieved successfully")
}

func (h *StockHandler) Alerts(c *gin.Context) {
	companyID, ok := companyIDFromContext(c)
	if !ok {
		return
	}
	result, err := h.service.Alerts(c.Request.Context(), companyID, parsePositiveInt(c.Query("page"), 1), parsePositiveInt(c.Query("limit"), 20))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, http.StatusOK, result, "stock alerts retrieved successfully")
}

func (h *StockHandler) Dashboard(c *gin.Context) {
	companyID, ok := companyIDFromContext(c)
	if !ok {
		return
	}
	dashboard, err := h.service.Dashboard(c.Request.Context(), companyID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, http.StatusOK, dashboard, "stock dashboard retrieved successfully")
}

func (h *StockHandler) Get(c *gin.Context) {
	companyID, variantID, ok := h.ids(c)
	if !ok {
		return
	}
	stock, err := h.service.Get(c.Request.Context(), companyID, variantID)
	if err != nil {
		response.Error(c, http.StatusNotFound, err.Error())
		return
	}
	response.Success(c, http.StatusOK, stock, "stock retrieved successfully")
}

func (h *StockHandler) Movements(c *gin.Context) {
	companyID, variantID, ok := h.ids(c)
	if !ok {
		return
	}
	result, err := h.service.Movements(c.Request.Context(), companyID, variantID, parsePositiveInt(c.Query("page"), 1), parsePositiveInt(c.Query("limit"), 20))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, http.StatusOK, result, "stock movements retrieved successfully")
}

func (h *StockHandler) Move(c *gin.Context) {
	companyID, variantID, ok := h.ids(c)
	if !ok {
		return
	}
	userID, ok := userIDFromContext(c)
	if !ok {
		return
	}
	var request domain.MoveStockRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid stock movement request")
		return
	}
	result, err := h.service.Move(c.Request.Context(), companyID, userID, variantID, &request)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, http.StatusOK, result, "stock movement applied successfully")
}

func (h *StockHandler) CreateInbound(c *gin.Context) {
	h.createRequest(c, domain.StockRequestInbound)
}

func (h *StockHandler) CreateOutbound(c *gin.Context) {
	h.createRequest(c, domain.StockRequestOutbound)
}

func (h *StockHandler) createRequest(c *gin.Context, requestType domain.StockRequestType) {
	companyID, ok := companyIDFromContext(c)
	if !ok {
		return
	}
	userID, ok := userIDFromContext(c)
	if !ok {
		return
	}
	var request domain.CreateStockRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid stock request")
		return
	}
	created, err := h.service.CreateRequest(c.Request.Context(), companyID, userID, requestType, &request)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, http.StatusCreated, created, "stock request created successfully")
}

func (h *StockHandler) ListRequests(c *gin.Context) {
	companyID, ok := companyIDFromContext(c)
	if !ok {
		return
	}
	result, err := h.service.ListRequests(c.Request.Context(), companyID, repository.StockRequestFilter{
		Type: domain.StockRequestType(c.Query("type")), Status: domain.StockRequestStatus(c.Query("status")),
		Page: parsePositiveInt(c.Query("page"), 1), Limit: parsePositiveInt(c.Query("limit"), 20),
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, http.StatusOK, result, "stock requests retrieved successfully")
}

func (h *StockHandler) Approve(c *gin.Context) { h.transition(c, h.service.ApproveRequest, "approved") }
func (h *StockHandler) Reject(c *gin.Context)  { h.transition(c, h.service.RejectRequest, "rejected") }
func (h *StockHandler) Cancel(c *gin.Context)  { h.transition(c, h.service.CancelRequest, "cancelled") }
func (h *StockHandler) Complete(c *gin.Context) {
	h.transition(c, h.service.CompleteRequest, "completed")
}

type requestTransition func(context.Context, bson.ObjectID, bson.ObjectID, bson.ObjectID) (*domain.StockRequest, error)

func (h *StockHandler) transition(c *gin.Context, transition requestTransition, message string) {
	companyID, ok := companyIDFromContext(c)
	if !ok {
		return
	}
	requestID, ok := parseObjectID(c.Param("id"))
	if !ok {
		response.Error(c, http.StatusBadRequest, "invalid stock request ID")
		return
	}
	userID, ok := userIDFromContext(c)
	if !ok {
		return
	}
	request, err := transition(c.Request.Context(), companyID, requestID, userID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, http.StatusOK, request, "stock request "+message+" successfully")
}

func (h *StockHandler) ids(c *gin.Context) (bson.ObjectID, bson.ObjectID, bool) {
	companyID, ok := companyIDFromContext(c)
	if !ok {
		return bson.NilObjectID, bson.NilObjectID, false
	}
	variantID, ok := parseObjectID(c.Param("variantId"))
	if !ok {
		response.Error(c, http.StatusBadRequest, "invalid variant ID")
		return bson.NilObjectID, bson.NilObjectID, false
	}
	return companyID, variantID, true
}

func userIDFromContext(c *gin.Context) (bson.ObjectID, bool) {
	value, exists := c.Get("userID")
	userID, ok := value.(bson.ObjectID)
	if !exists || !ok || userID.IsZero() {
		response.Error(c, http.StatusUnauthorized, "user context is missing")
		return bson.NilObjectID, false
	}
	return userID, true
}
