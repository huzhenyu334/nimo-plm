package handler

import (
	"context"

	"github.com/bitfantasy/nimo/internal/srm/service"
	"github.com/gin-gonic/gin"
)

// POItemReceiver PO行项收货接口
type POItemReceiver interface {
	ReceiveItem(ctx context.Context, itemID string, receivedQty float64) error
}

// POHandler 采购订单处理器
type POHandler struct {
	svc    *service.ProcurementService
	poRepo POItemReceiver
}

func NewPOHandler(svc *service.ProcurementService, poRepo POItemReceiver) *POHandler {
	return &POHandler{svc: svc, poRepo: poRepo}
}

// ListPOs 采购订单列表
// GET /api/v1/srm/purchase-orders?supplier_id=xxx&status=xxx&type=xxx&search=xxx
func (h *POHandler) ListPOs(c *gin.Context) {
	page, pageSize := GetPagination(c)
	filters := map[string]string{
		"supplier_id": c.Query("supplier_id"),
		"status":      c.Query("status"),
		"type":        c.Query("type"),
		"search":      c.Query("search"),
	}

	items, total, err := h.svc.ListPOs(c.Request.Context(), page, pageSize, filters)
	if err != nil {
		InternalError(c, "获取采购订单列表失败: "+err.Error())
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

// GetPO 采购订单详情
// GET /api/v1/srm/purchase-orders/:id
func (h *POHandler) GetPO(c *gin.Context) {
	id := c.Param("id")
	po, err := h.svc.GetPO(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "采购订单不存在")
		return
	}
	Success(c, po)
}

// CreatePO 创建采购订单
// POST /api/v1/srm/purchase-orders
func (h *POHandler) CreatePO(c *gin.Context) {
	var req service.CreatePORequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	userID := GetUserID(c)
	po, err := h.svc.CreatePO(c.Request.Context(), userID, &req)
	if err != nil {
		InternalError(c, "创建采购订单失败: "+err.Error())
		return
	}

	Created(c, po)
}

// UpdatePO 更新采购订单
// PUT /api/v1/srm/purchase-orders/:id
func (h *POHandler) UpdatePO(c *gin.Context) {
	id := c.Param("id")
	var req service.UpdatePORequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	po, err := h.svc.UpdatePO(c.Request.Context(), id, &req)
	if err != nil {
		InternalError(c, "更新采购订单失败: "+err.Error())
		return
	}

	Success(c, po)
}

// ApprovePO 审批采购订单
// POST /api/v1/srm/purchase-orders/:id/approve
func (h *POHandler) ApprovePO(c *gin.Context) {
	id := c.Param("id")
	userID := GetUserID(c)

	po, err := h.svc.ApprovePO(c.Request.Context(), id, userID)
	if err != nil {
		InternalError(c, "审批失败: "+err.Error())
		return
	}

	Success(c, po)
}

// ReceiveItem PO行项收货
// POST /api/v1/srm/purchase-orders/:id/items/:itemId/receive
func (h *POHandler) ReceiveItem(c *gin.Context) {
	itemID := c.Param("itemId")

	var req service.ReceiveItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if req.ReceivedQty <= 0 {
		BadRequest(c, "收货数量必须大于0")
		return
	}

	if err := h.poRepo.ReceiveItem(c.Request.Context(), itemID, req.ReceivedQty); err != nil {
		InternalError(c, "收货失败: "+err.Error())
		return
	}

	Success(c, nil)
}
