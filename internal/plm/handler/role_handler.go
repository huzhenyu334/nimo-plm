package handler

import (
	"time"

	"github.com/bitfantasy/nimo/internal/plm/entity"
	"github.com/bitfantasy/nimo/internal/shared/feishu"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RoleHandler 角色管理处理器
type RoleHandler struct {
	db           *gorm.DB
	feishuClient *feishu.FeishuClient
}

// NewRoleHandler 创建角色处理器
func NewRoleHandler(db *gorm.DB, feishuClient *feishu.FeishuClient) *RoleHandler {
	return &RoleHandler{db: db, feishuClient: feishuClient}
}

// List 获取角色列表
// GET /api/v1/roles
func (h *RoleHandler) List(c *gin.Context) {
	var roles []entity.Role
	if err := h.db.Order("created_at ASC").Find(&roles).Error; err != nil {
		InternalError(c, "获取角色列表失败: "+err.Error())
		return
	}
	Success(c, gin.H{"items": roles})
}

// Get 获取角色详情
// GET /api/v1/roles/:id
func (h *RoleHandler) Get(c *gin.Context) {
	id := c.Param("id")
	var role entity.Role
	if err := h.db.Where("id = ?", id).First(&role).Error; err != nil {
		NotFound(c, "角色不存在")
		return
	}
	Success(c, role)
}

// CreateRoleRequest 创建角色请求
type CreateRoleRequest struct {
	Code        string `json:"code" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// Create 创建角色
// POST /api/v1/roles
func (h *RoleHandler) Create(c *gin.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 检查 code 唯一性
	var count int64
	h.db.Model(&entity.Role{}).Where("code = ?", req.Code).Count(&count)
	if count > 0 {
		BadRequest(c, "角色编码已存在")
		return
	}

	now := time.Now()
	role := &entity.Role{
		ID:          uuid.New().String()[:32],
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.db.Create(role).Error; err != nil {
		InternalError(c, "创建角色失败: "+err.Error())
		return
	}

	Created(c, role)
}

// UpdateRoleRequest 更新角色请求
type UpdateRoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Update 更新角色
// PUT /api/v1/roles/:id
func (h *RoleHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var role entity.Role
	if err := h.db.Where("id = ?", id).First(&role).Error; err != nil {
		NotFound(c, "角色不存在")
		return
	}

	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if req.Name != "" {
		role.Name = req.Name
	}
	if req.Description != "" {
		role.Description = req.Description
	}
	role.UpdatedAt = time.Now()

	if err := h.db.Save(&role).Error; err != nil {
		InternalError(c, "更新角色失败: "+err.Error())
		return
	}

	Success(c, role)
}

// Delete 删除角色
// DELETE /api/v1/roles/:id
func (h *RoleHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	var role entity.Role
	if err := h.db.Where("id = ?", id).First(&role).Error; err != nil {
		NotFound(c, "角色不存在")
		return
	}

	if role.IsSystem {
		BadRequest(c, "系统角色不可删除")
		return
	}

	if err := h.db.Delete(&role).Error; err != nil {
		InternalError(c, "删除角色失败: "+err.Error())
		return
	}

	Success(c, gin.H{"message": "角色已删除"})
}

// ListTaskRoles 获取任务角色列表
// GET /api/v1/task-roles
func (h *RoleHandler) ListTaskRoles(c *gin.Context) {
	var taskRoles []entity.TaskRole
	if err := h.db.Order("sort_order ASC, created_at ASC").Find(&taskRoles).Error; err != nil {
		InternalError(c, "获取任务角色列表失败: "+err.Error())
		return
	}
	Success(c, gin.H{"items": taskRoles})
}

// ListFeishuRoles 获取飞书部门列表作为角色
// GET /api/v1/feishu/roles
func (h *RoleHandler) ListFeishuRoles(c *gin.Context) {
	if h.feishuClient == nil {
		Success(c, gin.H{"items": []interface{}{}})
		return
	}

	depts, err := h.feishuClient.ListDepartments(c.Request.Context())
	if err != nil {
		InternalError(c, "获取飞书部门列表失败: "+err.Error())
		return
	}

	type FeishuRole struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}

	items := make([]FeishuRole, 0, len(depts))
	for _, d := range depts {
		code := d.DepartmentID
		if code == "" {
			code = d.OpenDepartmentID
		}
		items = append(items, FeishuRole{
			Code: code,
			Name: d.Name,
		})
	}

	Success(c, gin.H{"items": items})
}
