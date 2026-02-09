package handler

import (
	"github.com/bitfantasy/nimo/internal/srm/service"
	"github.com/gin-gonic/gin"
)

// InspectionHandler 检验处理器
type InspectionHandler struct {
	svc *service.InspectionService
}

func NewInspectionHandler(svc *service.InspectionService) *InspectionHandler {
	return &InspectionHandler{svc: svc}
}

// ListInspections 检验列表
// GET /api/v1/srm/inspections?supplier_id=xxx&status=xxx&result=xxx&po_id=xxx
func (h *InspectionHandler) ListInspections(c *gin.Context) {
	page, pageSize := GetPagination(c)
	filters := map[string]string{
		"supplier_id": c.Query("supplier_id"),
		"status":      c.Query("status"),
		"result":      c.Query("result"),
		"po_id":       c.Query("po_id"),
	}

	items, total, err := h.svc.ListInspections(c.Request.Context(), page, pageSize, filters)
	if err != nil {
		InternalError(c, "获取检验列表失败: "+err.Error())
		return
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	Success(c, ListResponse{
		Items: items,
		Pagination: &Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      int(total),
			TotalPages: totalPages,
		},
	})
}

// GetInspection 检验详情
// GET /api/v1/srm/inspections/:id
func (h *InspectionHandler) GetInspection(c *gin.Context) {
	id := c.Param("id")
	inspection, err := h.svc.GetInspection(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "检验记录不存在")
		return
	}
	Success(c, inspection)
}

// UpdateInspection 更新检验
// PUT /api/v1/srm/inspections/:id
func (h *InspectionHandler) UpdateInspection(c *gin.Context) {
	id := c.Param("id")
	var req service.UpdateInspectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	inspection, err := h.svc.UpdateInspection(c.Request.Context(), id, &req)
	if err != nil {
		InternalError(c, "更新检验失败: "+err.Error())
		return
	}

	Success(c, inspection)
}

// CompleteInspection 完成检验
// POST /api/v1/srm/inspections/:id/complete
func (h *InspectionHandler) CompleteInspection(c *gin.Context) {
	id := c.Param("id")
	userID := GetUserID(c)

	var req service.CompleteInspectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	inspection, err := h.svc.CompleteInspection(c.Request.Context(), id, userID, &req)
	if err != nil {
		InternalError(c, "完成检验失败: "+err.Error())
		return
	}

	Success(c, inspection)
}
