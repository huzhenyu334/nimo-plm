package handler

import (
	"github.com/bitfantasy/nimo/internal/plm/service"
	"github.com/gin-gonic/gin"
)

// ApprovalDefinitionHandler 审批定义处理器
type ApprovalDefinitionHandler struct {
	svc *service.ApprovalDefinitionService
}

// NewApprovalDefinitionHandler 创建审批定义处理器
func NewApprovalDefinitionHandler(svc *service.ApprovalDefinitionService) *ApprovalDefinitionHandler {
	return &ApprovalDefinitionHandler{svc: svc}
}

// ListDefinitions 获取审批定义列表（按分组）
// GET /api/v1/approval-definitions
func (h *ApprovalDefinitionHandler) ListDefinitions(c *gin.Context) {
	groups, err := h.svc.List(c.Request.Context())
	if err != nil {
		InternalError(c, "获取审批定义列表失败: "+err.Error())
		return
	}
	Success(c, gin.H{"groups": groups})
}

// CreateDefinition 创建审批定义
// POST /api/v1/approval-definitions
func (h *ApprovalDefinitionHandler) CreateDefinition(c *gin.Context) {
	var req service.CreateDefinitionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	userID := GetUserID(c)
	if userID == "" {
		Unauthorized(c, "未登录")
		return
	}

	def, err := h.svc.Create(c.Request.Context(), req, userID)
	if err != nil {
		InternalError(c, "创建审批定义失败: "+err.Error())
		return
	}

	Created(c, def)
}

// GetDefinition 获取审批定义详情
// GET /api/v1/approval-definitions/:id
func (h *ApprovalDefinitionHandler) GetDefinition(c *gin.Context) {
	id := c.Param("id")
	def, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "审批定义不存在")
		return
	}
	Success(c, def)
}

// UpdateDefinition 更新审批定义
// PUT /api/v1/approval-definitions/:id
func (h *ApprovalDefinitionHandler) UpdateDefinition(c *gin.Context) {
	id := c.Param("id")
	var req service.UpdateDefinitionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	def, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		InternalError(c, "更新审批定义失败: "+err.Error())
		return
	}

	Success(c, def)
}

// DeleteDefinition 删除审批定义
// DELETE /api/v1/approval-definitions/:id
func (h *ApprovalDefinitionHandler) DeleteDefinition(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, gin.H{"message": "删除成功"})
}

// PublishDefinition 发布审批定义
// POST /api/v1/approval-definitions/:id/publish
func (h *ApprovalDefinitionHandler) PublishDefinition(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Publish(c.Request.Context(), id); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, gin.H{"message": "发布成功"})
}

// UnpublishDefinition 取消发布审批定义
// POST /api/v1/approval-definitions/:id/unpublish
func (h *ApprovalDefinitionHandler) UnpublishDefinition(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Unpublish(c.Request.Context(), id); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, gin.H{"message": "取消发布成功"})
}

// ListGroups 获取审批分组列表
// GET /api/v1/approval-groups
func (h *ApprovalDefinitionHandler) ListGroups(c *gin.Context) {
	groups, err := h.svc.ListGroups(c.Request.Context())
	if err != nil {
		InternalError(c, "获取分组列表失败: "+err.Error())
		return
	}
	Success(c, gin.H{"items": groups})
}

// CreateGroupReq 创建分组请求
type CreateGroupReq struct {
	Name string `json:"name" binding:"required"`
}

// CreateGroup 创建审批分组
// POST /api/v1/approval-groups
func (h *ApprovalDefinitionHandler) CreateGroup(c *gin.Context) {
	var req CreateGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	group, err := h.svc.CreateGroup(c.Request.Context(), req.Name)
	if err != nil {
		InternalError(c, "创建分组失败: "+err.Error())
		return
	}

	Created(c, group)
}

// DeleteGroup 删除审批分组
// DELETE /api/v1/approval-groups/:id
func (h *ApprovalDefinitionHandler) DeleteGroup(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.DeleteGroup(c.Request.Context(), id); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, gin.H{"message": "删除成功"})
}

// SubmitInstance 从定义发起审批实例
// POST /api/v1/approval-definitions/:id/submit
func (h *ApprovalDefinitionHandler) SubmitInstance(c *gin.Context) {
	definitionID := c.Param("id")

	var req service.CreateInstanceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	userID := GetUserID(c)
	if userID == "" {
		Unauthorized(c, "未登录")
		return
	}

	approval, err := h.svc.CreateInstance(c.Request.Context(), definitionID, req, userID)
	if err != nil {
		InternalError(c, "发起审批失败: "+err.Error())
		return
	}

	Created(c, approval)
}
