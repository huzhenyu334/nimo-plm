package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/bitfantasy/nimo/internal/srm/service"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
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
		BadRequest(c, "更新采购订单失败: "+err.Error())
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
		if strings.Contains(err.Error(), "不可重复审批") {
			BadRequest(c, err.Error())
			return
		}
		BadRequest(c, "审批失败: "+err.Error())
		return
	}

	Success(c, po)
}

// ExportPOs 导出采购订单Excel
// GET /api/v1/srm/purchase-orders/export?supplier_id=xxx&status=xxx&type=xxx
func (h *POHandler) ExportPOs(c *gin.Context) {
	filters := map[string]string{
		"supplier_id": c.Query("supplier_id"),
		"status":      c.Query("status"),
		"type":        c.Query("type"),
		"search":      c.Query("search"),
	}

	// 获取全部数据（不分页）
	items, _, err := h.svc.ListPOs(c.Request.Context(), 1, 10000, filters)
	if err != nil {
		InternalError(c, "获取采购订单数据失败: "+err.Error())
		return
	}

	f := excelize.NewFile()
	defer f.Close()
	sheet := "采购订单"
	f.SetSheetName("Sheet1", sheet)

	// 表头
	headers := []string{"PO编码", "供应商", "类型", "状态", "总金额", "创建时间"}
	for i, hdr := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, hdr)
	}

	// 表头样式
	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
	})
	f.SetRowStyle(sheet, 1, 1, style)

	typeMap := map[string]string{
		"sample": "打样", "production": "量产",
	}
	statusMap := map[string]string{
		"draft": "草稿", "approved": "已审批", "sent": "已发送",
		"partial": "部分收货", "received": "已收货", "completed": "已完成", "cancelled": "已取消",
	}

	for i, po := range items {
		row := i + 2

		f.SetCellValue(sheet, cellName(1, row), po.POCode)

		supplierName := ""
		if po.Supplier != nil {
			supplierName = po.Supplier.Name
		}
		f.SetCellValue(sheet, cellName(2, row), supplierName)

		typeText := po.Type
		if t, ok := typeMap[po.Type]; ok {
			typeText = t
		}
		f.SetCellValue(sheet, cellName(3, row), typeText)

		statusText := po.Status
		if t, ok := statusMap[po.Status]; ok {
			statusText = t
		}
		f.SetCellValue(sheet, cellName(4, row), statusText)

		if po.TotalAmount != nil {
			f.SetCellValue(sheet, cellName(5, row), *po.TotalAmount)
		}

		f.SetCellValue(sheet, cellName(6, row), po.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	// 设置列宽
	colWidths := []float64{18, 20, 10, 12, 14, 20}
	for i, w := range colWidths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, w)
	}

	filename := "purchase_orders"
	if sid := c.Query("supplier_id"); sid != "" {
		filename = fmt.Sprintf("purchase_orders_%s", sid)
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.xlsx"`, filename))

	if err := f.Write(c.Writer); err != nil {
		InternalError(c, "生成Excel失败: "+err.Error())
		return
	}
}

// SubmitPO 提交采购订单审批
// POST /api/v1/srm/purchase-orders/:id/submit
func (h *POHandler) SubmitPO(c *gin.Context) {
	id := c.Param("id")
	po, err := h.svc.SubmitPO(c.Request.Context(), id)
	if err != nil {
		BadRequest(c, "提交失败: "+err.Error())
		return
	}
	Success(c, po)
}

// DeletePO 删除采购订单
// DELETE /api/v1/srm/purchase-orders/:id
func (h *POHandler) DeletePO(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.DeletePO(c.Request.Context(), id); err != nil {
		BadRequest(c, "删除失败: "+err.Error())
		return
	}
	Success(c, nil)
}

// GenerateFromBOM 从BOM生成采购订单
// POST /api/v1/srm/purchase-orders/from-bom
func (h *POHandler) GenerateFromBOM(c *gin.Context) {
	var req service.GenerateFromBOMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}
	userID := GetUserID(c)
	pos, err := h.svc.GeneratePOsFromBOM(c.Request.Context(), userID, &req)
	if err != nil {
		InternalError(c, "从BOM生成采购订单失败: "+err.Error())
		return
	}
	Created(c, pos)
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
