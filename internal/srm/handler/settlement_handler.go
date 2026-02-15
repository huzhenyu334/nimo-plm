package handler

import (
	"github.com/bitfantasy/nimo/internal/srm/service"
	"github.com/gin-gonic/gin"
)

// SettlementHandler 结算处理器
type SettlementHandler struct {
	svc *service.SettlementService
}

func NewSettlementHandler(svc *service.SettlementService) *SettlementHandler {
	return &SettlementHandler{svc: svc}
}

// ListSettlements 对账单列表
func (h *SettlementHandler) ListSettlements(c *gin.Context) {
	page, pageSize := GetPagination(c)
	filters := map[string]string{
		"supplier_id": c.Query("supplier_id"),
		"status":      c.Query("status"),
		"start_date":  c.Query("start_date"),
		"end_date":    c.Query("end_date"),
	}

	items, total, err := h.svc.List(c.Request.Context(), page, pageSize, filters)
	if err != nil {
		InternalError(c, "获取对账单列表失败: "+err.Error())
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

// ExportSettlements 导出对账单（预留）
func (h *SettlementHandler) ExportSettlements(c *gin.Context) {
	Success(c, nil)
}

// CreateSettlement 创建对账单
func (h *SettlementHandler) CreateSettlement(c *gin.Context) {
	var req service.CreateSettlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	userID := GetUserID(c)
	settlement, err := h.svc.Create(c.Request.Context(), userID, &req)
	if err != nil {
		InternalError(c, "创建对账单失败: "+err.Error())
		return
	}

	Created(c, settlement)
}

// GenerateSettlement 自动生成对账单
func (h *SettlementHandler) GenerateSettlement(c *gin.Context) {
	var req service.GenerateSettlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	userID := GetUserID(c)
	settlement, err := h.svc.Generate(c.Request.Context(), userID, &req)
	if err != nil {
		InternalError(c, "生成对账单失败: "+err.Error())
		return
	}

	Created(c, settlement)
}

// GetSettlement 对账单详情
func (h *SettlementHandler) GetSettlement(c *gin.Context) {
	id := c.Param("id")
	settlement, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "对账单不存在")
		return
	}
	Success(c, settlement)
}

// UpdateSettlement 更新对账单
func (h *SettlementHandler) UpdateSettlement(c *gin.Context) {
	id := c.Param("id")
	var req service.UpdateSettlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	settlement, err := h.svc.Update(c.Request.Context(), id, &req)
	if err != nil {
		InternalError(c, "更新对账单失败: "+err.Error())
		return
	}

	Success(c, settlement)
}

// DeleteSettlement 删除对账单
func (h *SettlementHandler) DeleteSettlement(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		InternalError(c, "删除对账单失败: "+err.Error())
		return
	}
	Success(c, nil)
}

// ConfirmByBuyer 采购方确认
func (h *SettlementHandler) ConfirmByBuyer(c *gin.Context) {
	id := c.Param("id")
	settlement, err := h.svc.ConfirmByBuyer(c.Request.Context(), id)
	if err != nil {
		InternalError(c, "确认失败: "+err.Error())
		return
	}
	Success(c, settlement)
}

// ConfirmBySupplier 供应商确认
func (h *SettlementHandler) ConfirmBySupplier(c *gin.Context) {
	id := c.Param("id")
	settlement, err := h.svc.ConfirmBySupplier(c.Request.Context(), id)
	if err != nil {
		InternalError(c, "确认失败: "+err.Error())
		return
	}
	Success(c, settlement)
}

// AddDispute 添加差异记录
func (h *SettlementHandler) AddDispute(c *gin.Context) {
	id := c.Param("id")
	var req service.CreateDisputeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	dispute, err := h.svc.AddDispute(c.Request.Context(), id, &req)
	if err != nil {
		InternalError(c, "添加差异记录失败: "+err.Error())
		return
	}

	Created(c, dispute)
}

// UpdateDispute 更新差异记录
func (h *SettlementHandler) UpdateDispute(c *gin.Context) {
	disputeID := c.Param("disputeId")
	var req service.UpdateDisputeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	dispute, err := h.svc.UpdateDispute(c.Request.Context(), disputeID, &req)
	if err != nil {
		InternalError(c, "更新差异记录失败: "+err.Error())
		return
	}

	Success(c, dispute)
}
