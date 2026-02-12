package handler

import (
	"github.com/bitfantasy/nimo/internal/plm/entity"
	"github.com/bitfantasy/nimo/internal/plm/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PartDrawingHandler struct {
	repo    *repository.PartDrawingRepository
	bomRepo *repository.ProjectBOMRepository
}

func NewPartDrawingHandler(repo *repository.PartDrawingRepository, bomRepo *repository.ProjectBOMRepository) *PartDrawingHandler {
	return &PartDrawingHandler{repo: repo, bomRepo: bomRepo}
}

// ListDrawings GET /projects/:id/bom-items/:itemId/drawings
func (h *PartDrawingHandler) ListDrawings(c *gin.Context) {
	itemID := c.Param("itemId")
	drawings, err := h.repo.ListByBOMItem(c.Request.Context(), itemID)
	if err != nil {
		InternalError(c, "获取图纸列表失败: "+err.Error())
		return
	}

	// 按类型分组
	result := map[string][]entity.PartDrawing{
		"2D": {},
		"3D": {},
	}
	for _, d := range drawings {
		result[d.DrawingType] = append(result[d.DrawingType], d)
	}
	Success(c, result)
}

// UploadDrawing POST /projects/:id/bom-items/:itemId/drawings
func (h *PartDrawingHandler) UploadDrawing(c *gin.Context) {
	itemID := c.Param("itemId")
	userID := GetUserID(c)

	var req struct {
		DrawingType       string `json:"drawing_type" binding:"required"`
		FileID            string `json:"file_id" binding:"required"`
		FileName          string `json:"file_name" binding:"required"`
		FileSize          int64  `json:"file_size"`
		ChangeDescription string `json:"change_description"`
		ChangeReason      string `json:"change_reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if req.DrawingType != "2D" && req.DrawingType != "3D" {
		BadRequest(c, "drawing_type 必须为 2D 或 3D")
		return
	}

	// 自动计算版本号
	version, err := h.repo.GetNextVersion(c.Request.Context(), itemID, req.DrawingType)
	if err != nil {
		InternalError(c, "获取版本号失败: "+err.Error())
		return
	}

	drawing := &entity.PartDrawing{
		ID:                uuid.New().String()[:32],
		BOMItemID:         itemID,
		DrawingType:       req.DrawingType,
		Version:           version,
		FileID:            req.FileID,
		FileName:          req.FileName,
		FileSize:          req.FileSize,
		FileURL:           "/uploads/" + req.FileID + "/" + req.FileName,
		ChangeDescription: req.ChangeDescription,
		ChangeReason:      req.ChangeReason,
		UploadedBy:        userID,
	}

	if err := h.repo.Create(c.Request.Context(), drawing); err != nil {
		InternalError(c, "创建图纸记录失败: "+err.Error())
		return
	}

	Created(c, drawing)
}

// DeleteDrawing DELETE /projects/:id/bom-items/:itemId/drawings/:drawingId
func (h *PartDrawingHandler) DeleteDrawing(c *gin.Context) {
	drawingID := c.Param("drawingId")

	_, err := h.repo.FindByID(c.Request.Context(), drawingID)
	if err != nil {
		NotFound(c, "图纸不存在")
		return
	}

	if err := h.repo.Delete(c.Request.Context(), drawingID); err != nil {
		InternalError(c, "删除图纸失败: "+err.Error())
		return
	}

	Success(c, gin.H{"message": "删除成功"})
}

// DownloadDrawing GET /projects/:id/bom-items/:itemId/drawings/:drawingId/download
func (h *PartDrawingHandler) DownloadDrawing(c *gin.Context) {
	drawingID := c.Param("drawingId")

	drawing, err := h.repo.FindByID(c.Request.Context(), drawingID)
	if err != nil {
		NotFound(c, "图纸不存在")
		return
	}

	c.Redirect(302, drawing.FileURL)
}

// ListDrawingsByBOM GET /projects/:id/boms/:bomId/drawings
func (h *PartDrawingHandler) ListDrawingsByBOM(c *gin.Context) {
	bomID := c.Param("bomId")

	// 获取BOM的所有item IDs
	bom, err := h.bomRepo.FindByID(c.Request.Context(), bomID)
	if err != nil {
		NotFound(c, "BOM不存在")
		return
	}

	itemIDs := make([]string, len(bom.Items))
	for i, item := range bom.Items {
		itemIDs[i] = item.ID
	}

	drawings, err := h.repo.ListByBOMItems(c.Request.Context(), itemIDs)
	if err != nil {
		InternalError(c, "获取图纸列表失败: "+err.Error())
		return
	}

	// 按 BOMItemID 分组
	grouped := make(map[string]map[string][]entity.PartDrawing)
	for _, d := range drawings {
		if grouped[d.BOMItemID] == nil {
			grouped[d.BOMItemID] = map[string][]entity.PartDrawing{"2D": {}, "3D": {}}
		}
		grouped[d.BOMItemID][d.DrawingType] = append(grouped[d.BOMItemID][d.DrawingType], d)
	}

	Success(c, grouped)
}
