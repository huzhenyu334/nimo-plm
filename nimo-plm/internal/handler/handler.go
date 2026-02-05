package handler

import (
	"strconv"

	"github.com/bitfantasy/nimo-plm/internal/config"
	"github.com/bitfantasy/nimo-plm/internal/service"
	"github.com/gin-gonic/gin"
)

// Handlers 处理器集合
type Handlers struct {
	Auth     *AuthHandler
	User     *UserHandler
	Product  *ProductHandler
	Material *MaterialHandler
	BOM      *BOMHandler
	Project  *ProjectHandler
	Task     *ProjectHandler  // Task methods are on ProjectHandler
	ECN      *ECNHandler
	Document *DocumentHandler
}

// NewHandlers 创建处理器集合
func NewHandlers(svc *service.Services, cfg *config.Config) *Handlers {
	projectHandler := NewProjectHandler(svc.Project)
	return &Handlers{
		Auth:     NewAuthHandler(svc.Auth, cfg),
		User:     NewUserHandler(svc.User),
		Product:  NewProductHandler(svc.Product),
		Material: NewMaterialHandler(svc.Material),
		BOM:      NewBOMHandler(svc.BOM),
		Project:  projectHandler,
		Task:     projectHandler,  // Reuse ProjectHandler for task routes
		ECN:      NewECNHandler(svc.ECN),
		Document: NewDocumentHandler(svc.Document),
	}
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
	Success(c, gin.H{"users": []interface{}{}})
}
func (h *UserHandler) Get(c *gin.Context)  {
	id := c.Param("id")
	Success(c, gin.H{"user_id": id})
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
