package handler

import (
	"github.com/bitfantasy/nimo/internal/plm/service"
	"github.com/gin-gonic/gin"
)

type SKUHandler struct {
	svc *service.SKUService
}

func NewSKUHandler(svc *service.SKUService) *SKUHandler {
	return &SKUHandler{svc: svc}
}

// ListSKUs GET /projects/:id/skus
func (h *SKUHandler) ListSKUs(c *gin.Context) {
	projectID := c.Param("id")
	skus, err := h.svc.ListSKUs(c.Request.Context(), projectID)
	if err != nil {
		InternalError(c, "获取SKU列表失败: "+err.Error())
		return
	}
	Success(c, gin.H{"items": skus})
}

// CreateSKU POST /projects/:id/skus
func (h *SKUHandler) CreateSKU(c *gin.Context) {
	projectID := c.Param("id")
	var input service.CreateSKUInput
	if err := c.ShouldBindJSON(&input); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}
	userID := GetUserID(c)
	sku, err := h.svc.CreateSKU(c.Request.Context(), projectID, &input, userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Created(c, sku)
}

// UpdateSKU PUT /projects/:id/skus/:skuId
func (h *SKUHandler) UpdateSKU(c *gin.Context) {
	skuID := c.Param("skuId")
	var input service.UpdateSKUInput
	if err := c.ShouldBindJSON(&input); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}
	sku, err := h.svc.UpdateSKU(c.Request.Context(), skuID, &input)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, sku)
}

// DeleteSKU DELETE /projects/:id/skus/:skuId
func (h *SKUHandler) DeleteSKU(c *gin.Context) {
	skuID := c.Param("skuId")
	if err := h.svc.DeleteSKU(c.Request.Context(), skuID); err != nil {
		InternalError(c, "删除SKU失败: "+err.Error())
		return
	}
	Success(c, nil)
}

// GetCMFConfigs GET /projects/:id/skus/:skuId/cmf
func (h *SKUHandler) GetCMFConfigs(c *gin.Context) {
	skuID := c.Param("skuId")
	configs, err := h.svc.GetCMFConfigs(c.Request.Context(), skuID)
	if err != nil {
		InternalError(c, "获取CMF配置失败: "+err.Error())
		return
	}
	Success(c, gin.H{"items": configs})
}

// BatchSaveCMFConfigs PUT /projects/:id/skus/:skuId/cmf
func (h *SKUHandler) BatchSaveCMFConfigs(c *gin.Context) {
	skuID := c.Param("skuId")
	var inputs []service.CMFConfigInput
	if err := c.ShouldBindJSON(&inputs); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}
	configs, err := h.svc.BatchSaveCMFConfigs(c.Request.Context(), skuID, inputs)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, gin.H{"items": configs})
}

// GetBOMOverrides GET /projects/:id/skus/:skuId/bom-overrides
func (h *SKUHandler) GetBOMOverrides(c *gin.Context) {
	skuID := c.Param("skuId")
	overrides, err := h.svc.GetBOMOverrides(c.Request.Context(), skuID)
	if err != nil {
		InternalError(c, "获取BOM差异失败: "+err.Error())
		return
	}
	Success(c, gin.H{"items": overrides})
}

// CreateBOMOverride POST /projects/:id/skus/:skuId/bom-overrides
func (h *SKUHandler) CreateBOMOverride(c *gin.Context) {
	skuID := c.Param("skuId")
	var input service.BOMOverrideInput
	if err := c.ShouldBindJSON(&input); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}
	override, err := h.svc.CreateBOMOverride(c.Request.Context(), skuID, &input)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Created(c, override)
}

// UpdateBOMOverride PUT /projects/:id/skus/:skuId/bom-overrides/:overrideId
func (h *SKUHandler) UpdateBOMOverride(c *gin.Context) {
	overrideID := c.Param("overrideId")
	var input service.BOMOverrideInput
	if err := c.ShouldBindJSON(&input); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}
	override, err := h.svc.UpdateBOMOverride(c.Request.Context(), overrideID, &input)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, override)
}

// DeleteBOMOverride DELETE /projects/:id/skus/:skuId/bom-overrides/:overrideId
func (h *SKUHandler) DeleteBOMOverride(c *gin.Context) {
	overrideID := c.Param("overrideId")
	if err := h.svc.DeleteBOMOverride(c.Request.Context(), overrideID); err != nil {
		InternalError(c, "删除BOM差异失败: "+err.Error())
		return
	}
	Success(c, nil)
}

// GetFullBOM GET /projects/:id/skus/:skuId/full-bom
func (h *SKUHandler) GetFullBOM(c *gin.Context) {
	projectID := c.Param("id")
	skuID := c.Param("skuId")
	items, err := h.svc.GetFullBOM(c.Request.Context(), skuID, projectID)
	if err != nil {
		InternalError(c, "获取完整BOM失败: "+err.Error())
		return
	}
	Success(c, gin.H{"items": items})
}
