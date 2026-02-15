package handler

import (
	"net/http"
	"path/filepath"
	"strings"

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

// ReleaseBOM POST /projects/:id/boms/:bomId/release
func (h *BOMHandler) ReleaseBOM(c *gin.Context) {
	bomID := c.Param("bomId")
	userID := c.GetString("user_id")

	var input struct {
		ReleaseNote string `json:"release_note"`
	}
	c.ShouldBindJSON(&input)

	bom, err := h.svc.ReleaseBOM(c.Request.Context(), bomID, userID, input.ReleaseNote)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, bom)
}

// CreateFromBOM POST /projects/:id/boms/create-from
func (h *BOMHandler) CreateFromBOM(c *gin.Context) {
	projectID := c.Param("id")
	userID := c.GetString("user_id")

	var input struct {
		SourceBOMID string `json:"source_bom_id" binding:"required"`
		TargetType  string `json:"target_type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	bom, err := h.svc.CreateFromBOM(c.Request.Context(), projectID, input.SourceBOMID, input.TargetType, userID)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	Created(c, bom)
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

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		BadRequest(c, "请上传BOM文件")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))

	switch ext {
	case ".rep":
		// PADS BOM格式
		result, err := h.svc.ImportPADSBOM(c.Request.Context(), bomID, file)
		if err != nil {
			BadRequest(c, err.Error())
			return
		}
		Success(c, result)

	case ".xlsx", ".xls":
		// Excel格式
		f, err := excelize.OpenReader(file)
		if err != nil {
			BadRequest(c, "无法解析Excel文件: "+err.Error())
			return
		}
		defer f.Close()

		// 根据BOM类型路由到不同的导入方法
		bom, err := h.svc.GetBOM(c.Request.Context(), bomID)
		if err != nil {
			BadRequest(c, "BOM not found: "+err.Error())
			return
		}

		var result *service.ImportResult
		if bom.BOMType == "PBOM" {
			result, err = h.svc.ImportStructuralBOM(c.Request.Context(), bomID, f)
		} else {
			result, err = h.svc.ImportBOM(c.Request.Context(), bomID, f)
		}
		if err != nil {
			BadRequest(c, err.Error())
			return
		}
		Success(c, result)

	default:
		BadRequest(c, "不支持的文件格式，请上传 .xlsx、.xls 或 .rep 文件")
	}
}

// ParseBOM POST /api/v1/bom/parse — parse BOM file without saving (preview)
func (h *BOMHandler) ParseBOM(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		BadRequest(c, "请上传BOM文件")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))

	switch ext {
	case ".rep":
		items, err := h.svc.ParsePADSBOM(c.Request.Context(), file)
		if err != nil {
			BadRequest(c, err.Error())
			return
		}
		Success(c, gin.H{"items": items})

	case ".xlsx", ".xls":
		f, err := excelize.OpenReader(file)
		if err != nil {
			BadRequest(c, "无法解析Excel文件: "+err.Error())
			return
		}
		defer f.Close()

		items, err := h.svc.ParseExcelBOM(c.Request.Context(), f)
		if err != nil {
			BadRequest(c, err.Error())
			return
		}
		Success(c, gin.H{"items": items})

	default:
		BadRequest(c, "不支持的文件格式，请上传 .xlsx、.xls 或 .rep 文件")
	}
}

// DownloadTemplate GET /api/v1/bom-template?bom_type=SBOM
func (h *BOMHandler) DownloadTemplate(c *gin.Context) {
	bomType := c.Query("bom_type") // EBOM(默认) 或 SBOM

	f, err := h.svc.GenerateTemplate(bomType)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	defer f.Close()

	filename := "BOM_Import_Template.xlsx"
	if bomType == "PBOM" {
		filename = "PBOM_Import_Template.xlsx"
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
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

// ==================== Phase 5: 属性模板 ====================

// ListTemplates GET /api/v1/bom-templates?category=X&sub_category=Y
func (h *BOMHandler) ListTemplates(c *gin.Context) {
	category := c.Query("category")
	subCategory := c.Query("sub_category")

	templates, err := h.svc.ListTemplates(c.Request.Context(), category, subCategory)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, templates)
}

// CreateTemplate POST /api/v1/bom-templates
func (h *BOMHandler) CreateTemplate(c *gin.Context) {
	var input service.TemplateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	t, err := h.svc.CreateTemplate(c.Request.Context(), &input)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	Created(c, t)
}

// UpdateTemplate PUT /api/v1/bom-templates/:id
func (h *BOMHandler) UpdateTemplate(c *gin.Context) {
	id := c.Param("id")
	var input service.TemplateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	t, err := h.svc.UpdateTemplate(c.Request.Context(), id, &input)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	Success(c, t)
}

// DeleteTemplate DELETE /api/v1/bom-templates/:id
func (h *BOMHandler) DeleteTemplate(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.DeleteTemplate(c.Request.Context(), id); err != nil {
		BadRequest(c, err.Error())
		return
	}
	Success(c, gin.H{"deleted": true})
}

// GetCategoryTree GET /projects/:id/boms/:bomId/category-tree
func (h *BOMHandler) GetCategoryTree(c *gin.Context) {
	bomID := c.Param("bomId")
	tree, err := h.svc.GetCategoryTree(c.Request.Context(), bomID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, tree)
}

// SeedTemplates POST /api/v1/bom-templates/seed
func (h *BOMHandler) SeedTemplates(c *gin.Context) {
	h.svc.SeedDefaultTemplates(c.Request.Context())
	Success(c, gin.H{"seeded": true})
}

// ==================== Phase 6: 工艺路线 ====================

// CreateRoute POST /projects/:id/boms/:bomId/routes
func (h *BOMHandler) CreateRoute(c *gin.Context) {
	projectID := c.Param("id")
	bomID := c.Param("bomId")
	var input service.RouteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	input.BOMID = bomID
	userID := c.GetString("user_id")
	route, err := h.svc.CreateRoute(c.Request.Context(), projectID, &input, userID)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	Created(c, route)
}

// GetRoute GET /projects/:id/routes/:routeId
func (h *BOMHandler) GetRoute(c *gin.Context) {
	routeID := c.Param("routeId")
	route, err := h.svc.GetRoute(c.Request.Context(), routeID)
	if err != nil {
		NotFound(c, "Route not found")
		return
	}
	Success(c, route)
}

// ListRoutes GET /projects/:id/routes
func (h *BOMHandler) ListRoutes(c *gin.Context) {
	projectID := c.Param("id")
	routes, err := h.svc.ListRoutes(c.Request.Context(), projectID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, routes)
}

// UpdateRoute PUT /projects/:id/routes/:routeId
func (h *BOMHandler) UpdateRoute(c *gin.Context) {
	routeID := c.Param("routeId")
	var input service.RouteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	route, err := h.svc.UpdateRoute(c.Request.Context(), routeID, &input)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	Success(c, route)
}

// CreateStep POST /projects/:id/routes/:routeId/steps
func (h *BOMHandler) CreateStep(c *gin.Context) {
	routeID := c.Param("routeId")
	var input service.StepInput
	if err := c.ShouldBindJSON(&input); err != nil {
		BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	step, err := h.svc.CreateStep(c.Request.Context(), routeID, &input)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	Created(c, step)
}

// UpdateStep PUT /projects/:id/routes/:routeId/steps/:stepId
func (h *BOMHandler) UpdateStep(c *gin.Context) {
	stepID := c.Param("stepId")
	var input service.StepInput
	if err := c.ShouldBindJSON(&input); err != nil {
		BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	step, err := h.svc.UpdateStep(c.Request.Context(), stepID, &input)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	Success(c, step)
}

// DeleteStep DELETE /projects/:id/routes/:routeId/steps/:stepId
func (h *BOMHandler) DeleteStep(c *gin.Context) {
	stepID := c.Param("stepId")
	if err := h.svc.DeleteStep(c.Request.Context(), stepID); err != nil {
		BadRequest(c, err.Error())
		return
	}
	Success(c, gin.H{"deleted": true})
}

// CreateStepMaterial POST /projects/:id/routes/:routeId/steps/:stepId/materials
func (h *BOMHandler) CreateStepMaterial(c *gin.Context) {
	stepID := c.Param("stepId")
	var input service.StepMaterialInput
	if err := c.ShouldBindJSON(&input); err != nil {
		BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	m, err := h.svc.CreateStepMaterial(c.Request.Context(), stepID, &input)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	Created(c, m)
}

// DeleteStepMaterial DELETE /projects/:id/routes/:routeId/steps/:stepId/materials/:materialId
func (h *BOMHandler) DeleteStepMaterial(c *gin.Context) {
	materialID := c.Param("materialId")
	if err := h.svc.DeleteStepMaterial(c.Request.Context(), materialID); err != nil {
		BadRequest(c, err.Error())
		return
	}
	Success(c, gin.H{"deleted": true})
}

// ConvertToPBOM POST /projects/:id/boms/:bomId/convert-to-pbom
func (h *BOMHandler) ConvertToPBOM(c *gin.Context) {
	bomID := c.Param("bomId")
	userID := c.GetString("user_id")

	pbom, err := h.svc.ConvertToPBOM(c.Request.Context(), bomID, userID)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	Created(c, pbom)
}

// GetBOMPermissions GET /projects/:id/bom-permissions
func (h *BOMHandler) GetBOMPermissions(c *gin.Context) {
	projectID := c.Param("id")
	userID := GetUserID(c)

	perms, err := h.svc.GetBOMPermissions(c.Request.Context(), projectID, userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, perms)
}
