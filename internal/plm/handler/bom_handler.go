package handler

import (
	"net/http"

	"github.com/bitfantasy/nimo/internal/plm/service"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type BOMHandler struct {
	svc *service.ProjectBOMService
}

func NewBOMHandler(svc *service.ProjectBOMService) *BOMHandler {
	return &BOMHandler{svc: svc}
}

// ListBOMs GET /projects/:id/boms
func (h *BOMHandler) ListBOMs(c *gin.Context) {
	projectID := c.Param("id")
	bomType := c.Query("bom_type")
	status := c.Query("status")

	boms, err := h.svc.ListBOMs(c.Request.Context(), projectID, bomType, status)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, boms)
}

// GetBOM GET /projects/:id/boms/:bomId
func (h *BOMHandler) GetBOM(c *gin.Context) {
	bomID := c.Param("bomId")

	bom, err := h.svc.GetBOM(c.Request.Context(), bomID)
	if err != nil {
		NotFound(c, "BOM not found")
		return
	}

	Success(c, bom)
}

// CreateBOM POST /projects/:id/boms
func (h *BOMHandler) CreateBOM(c *gin.Context) {
	projectID := c.Param("id")
	var input service.CreateBOMInput
	if err := c.ShouldBindJSON(&input); err != nil {
		BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	userID := c.GetString("user_id")
	bom, err := h.svc.CreateBOM(c.Request.Context(), projectID, &input, userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Created(c, bom)
}

// UpdateBOM PUT /projects/:id/boms/:bomId
func (h *BOMHandler) UpdateBOM(c *gin.Context) {
	bomID := c.Param("bomId")
	var input service.UpdateBOMInput
	if err := c.ShouldBindJSON(&input); err != nil {
		BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	bom, err := h.svc.UpdateBOM(c.Request.Context(), bomID, &input)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, bom)
}

// SubmitBOM POST /projects/:id/boms/:bomId/submit
func (h *BOMHandler) SubmitBOM(c *gin.Context) {
	bomID := c.Param("bomId")
	userID := c.GetString("user_id")

	bom, err := h.svc.SubmitBOM(c.Request.Context(), bomID, userID)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, bom)
}

// ApproveBOM POST /projects/:id/boms/:bomId/approve
func (h *BOMHandler) ApproveBOM(c *gin.Context) {
	bomID := c.Param("bomId")
	userID := c.GetString("user_id")

	var input struct {
		Comment string `json:"comment"`
	}
	c.ShouldBindJSON(&input)

	bom, err := h.svc.ApproveBOM(c.Request.Context(), bomID, userID, input.Comment)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, bom)
}

// RejectBOM POST /projects/:id/boms/:bomId/reject
func (h *BOMHandler) RejectBOM(c *gin.Context) {
	bomID := c.Param("bomId")
	userID := c.GetString("user_id")

	var input struct {
		Comment string `json:"comment" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		BadRequest(c, "请填写驳回原因")
		return
	}

	bom, err := h.svc.RejectBOM(c.Request.Context(), bomID, userID, input.Comment)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, bom)
}

// FreezeBOM POST /projects/:id/boms/:bomId/freeze
func (h *BOMHandler) FreezeBOM(c *gin.Context) {
	bomID := c.Param("bomId")
	userID := c.GetString("user_id")

	bom, err := h.svc.FreezeBOM(c.Request.Context(), bomID, userID)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, bom)
}

// AddItem POST /projects/:id/boms/:bomId/items
func (h *BOMHandler) AddItem(c *gin.Context) {
	bomID := c.Param("bomId")
	var input service.BOMItemInput
	if err := c.ShouldBindJSON(&input); err != nil {
		BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	item, err := h.svc.AddItem(c.Request.Context(), bomID, &input)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	Created(c, item)
}

// BatchAddItems POST /projects/:id/boms/:bomId/items/batch
func (h *BOMHandler) BatchAddItems(c *gin.Context) {
	bomID := c.Param("bomId")
	var input struct {
		Items []service.BOMItemInput `json:"items" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	count, err := h.svc.BatchAddItems(c.Request.Context(), bomID, input.Items)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, gin.H{"created": count})
}

// UpdateItem PUT /projects/:id/boms/:bomId/items/:itemId
func (h *BOMHandler) UpdateItem(c *gin.Context) {
	bomID := c.Param("bomId")
	itemID := c.Param("itemId")
	var input service.BOMItemInput
	if err := c.ShouldBindJSON(&input); err != nil {
		BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	item, err := h.svc.UpdateItem(c.Request.Context(), bomID, itemID, &input)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, item)
}

// ReorderItems POST /projects/:id/boms/:bomId/reorder
func (h *BOMHandler) ReorderItems(c *gin.Context) {
	bomID := c.Param("bomId")
	var input service.ReorderItemsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if err := h.svc.ReorderItems(c.Request.Context(), bomID, input.ItemIDs); err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, gin.H{"reordered": true})
}

// DeleteItem DELETE /projects/:id/boms/:bomId/items/:itemId
func (h *BOMHandler) DeleteItem(c *gin.Context) {
	bomID := c.Param("bomId")
	itemID := c.Param("itemId")

	if err := h.svc.DeleteItem(c.Request.Context(), bomID, itemID); err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, gin.H{"deleted": true})
}

// ==================== Phase 2: Excel 导入/导出 ====================

// ExportBOM GET /projects/:id/boms/:bomId/export
func (h *BOMHandler) ExportBOM(c *gin.Context) {
	bomID := c.Param("bomId")

	f, filename, err := h.svc.ExportBOM(c.Request.Context(), bomID)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	defer f.Close()

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Header("Content-Transfer-Encoding", "binary")

	if err := f.Write(c.Writer); err != nil {
		InternalError(c, "write excel: "+err.Error())
	}
}

// ImportBOM POST /projects/:id/boms/:bomId/import
func (h *BOMHandler) ImportBOM(c *gin.Context) {
	bomID := c.Param("bomId")

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		BadRequest(c, "请上传Excel文件")
		return
	}
	defer file.Close()

	f, err := excelize.OpenReader(file)
	if err != nil {
		BadRequest(c, "无法解析Excel文件: "+err.Error())
		return
	}
	defer f.Close()

	result, err := h.svc.ImportBOM(c.Request.Context(), bomID, f)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, result)
}

// DownloadTemplate GET /api/v1/bom-template
func (h *BOMHandler) DownloadTemplate(c *gin.Context) {
	f, err := h.svc.GenerateTemplate()
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	defer f.Close()

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=\"BOM_Import_Template.xlsx\"")
	c.Header("Content-Transfer-Encoding", "binary")

	if err := f.Write(c.Writer); err != nil {
		InternalError(c, "write template: "+err.Error())
	}
}

// ==================== Phase 3: EBOM→MBOM转换 + 版本对比 ====================

// ConvertToMBOM POST /projects/:id/boms/:bomId/convert-to-mbom
func (h *BOMHandler) ConvertToMBOM(c *gin.Context) {
	bomID := c.Param("bomId")
	userID := c.GetString("user_id")

	mbom, err := h.svc.ConvertToMBOM(c.Request.Context(), bomID, userID)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	Created(c, mbom)
}

// CompareBOMs GET /api/v1/bom-compare?bom1=ID1&bom2=ID2
func (h *BOMHandler) CompareBOMs(c *gin.Context) {
	bom1 := c.Query("bom1")
	bom2 := c.Query("bom2")

	if bom1 == "" || bom2 == "" {
		BadRequest(c, "请提供bom1和bom2参数")
		return
	}

	result, err := h.svc.CompareBOMs(c.Request.Context(), bom1, bom2)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, result)
}

// ==================== Phase 4: ERP对接桥梁 ====================

// ListBOMReleases GET /api/v1/erp/bom-releases
func (h *BOMHandler) ListBOMReleases(c *gin.Context) {
	releases, err := h.svc.ListPendingReleases(c.Request.Context())
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    gin.H{"items": releases, "total": len(releases)},
	})
}

// AckBOMRelease POST /api/v1/erp/bom-releases/:id/ack
func (h *BOMHandler) AckBOMRelease(c *gin.Context) {
	releaseID := c.Param("id")

	release, err := h.svc.AckRelease(c.Request.Context(), releaseID)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, release)
}
