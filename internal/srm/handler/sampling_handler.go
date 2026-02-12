package handler

import (
	"github.com/bitfantasy/nimo/internal/srm/service"
	"github.com/gin-gonic/gin"
)

// SamplingHandler 打样处理器
type SamplingHandler struct {
	svc *service.SamplingService
}

func NewSamplingHandler(svc *service.SamplingService) *SamplingHandler {
	return &SamplingHandler{svc: svc}
}

// CreateSamplingRequest 发起打样
// POST /srm/pr-items/:itemId/sampling
func (h *SamplingHandler) CreateSamplingRequest(c *gin.Context) {
	itemID := c.Param("itemId")
	if itemID == "" {
		BadRequest(c, "缺少物料ID")
		return
	}

	var req service.CreateSamplingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	userID := GetUserID(c)
	sampling, err := h.svc.CreateSamplingRequest(c.Request.Context(), itemID, req, userID)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	Created(c, sampling)
}

// ListSamplingRequests 获取物料的打样记录列表
// GET /srm/pr-items/:itemId/sampling
func (h *SamplingHandler) ListSamplingRequests(c *gin.Context) {
	itemID := c.Param("itemId")
	if itemID == "" {
		BadRequest(c, "缺少物料ID")
		return
	}

	items, err := h.svc.ListSamplingRequests(c.Request.Context(), itemID)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, gin.H{"items": items})
}

// UpdateSamplingStatus 更新打样状态
// PUT /srm/sampling/:id/status
func (h *SamplingHandler) UpdateSamplingStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequest(c, "缺少打样ID")
		return
	}

	var req service.UpdateSamplingStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	userID := GetUserID(c)
	sampling, err := h.svc.UpdateSamplingStatus(c.Request.Context(), id, req, userID)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, sampling)
}

// RequestVerify 发起研发验证（发飞书审批）
// POST /srm/sampling/:id/request-verify
func (h *SamplingHandler) RequestVerify(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequest(c, "缺少打样ID")
		return
	}

	var req service.RequestVerifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	userID := GetUserID(c)
	sampling, err := h.svc.RequestVerify(c.Request.Context(), id, req, userID)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, sampling)
}

// VerifyCallback 飞书审批回调
// POST /srm/sampling/verify-callback
func (h *SamplingHandler) VerifyCallback(c *gin.Context) {
	var req struct {
		InstanceCode string `json:"instance_code"`
		Status       string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请求参数错误")
		return
	}

	if err := h.svc.HandleVerifyCallback(c.Request.Context(), req.InstanceCode, req.Status); err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, nil)
}
