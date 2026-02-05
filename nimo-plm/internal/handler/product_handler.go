package handler

import (
	"strconv"

	"github.com/bitfantasy/nimo-plm/internal/service"
	"github.com/gin-gonic/gin"
)

// ProductHandler 产品处理器
type ProductHandler struct {
	svc *service.ProductService
}

// NewProductHandler 创建产品处理器
func NewProductHandler(svc *service.ProductService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

// List 获取产品列表
func (h *ProductHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	filters := map[string]interface{}{
		"keyword":     c.Query("keyword"),
		"category_id": c.Query("category_id"),
		"status":      c.Query("status"),
		"created_by":  c.Query("created_by"),
	}

	result, err := h.svc.List(c.Request.Context(), page, pageSize, filters)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, result)
}

// Create 创建产品
func (h *ProductHandler) Create(c *gin.Context) {
	var req service.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	userID := GetUserID(c)
	product, err := h.svc.Create(c.Request.Context(), userID, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Created(c, product)
}

// Get 获取产品详情
func (h *ProductHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequest(c, "Product ID is required")
		return
	}

	product, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "Product not found")
		return
	}

	Success(c, product)
}

// Update 更新产品
func (h *ProductHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequest(c, "Product ID is required")
		return
	}

	var req service.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	userID := GetUserID(c)
	product, err := h.svc.Update(c.Request.Context(), id, userID, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, product)
}

// Delete 删除产品
func (h *ProductHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequest(c, "Product ID is required")
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, nil)
}

// Release 发布产品
func (h *ProductHandler) Release(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequest(c, "Product ID is required")
		return
	}

	userID := GetUserID(c)
	product, err := h.svc.Release(c.Request.Context(), id, userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, product)
}

// ListCategories 获取产品类别列表
func (h *ProductHandler) ListCategories(c *gin.Context) {
	categories, err := h.svc.GetCategories(c.Request.Context())
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, categories)
}
