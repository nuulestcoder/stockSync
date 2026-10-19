package handler

import (
	"net/http"
	"strconv"

	"github.com/adapolytech/stock-manager/internal/domain"
	"github.com/adapolytech/stock-manager/internal/repository"
	"github.com/adapolytech/stock-manager/internal/service"
	"github.com/adapolytech/stock-manager/pkg/response"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type ProductHandler struct {
	productService *service.ProductService
}

func NewProductHandler(productService *service.ProductService) *ProductHandler {
	return &ProductHandler{productService: productService}
}

func (h *ProductHandler) List(c *gin.Context) {
	companyID, ok := companyIDFromContext(c)
	if !ok {
		return
	}
	filter := repository.ProductFilter{
		Page:     parsePositiveInt(c.Query("page"), 1),
		Limit:    parsePositiveInt(c.Query("limit"), 20),
		Search:   c.Query("search"),
		Brand:    c.Query("brand"),
		Category: c.Query("category"),
	}
	result, err := h.productService.List(c.Request.Context(), companyID, filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, http.StatusOK, gin.H{
		"items": result.Products,
		"meta": gin.H{
			"total": result.Total, "page": result.Page, "limit": result.Limit, "totalPages": result.TotalPages,
		},
	}, "products retrieved successfully")
}

func (h *ProductHandler) Create(c *gin.Context) {
	companyID, ok := companyIDFromContext(c)
	if !ok {
		return
	}
	var request domain.CreateProductRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid product request")
		return
	}
	product, err := h.productService.Create(c.Request.Context(), companyID, &request)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, http.StatusCreated, product, "product created successfully")
}

func (h *ProductHandler) GetByID(c *gin.Context) {
	companyID, ok := companyIDFromContext(c)
	if !ok {
		return
	}
	productID, ok := parseObjectID(c.Param("id"))
	if !ok {
		return
	}
	product, err := h.productService.GetByID(c.Request.Context(), companyID, productID)
	if err != nil {
		response.Error(c, http.StatusNotFound, err.Error())
		return
	}
	response.Success(c, http.StatusOK, product, "product retrieved successfully")
}

func (h *ProductHandler) Update(c *gin.Context) {
	companyID, ok := companyIDFromContext(c)
	if !ok {
		return
	}
	productID, ok := parseObjectID(c.Param("id"))
	if !ok {
		return
	}
	var request domain.UpdateProductRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid product request")
		return
	}
	product, err := h.productService.Update(c.Request.Context(), companyID, productID, &request)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, http.StatusOK, product, "product updated successfully")
}

func (h *ProductHandler) Delete(c *gin.Context) {
	companyID, ok := companyIDFromContext(c)
	if !ok {
		return
	}
	productID, ok := parseObjectID(c.Param("id"))
	if !ok {
		return
	}
	if err := h.productService.Delete(c.Request.Context(), companyID, productID); err != nil {
		response.Error(c, http.StatusNotFound, err.Error())
		return
	}
	response.Success(c, http.StatusOK, nil, "product deleted successfully")
}

func (h *ProductHandler) AddVariant(c *gin.Context) {
	companyID, ok := companyIDFromContext(c)
	if !ok {
		return
	}
	productID, ok := parseObjectID(c.Param("id"))
	if !ok {
		return
	}
	var request domain.CreateVariantRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid variant request")
		return
	}
	product, err := h.productService.AddVariant(c.Request.Context(), companyID, productID, &request)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, http.StatusCreated, product, "variant added successfully")
}

func (h *ProductHandler) UpdateVariant(c *gin.Context) {
	companyID, ok := companyIDFromContext(c)
	if !ok {
		return
	}
	productID, ok := parseObjectID(c.Param("id"))
	if !ok {
		return
	}
	variantID, ok := parseObjectID(c.Param("variantId"))
	if !ok {
		return
	}
	var request domain.UpdateVariantRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid variant request")
		return
	}
	product, err := h.productService.UpdateVariant(c.Request.Context(), companyID, productID, variantID, &request)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, http.StatusOK, product, "variant updated successfully")
}

func (h *ProductHandler) DeleteVariant(c *gin.Context) {
	companyID, ok := companyIDFromContext(c)
	if !ok {
		return
	}
	productID, ok := parseObjectID(c.Param("id"))
	if !ok {
		return
	}
	variantID, ok := parseObjectID(c.Param("variantId"))
	if !ok {
		return
	}
	product, err := h.productService.DeleteVariant(c.Request.Context(), companyID, productID, variantID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, http.StatusOK, product, "variant deleted successfully")
}

func companyIDFromContext(c *gin.Context) (bson.ObjectID, bool) {
	value, exists := c.Get("companyID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "tenant context is missing")
		return bson.NilObjectID, false
	}
	companyID, ok := value.(bson.ObjectID)
	if !ok || companyID.IsZero() {
		response.Error(c, http.StatusUnauthorized, "invalid tenant context")
		return bson.NilObjectID, false
	}
	return companyID, true
}

func parseObjectID(value string) (bson.ObjectID, bool) {
	id, err := bson.ObjectIDFromHex(value)
	if err != nil {
		return bson.NilObjectID, false
	}
	return id, true
}

func parsePositiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}
