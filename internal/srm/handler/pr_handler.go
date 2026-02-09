package handler

import (
	"github.com/bitfantasy/nimo/internal/srm/service"
	"github.com/gin-gonic/gin"
)

// PRHandler 采购需求处理器
type PRHandler struct {
	svc         *service.ProcurementService
	bomProvider service.BOMProvider
}

func NewPRHandler(svc *service.ProcurementService) *PRHandler {
	return &PRHandler{svc: svc}
}

// SetBOMProvider 设置BOM数据提供者（PLM集成用）
func (h *PRHandler) SetBOMProvider(provider service.BOMProvider) {
	h.bomProvider = provider
}

// ListPRs 采购需求列表
// GET /api/v1/srm/purchase-requests?project_id=xxx&status=xxx&type=xxx&search=xxx
func (h *PRHandler) ListPRs(c *gin.Context) {
	page, pageSize := GetPagination(c)
	filters := map[string]string{
		"project_id": c.Query("project_id"),
		"status":     c.Query("status"),
		"type":       c.Query("type"),
		"search":     c.Query("search"),
	}

	items, total, err := h.svc.ListPRs(c.Request.Context(), page, pageSize, filters)
	if err != nil {
		InternalError(c, "获取采购需求列表失败: "+err.Error())
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

// GetPR 采购需求详情
// GET /api/v1/srm/purchase-requests/:id
func (h *PRHandler) GetPR(c *gin.Context) {
	id := c.Param("id")
	pr, err := h.svc.GetPR(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "采购需求不存在")
		return
	}
	Success(c, pr)
}

// CreatePR 创建采购需求
// POST /api/v1/srm/purchase-requests
func (h *PRHandler) CreatePR(c *gin.Context) {
	var req service.CreatePRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	userID := GetUserID(c)
	pr, err := h.svc.CreatePR(c.Request.Context(), userID, &req)
	if err != nil {
		InternalError(c, "创建采购需求失败: "+err.Error())
		return
	}

	Created(c, pr)
}

// UpdatePR 更新采购需求
// PUT /api/v1/srm/purchase-requests/:id
func (h *PRHandler) UpdatePR(c *gin.Context) {
	id := c.Param("id")
	var req service.UpdatePRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	pr, err := h.svc.UpdatePR(c.Request.Context(), id, &req)
	if err != nil {
		InternalError(c, "更新采购需求失败: "+err.Error())
		return
	}

	Success(c, pr)
}

// CreatePRFromBOM 从BOM创建采购需求
// POST /api/v1/srm/purchase-requests/from-bom
func (h *PRHandler) CreatePRFromBOM(c *gin.Context) {
	var req struct {
		ProjectID string `json:"project_id" binding:"required"`
		BOMID     string `json:"bom_id" binding:"required"`
		Phase     string `json:"phase"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	userID := GetUserID(c)

	if h.bomProvider == nil {
		InternalError(c, "BOM数据源未配置")
		return
	}

	_, phase, bomItems, err := h.bomProvider.GetBOMWithItems(c.Request.Context(), req.BOMID)
	if err != nil {
		InternalError(c, "获取BOM数据失败: "+err.Error())
		return
	}

	if req.Phase == "" {
		req.Phase = phase
	}

	pr, err := h.svc.CreatePRFromBOM(c.Request.Context(), req.ProjectID, req.BOMID, userID, bomItems, req.Phase)
	if err != nil {
		InternalError(c, "从BOM创建PR失败: "+err.Error())
		return
	}

	Created(c, pr)
}

// ApprovePR 审批采购需求
// POST /api/v1/srm/purchase-requests/:id/approve
func (h *PRHandler) ApprovePR(c *gin.Context) {
	id := c.Param("id")
	userID := GetUserID(c)

	pr, err := h.svc.ApprovePR(c.Request.Context(), id, userID)
	if err != nil {
		InternalError(c, "审批失败: "+err.Error())
		return
	}

	Success(c, pr)
}
