package handler

import (
	"strconv"

	"github.com/bitfantasy/nimo/internal/config"
	"github.com/bitfantasy/nimo/internal/plm/repository"
	"github.com/bitfantasy/nimo/internal/plm/service"
	"github.com/gin-gonic/gin"
)

// Handlers 处理器集合
type Handlers struct {
	Auth        *AuthHandler
	User        *UserHandler
	Product     *ProductHandler
	Material    *MaterialHandler
	BOM         *OldBOMHandler
	Project     *ProjectHandler
	Task        *ProjectHandler  // Task methods are on ProjectHandler
	ECN         *ECNHandler
	Document    *DocumentHandler
	Template    *TemplateHandler
	// V2 新增
	ProjectBOM  *BOMHandler
	Deliverable *DeliverableHandler
	Codename    *CodenameHandler
	// V3 工作流
	Workflow    *WorkflowHandler
	// V4 审批 + 管理
	Admin       *AdminHandler
	Approval    *ApprovalHandler
	// V5 审批定义
	ApprovalDef *ApprovalDefinitionHandler
	// V6 任务表单 + 文件上传
	TaskForm    *TaskFormHandler
	Upload      *UploadHandler
	// V7 SSE
	SSE         *SSEHandler
	// V8 角色
	Role        *RoleHandler
}

// NewHandlers 创建处理器集合
func NewHandlers(svc *service.Services, repos *repository.Repositories, cfg *config.Config, workflowSvc *service.WorkflowService) *Handlers {
	projectHandler := NewProjectHandler(svc.Project)
	h := &Handlers{
		Auth:        NewAuthHandler(svc.Auth, cfg),
		User:        NewUserHandler(svc.User),
		Product:     NewProductHandler(svc.Product),
		Material:    NewMaterialHandler(svc.Material),
		BOM:         &OldBOMHandler{},
		Project:     projectHandler,
		Task:        projectHandler,  // Reuse ProjectHandler for task routes
		ECN:         NewECNHandler(svc.ECN),
		Document:    NewDocumentHandler(svc.Document),
		Template:    NewTemplateHandler(svc.Template),
		// V2 新增
		ProjectBOM:  NewBOMHandler(svc.ProjectBOM),
		Deliverable: NewDeliverableHandler(repos.Deliverable),
		Codename:    NewCodenameHandler(repos.Codename),
		// V6 任务表单 + 文件上传
		TaskForm:    NewTaskFormHandler(repos.TaskForm, repos.Project),
		Upload:      NewUploadHandler(),
		// V7 SSE
		SSE:         NewSSEHandler(),
	}
	// V3 工作流
	if workflowSvc != nil {
		h.Workflow = NewWorkflowHandler(workflowSvc)
	}
	return h
}

// Response 通用响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ListResponse 列表响应结构
type ListResponse struct {
	Items      interface{} `json:"items"`
	Pagination *Pagination `json:"pagination"`
}

// Pagination 分页信息
type Pagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(200, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// Created 创建成功响应
func Created(c *gin.Context, data interface{}) {
	c.JSON(201, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// Error 错误响应
func Error(c *gin.Context, code int, message string) {
	statusCode := code / 100
	if statusCode < 100 || statusCode > 599 {
		statusCode = 500
	}
	c.JSON(statusCode, Response{
		Code:    code,
		Message: message,
	})
}

// BadRequest 参数错误响应
func BadRequest(c *gin.Context, message string) {
	Error(c, 40000, message)
}

// Unauthorized 未授权响应
func Unauthorized(c *gin.Context, message string) {
	Error(c, 40100, message)
}

// Forbidden 禁止访问响应
func Forbidden(c *gin.Context, message string) {
	Error(c, 40300, message)
}

// NotFound 资源不存在响应
func NotFound(c *gin.Context, message string) {
	Error(c, 40400, message)
}

// InternalError 服务器错误响应
func InternalError(c *gin.Context, message string) {
	Error(c, 50000, message)
}

// GetUserID 从上下文获取用户ID
func GetUserID(c *gin.Context) string {
	userID, _ := c.Get("user_id")
	if id, ok := userID.(string); ok {
		return id
	}
	return ""
}

// GetPagination 从请求获取分页参数
func GetPagination(c *gin.Context) (page, pageSize int) {
	page = 1
	pageSize = 20

	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	if ps := c.Query("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 100 {
			pageSize = v
		}
	}

	return page, pageSize
}

// ============================================================
// User Handler
// ============================================================

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) List(c *gin.Context) {
	users, err := h.svc.ListAll(c.Request.Context())
	if err != nil {
		InternalError(c, "获取用户列表失败: "+err.Error())
		return
	}
	Success(c, gin.H{"items": users})
}

func (h *UserHandler) Get(c *gin.Context) {
	id := c.Param("id")
	Success(c, gin.H{"user_id": id})
}

// Search 搜索用户（按名字模糊匹配）
// GET /api/v1/users/search?q=xxx
func (h *UserHandler) Search(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		BadRequest(c, "搜索关键字不能为空")
		return
	}
	users, err := h.svc.Search(c.Request.Context(), q)
	if err != nil {
		InternalError(c, "搜索用户失败: "+err.Error())
		return
	}
	Success(c, gin.H{"items": users})
}

// ============================================================
// Material Handler
// ============================================================

type MaterialHandler struct {
	svc *service.MaterialService
}

func NewMaterialHandler(svc *service.MaterialService) *MaterialHandler {
	return &MaterialHandler{svc: svc}
}

func (h *MaterialHandler) List(c *gin.Context)           {
	Success(c, gin.H{"materials": []interface{}{}})
}
func (h *MaterialHandler) Create(c *gin.Context)         {
	Success(c, gin.H{"message": "Material created"})
}
func (h *MaterialHandler) Get(c *gin.Context)            {
	id := c.Param("id")
	Success(c, gin.H{"material_id": id})
}
func (h *MaterialHandler) Update(c *gin.Context)         {
	Success(c, gin.H{"message": "Material updated"})
}
func (h *MaterialHandler) ListCategories(c *gin.Context) {
	Success(c, gin.H{"categories": []interface{}{}})
}
