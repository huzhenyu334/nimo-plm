package handler

import (
	"github.com/bitfantasy/nimo/internal/srm/service"
	"github.com/gin-gonic/gin"
)

// ProjectHandler 采购项目处理器
type ProjectHandler struct {
	svc *service.SRMProjectService
}

func NewProjectHandler(svc *service.SRMProjectService) *ProjectHandler {
	return &ProjectHandler{svc: svc}
}

// ListProjects 采购项目列表
// GET /api/v1/srm/projects?status=xxx&type=xxx&phase=xxx&plm_project_id=xxx&search=xxx
func (h *ProjectHandler) ListProjects(c *gin.Context) {
	page, pageSize := GetPagination(c)
	filters := map[string]string{
		"status":         c.Query("status"),
		"type":           c.Query("type"),
		"phase":          c.Query("phase"),
		"plm_project_id": c.Query("plm_project_id"),
		"search":         c.Query("search"),
	}

	items, total, err := h.svc.ListProjects(c.Request.Context(), page, pageSize, filters)
	if err != nil {
		InternalError(c, "获取采购项目列表失败: "+err.Error())
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

// GetProject 采购项目详情
// GET /api/v1/srm/projects/:id
func (h *ProjectHandler) GetProject(c *gin.Context) {
	id := c.Param("id")
	project, err := h.svc.GetProject(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "采购项目不存在")
		return
	}
	Success(c, project)
}

// CreateProject 创建采购项目
// POST /api/v1/srm/projects
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	var req service.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	userID := GetUserID(c)
	project, err := h.svc.CreateProject(c.Request.Context(), userID, &req)
	if err != nil {
		InternalError(c, "创建采购项目失败: "+err.Error())
		return
	}

	Created(c, project)
}

// UpdateProject 更新采购项目
// PUT /api/v1/srm/projects/:id
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	id := c.Param("id")
	var req service.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	project, err := h.svc.UpdateProject(c.Request.Context(), id, &req)
	if err != nil {
		InternalError(c, "更新采购项目失败: "+err.Error())
		return
	}

	Success(c, project)
}

// GetProjectProgress 获取采购项目进度
// GET /api/v1/srm/projects/:id/progress
func (h *ProjectHandler) GetProjectProgress(c *gin.Context) {
	id := c.Param("id")
	project, err := h.svc.GetProjectProgress(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "采购项目不存在")
		return
	}
	Success(c, gin.H{
		"total":          project.TotalItems,
		"sourcing_count": project.SourcingCount,
		"ordered_count":  project.OrderedCount,
		"received_count": project.ReceivedCount,
		"passed_count":   project.PassedCount,
		"failed_count":   project.FailedCount,
		"percent":        calcPercent(project.PassedCount, project.TotalItems),
		"status":         project.Status,
	})
}

// ListActivityLogs 查询操作日志
// GET /api/v1/srm/projects/:id/activity-logs?entity_type=project
func (h *ProjectHandler) ListActivityLogs(c *gin.Context) {
	entityType := c.Query("entity_type")
	entityID := c.Param("id")
	if entityType == "" {
		entityType = "project"
	}
	page, pageSize := GetPagination(c)

	items, total, err := h.svc.ListActivityLogs(c.Request.Context(), entityType, entityID, page, pageSize)
	if err != nil {
		InternalError(c, "获取操作日志失败: "+err.Error())
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

// ListDelayRequests 延期审批列表
// GET /api/v1/srm/delay-requests?srm_project_id=xxx&status=xxx
func (h *ProjectHandler) ListDelayRequests(c *gin.Context) {
	page, pageSize := GetPagination(c)
	filters := map[string]string{
		"srm_project_id": c.Query("srm_project_id"),
		"status":         c.Query("status"),
	}

	items, total, err := h.svc.ListDelayRequests(c.Request.Context(), page, pageSize, filters)
	if err != nil {
		InternalError(c, "获取延期申请列表失败: "+err.Error())
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

// GetDelayRequest 延期审批详情
// GET /api/v1/srm/delay-requests/:id
func (h *ProjectHandler) GetDelayRequest(c *gin.Context) {
	id := c.Param("id")
	dr, err := h.svc.GetDelayRequest(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "延期申请不存在")
		return
	}
	Success(c, dr)
}

// CreateDelayRequest 创建延期申请
// POST /api/v1/srm/delay-requests
func (h *ProjectHandler) CreateDelayRequest(c *gin.Context) {
	var req service.CreateDelayRequestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	userID := GetUserID(c)
	dr, err := h.svc.CreateDelayRequest(c.Request.Context(), userID, &req)
	if err != nil {
		InternalError(c, "创建延期申请失败: "+err.Error())
		return
	}

	Created(c, dr)
}

// ApproveDelayRequest 审批通过延期申请
// POST /api/v1/srm/delay-requests/:id/approve
func (h *ProjectHandler) ApproveDelayRequest(c *gin.Context) {
	id := c.Param("id")
	userID := GetUserID(c)

	dr, err := h.svc.ApproveDelayRequest(c.Request.Context(), id, userID)
	if err != nil {
		InternalError(c, "审批失败: "+err.Error())
		return
	}

	Success(c, dr)
}

// RejectDelayRequest 驳回延期申请
// POST /api/v1/srm/delay-requests/:id/reject
func (h *ProjectHandler) RejectDelayRequest(c *gin.Context) {
	id := c.Param("id")
	userID := GetUserID(c)

	dr, err := h.svc.RejectDelayRequest(c.Request.Context(), id, userID)
	if err != nil {
		InternalError(c, "驳回失败: "+err.Error())
		return
	}

	Success(c, dr)
}

// ListEntityActivityLogs 查询任意实体的操作日志
// GET /api/v1/srm/activity-logs?entity_type=pr&entity_id=xxx
func (h *ProjectHandler) ListEntityActivityLogs(c *gin.Context) {
	entityType := c.Query("entity_type")
	entityID := c.Query("entity_id")
	if entityType == "" || entityID == "" {
		BadRequest(c, "entity_type和entity_id必填")
		return
	}
	page, pageSize := GetPagination(c)

	items, total, err := h.svc.ListActivityLogs(c.Request.Context(), entityType, entityID, page, pageSize)
	if err != nil {
		InternalError(c, "获取操作日志失败: "+err.Error())
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

func calcPercent(done, total int) int {
	if total == 0 {
		return 0
	}
	return done * 100 / total
}
