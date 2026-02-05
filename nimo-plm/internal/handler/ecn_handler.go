package handler

import (
	"strconv"

	"github.com/bitfantasy/nimo-plm/internal/service"
	"github.com/gin-gonic/gin"
)

// ECNHandler ECN处理器
type ECNHandler struct {
	svc *service.ECNService
}

// NewECNHandler 创建ECN处理器
func NewECNHandler(svc *service.ECNService) *ECNHandler {
	return &ECNHandler{svc: svc}
}

// List 获取ECN列表
func (h *ECNHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	filters := map[string]interface{}{
		"keyword":      c.Query("keyword"),
		"product_id":   c.Query("product_id"),
		"status":       c.Query("status"),
		"change_type":  c.Query("change_type"),
		"requested_by": c.Query("requested_by"),
		"urgency":      c.Query("urgency"),
	}

	result, err := h.svc.List(c.Request.Context(), page, pageSize, filters)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, result)
}

// Get 获取ECN详情
func (h *ECNHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequest(c, "ECN ID is required")
		return
	}

	ecn, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "ECN not found")
		return
	}

	Success(c, ecn)
}

// Create 创建ECN
func (h *ECNHandler) Create(c *gin.Context) {
	var req service.CreateECNRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	userID := GetUserID(c)
	ecn, err := h.svc.Create(c.Request.Context(), userID, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Created(c, ecn)
}

// Update 更新ECN
func (h *ECNHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequest(c, "ECN ID is required")
		return
	}

	var req service.UpdateECNRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	ecn, err := h.svc.Update(c.Request.Context(), id, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, ecn)
}

// Submit 提交审批
func (h *ECNHandler) Submit(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequest(c, "ECN ID is required")
		return
	}

	ecn, err := h.svc.Submit(c.Request.Context(), id)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, ecn)
}

// Approve 审批通过
func (h *ECNHandler) Approve(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequest(c, "ECN ID is required")
		return
	}

	var req struct {
		Comment string `json:"comment"`
	}
	c.ShouldBindJSON(&req)

	userID := GetUserID(c)
	ecn, err := h.svc.Approve(c.Request.Context(), id, userID, req.Comment)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, ecn)
}

// Reject 审批拒绝
func (h *ECNHandler) Reject(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequest(c, "ECN ID is required")
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Reason is required")
		return
	}

	userID := GetUserID(c)
	ecn, err := h.svc.Reject(c.Request.Context(), id, userID, req.Reason)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, ecn)
}

// Implement 实施ECN
func (h *ECNHandler) Implement(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequest(c, "ECN ID is required")
		return
	}

	userID := GetUserID(c)
	ecn, err := h.svc.Implement(c.Request.Context(), id, userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, ecn)
}

// ListAffectedItems 获取受影响项目列表
func (h *ECNHandler) ListAffectedItems(c *gin.Context) {
	ecnID := c.Param("id")
	if ecnID == "" {
		BadRequest(c, "ECN ID is required")
		return
	}

	items, err := h.svc.ListAffectedItems(c.Request.Context(), ecnID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, items)
}

// AddAffectedItem 添加受影响项目
func (h *ECNHandler) AddAffectedItem(c *gin.Context) {
	ecnID := c.Param("id")
	if ecnID == "" {
		BadRequest(c, "ECN ID is required")
		return
	}

	var req service.AffectedItemInput
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	item, err := h.svc.AddAffectedItem(c.Request.Context(), ecnID, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Created(c, item)
}

// RemoveAffectedItem 移除受影响项目
func (h *ECNHandler) RemoveAffectedItem(c *gin.Context) {
	ecnID := c.Param("id")
	itemID := c.Param("itemId")
	if ecnID == "" || itemID == "" {
		BadRequest(c, "ECN ID and Item ID are required")
		return
	}

	if err := h.svc.RemoveAffectedItem(c.Request.Context(), ecnID, itemID); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, nil)
}

// ListApprovals 获取审批记录
func (h *ECNHandler) ListApprovals(c *gin.Context) {
	ecnID := c.Param("id")
	if ecnID == "" {
		BadRequest(c, "ECN ID is required")
		return
	}

	approvals, err := h.svc.ListApprovals(c.Request.Context(), ecnID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, approvals)
}

// AddApprover 添加审批人
func (h *ECNHandler) AddApprover(c *gin.Context) {
	ecnID := c.Param("id")
	if ecnID == "" {
		BadRequest(c, "ECN ID is required")
		return
	}

	var req struct {
		ApproverID string `json:"approver_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	approval, err := h.svc.AddApprover(c.Request.Context(), ecnID, req.ApproverID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Created(c, approval)
}

// ListByProduct 获取产品的ECN列表
func (h *ECNHandler) ListByProduct(c *gin.Context) {
	productID := c.Param("productId")
	if productID == "" {
		BadRequest(c, "Product ID is required")
		return
	}

	ecns, err := h.svc.ListByProduct(c.Request.Context(), productID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, ecns)
}
