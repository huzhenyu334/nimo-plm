package handler

import (
	"github.com/bitfantasy/nimo/internal/srm/service"
	"github.com/gin-gonic/gin"
)

// EvaluationHandler 评估处理器
type EvaluationHandler struct {
	svc *service.EvaluationService
}

func NewEvaluationHandler(svc *service.EvaluationService) *EvaluationHandler {
	return &EvaluationHandler{svc: svc}
}

// ListEvaluations 评估列表
func (h *EvaluationHandler) ListEvaluations(c *gin.Context) {
	page, pageSize := GetPagination(c)
	filters := map[string]string{
		"supplier_id": c.Query("supplier_id"),
		"status":      c.Query("status"),
		"eval_type":   c.Query("eval_type"),
		"period":      c.Query("period"),
	}

	items, total, err := h.svc.List(c.Request.Context(), page, pageSize, filters)
	if err != nil {
		InternalError(c, "获取评估列表失败: "+err.Error())
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

// CreateEvaluation 创建评估
func (h *EvaluationHandler) CreateEvaluation(c *gin.Context) {
	var req service.CreateEvaluationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	userID := GetUserID(c)
	eval, err := h.svc.Create(c.Request.Context(), userID, &req)
	if err != nil {
		InternalError(c, "创建评估失败: "+err.Error())
		return
	}

	Created(c, eval)
}

// AutoGenerate 自动生成评估
func (h *EvaluationHandler) AutoGenerate(c *gin.Context) {
	var req service.AutoGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	userID := GetUserID(c)
	eval, err := h.svc.AutoGenerate(c.Request.Context(), userID, &req)
	if err != nil {
		InternalError(c, "自动生成评估失败: "+err.Error())
		return
	}

	Created(c, eval)
}

// GetSupplierHistory 供应商评估历史
func (h *EvaluationHandler) GetSupplierHistory(c *gin.Context) {
	supplierID := c.Param("supplierId")
	items, err := h.svc.GetSupplierHistory(c.Request.Context(), supplierID)
	if err != nil {
		InternalError(c, "获取评估历史失败: "+err.Error())
		return
	}
	Success(c, items)
}

// GetEvaluation 评估详情
func (h *EvaluationHandler) GetEvaluation(c *gin.Context) {
	id := c.Param("id")
	eval, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "评估不存在")
		return
	}
	Success(c, eval)
}

// UpdateEvaluation 更新评估
func (h *EvaluationHandler) UpdateEvaluation(c *gin.Context) {
	id := c.Param("id")
	var req service.UpdateEvaluationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	eval, err := h.svc.Update(c.Request.Context(), id, &req)
	if err != nil {
		InternalError(c, "更新评估失败: "+err.Error())
		return
	}

	Success(c, eval)
}

// Submit 提交评估
func (h *EvaluationHandler) Submit(c *gin.Context) {
	id := c.Param("id")
	eval, err := h.svc.Submit(c.Request.Context(), id)
	if err != nil {
		InternalError(c, "提交评估失败: "+err.Error())
		return
	}
	Success(c, eval)
}

// Approve 审批评估
func (h *EvaluationHandler) Approve(c *gin.Context) {
	id := c.Param("id")
	eval, err := h.svc.Approve(c.Request.Context(), id)
	if err != nil {
		InternalError(c, "审批评估失败: "+err.Error())
		return
	}
	Success(c, eval)
}
