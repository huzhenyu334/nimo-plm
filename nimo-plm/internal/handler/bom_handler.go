package handler

import (
	"strconv"

	"github.com/bitfantasy/nimo-plm/internal/service"
	"github.com/gin-gonic/gin"
)

// BOMHandler BOM处理器
type BOMHandler struct {
	svc *service.BOMService
}

// NewBOMHandler 创建BOM处理器
func NewBOMHandler(svc *service.BOMService) *BOMHandler {
	return &BOMHandler{svc: svc}
}

// Get 获取产品BOM
func (h *BOMHandler) Get(c *gin.Context) {
	productID := c.Param("id")
	if productID == "" {
		BadRequest(c, "Product ID is required")
		return
	}

	version := c.Query("version")
	expandLevel, _ := strconv.Atoi(c.DefaultQuery("expand_level", "-1"))
	includeCost := c.Query("include_cost") == "true"

	bom, err := h.svc.GetBOM(c.Request.Context(), productID, version, expandLevel, includeCost)
	if err != nil {
		NotFound(c, "BOM not found: "+err.Error())
		return
	}

	Success(c, bom)
}

// ListVersions 获取BOM版本列表
func (h *BOMHandler) ListVersions(c *gin.Context) {
	productID := c.Param("id")
	if productID == "" {
		BadRequest(c, "Product ID is required")
		return
	}

	versions, err := h.svc.ListVersions(c.Request.Context(), productID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, versions)
}

// AddItem 添加BOM行项
func (h *BOMHandler) AddItem(c *gin.Context) {
	productID := c.Param("id")
	if productID == "" {
		BadRequest(c, "Product ID is required")
		return
	}

	var req service.AddBOMItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	userID := GetUserID(c)
	item, err := h.svc.AddItem(c.Request.Context(), productID, userID, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Created(c, item)
}

// UpdateItem 更新BOM行项
func (h *BOMHandler) UpdateItem(c *gin.Context) {
	productID := c.Param("id")
	itemID := c.Param("itemId")
	if productID == "" || itemID == "" {
		BadRequest(c, "Product ID and Item ID are required")
		return
	}

	var req service.UpdateBOMItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	userID := GetUserID(c)
	item, err := h.svc.UpdateItem(c.Request.Context(), productID, itemID, userID, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, item)
}

// DeleteItem 删除BOM行项
func (h *BOMHandler) DeleteItem(c *gin.Context) {
	productID := c.Param("id")
	itemID := c.Param("itemId")
	if productID == "" || itemID == "" {
		BadRequest(c, "Product ID and Item ID are required")
		return
	}

	userID := GetUserID(c)
	if err := h.svc.DeleteItem(c.Request.Context(), productID, itemID, userID); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, nil)
}

// Release 发布BOM
func (h *BOMHandler) Release(c *gin.Context) {
	productID := c.Param("id")
	if productID == "" {
		BadRequest(c, "Product ID is required")
		return
	}

	var req service.ReleaseBOMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	userID := GetUserID(c)
	bom, err := h.svc.Release(c.Request.Context(), productID, userID, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, bom)
}

// Compare 对比BOM版本
func (h *BOMHandler) Compare(c *gin.Context) {
	productID := c.Param("id")
	versionA := c.Query("version_a")
	versionB := c.Query("version_b")

	if productID == "" || versionA == "" || versionB == "" {
		BadRequest(c, "Product ID and both versions are required")
		return
	}

	result, err := h.svc.Compare(c.Request.Context(), productID, versionA, versionB)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, result)
}
