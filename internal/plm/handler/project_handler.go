package handler

import (
	"strconv"

	"github.com/bitfantasy/nimo/internal/plm/service"
	"github.com/gin-gonic/gin"
)

// ProjectHandler 项目处理器
type ProjectHandler struct {
	svc *service.ProjectService
}

// NewProjectHandler 创建项目处理器
func NewProjectHandler(svc *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{svc: svc}
}

// ============================================================
// 项目相关接口
// ============================================================

// ListProjects 获取项目列表
func (h *ProjectHandler) ListProjects(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 500 {
		pageSize = 20
	}

	filters := map[string]interface{}{
		"keyword":    c.Query("keyword"),
		"status":     c.Query("status"),
		"product_id": c.Query("product_id"),
		"owner_id":   c.Query("owner_id"),
	}

	result, err := h.svc.ListProjects(c.Request.Context(), page, pageSize, filters)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, result)
}

// GetProject 获取项目详情
func (h *ProjectHandler) GetProject(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequest(c, "Project ID is required")
		return
	}

	project, err := h.svc.GetProject(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "Project not found")
		return
	}

	Success(c, project)
}

// CreateProject 创建项目
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	var req service.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	userID := GetUserID(c)
	project, err := h.svc.CreateProject(c.Request.Context(), userID, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Created(c, project)
}

// UpdateProject 更新项目
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequest(c, "Project ID is required")
		return
	}

	var req service.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	project, err := h.svc.UpdateProject(c.Request.Context(), id, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, project)
}

// DeleteProject 删除项目
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequest(c, "Project ID is required")
		return
	}

	if err := h.svc.DeleteProject(c.Request.Context(), id); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, nil)
}

// UpdateProjectStatus 更新项目状态
func (h *ProjectHandler) UpdateProjectStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequest(c, "Project ID is required")
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	project, err := h.svc.UpdateProjectStatus(c.Request.Context(), id, req.Status)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, project)
}

// ListPhases 获取项目阶段列表
func (h *ProjectHandler) ListPhases(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		BadRequest(c, "Project ID is required")
		return
	}

	phases, err := h.svc.ListPhases(c.Request.Context(), projectID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, phases)
}

// UpdatePhaseStatus 更新阶段状态
func (h *ProjectHandler) UpdatePhaseStatus(c *gin.Context) {
	phaseID := c.Param("phaseId")
	if phaseID == "" {
		BadRequest(c, "Phase ID is required")
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	phase, err := h.svc.UpdatePhaseStatus(c.Request.Context(), phaseID, req.Status)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, phase)
}

// ============================================================
// 任务相关接口
// ============================================================

// ListTasks 获取任务列表
func (h *ProjectHandler) ListTasks(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		BadRequest(c, "Project ID is required")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 500 {
		pageSize = 20
	}

	filters := map[string]interface{}{
		"phase_id":    c.Query("phase_id"),
		"status":      c.Query("status"),
		"assignee_id": c.Query("assignee_id"),
		"priority":    c.Query("priority"),
	}

	result, err := h.svc.ListTasks(c.Request.Context(), projectID, page, pageSize, filters)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, result)
}

// GetTask 获取任务详情
func (h *ProjectHandler) GetTask(c *gin.Context) {
	id := c.Param("taskId")
	if id == "" {
		BadRequest(c, "Task ID is required")
		return
	}

	task, err := h.svc.GetTask(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "Task not found")
		return
	}

	Success(c, task)
}

// CreateTask 创建任务
func (h *ProjectHandler) CreateTask(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		BadRequest(c, "Project ID is required")
		return
	}

	var req service.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	userID := GetUserID(c)
	task, err := h.svc.CreateTask(c.Request.Context(), projectID, userID, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Created(c, task)
}

// UpdateTask 更新任务
func (h *ProjectHandler) UpdateTask(c *gin.Context) {
	id := c.Param("taskId")
	if id == "" {
		BadRequest(c, "Task ID is required")
		return
	}

	var req service.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	task, err := h.svc.UpdateTask(c.Request.Context(), id, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, task)
}

// DeleteTask 删除任务
func (h *ProjectHandler) DeleteTask(c *gin.Context) {
	id := c.Param("taskId")
	if id == "" {
		BadRequest(c, "Task ID is required")
		return
	}

	if err := h.svc.DeleteTask(c.Request.Context(), id); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, nil)
}

// UpdateTaskStatus 更新任务状态
func (h *ProjectHandler) UpdateTaskStatus(c *gin.Context) {
	id := c.Param("taskId")
	if id == "" {
		BadRequest(c, "Task ID is required")
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	task, err := h.svc.UpdateTaskStatus(c.Request.Context(), id, req.Status)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, task)
}

// ListSubTasks 获取子任务列表
func (h *ProjectHandler) ListSubTasks(c *gin.Context) {
	parentID := c.Param("taskId")
	if parentID == "" {
		BadRequest(c, "Task ID is required")
		return
	}

	tasks, err := h.svc.ListSubTasks(c.Request.Context(), parentID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, tasks)
}

// ListTaskComments 获取任务评论列表
func (h *ProjectHandler) ListTaskComments(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		BadRequest(c, "Task ID is required")
		return
	}

	comments, err := h.svc.ListTaskComments(c.Request.Context(), taskID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, comments)
}

// AddTaskComment 添加任务评论
func (h *ProjectHandler) AddTaskComment(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		BadRequest(c, "Task ID is required")
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	userID := GetUserID(c)
	comment, err := h.svc.AddTaskComment(c.Request.Context(), taskID, userID, req.Content)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Created(c, comment)
}

// ListTaskDependencies 获取任务依赖列表
func (h *ProjectHandler) ListTaskDependencies(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		BadRequest(c, "Task ID is required")
		return
	}

	deps, err := h.svc.ListTaskDependencies(c.Request.Context(), taskID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, deps)
}

// AddTaskDependency 添加任务依赖
func (h *ProjectHandler) AddTaskDependency(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		BadRequest(c, "Task ID is required")
		return
	}

	var req struct {
		DependsOnID    string `json:"depends_on_id" binding:"required"`
		DependencyType string `json:"dependency_type"`
		LagDays        int    `json:"lag_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	if req.DependencyType == "" {
		req.DependencyType = "finish_to_start"
	}

	dep, err := h.svc.AddTaskDependency(c.Request.Context(), taskID, req.DependsOnID, req.DependencyType, req.LagDays)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Created(c, dep)
}

// RemoveTaskDependency 移除任务依赖
func (h *ProjectHandler) RemoveTaskDependency(c *gin.Context) {
	depID := c.Param("depId")
	if depID == "" {
		BadRequest(c, "Dependency ID is required")
		return
	}

	if err := h.svc.RemoveTaskDependency(c.Request.Context(), depID); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, nil)
}

// GetMyTasks 获取我的任务
func (h *ProjectHandler) GetMyTasks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 500 {
		pageSize = 20
	}

	filters := map[string]interface{}{
		"status":   c.Query("status"),
		"priority": c.Query("priority"),
	}

	userID := GetUserID(c)
	result, err := h.svc.GetMyTasks(c.Request.Context(), userID, page, pageSize, filters)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, result)
}

// CompleteMyTask 完成我的任务
// POST /my/tasks/:taskId/complete
func (h *ProjectHandler) CompleteMyTask(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		BadRequest(c, "Task ID is required")
		return
	}

	userID := GetUserID(c)

	var req struct {
		FormData map[string]interface{} `json:"form_data"`
	}
	// form_data 可选，不 bind required
	c.ShouldBindJSON(&req)

	if err := h.svc.CompleteMyTask(c.Request.Context(), taskID, userID, req.FormData); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "任务已提交"})
}

// ConfirmTask 确认任务
// POST /projects/:id/tasks/:taskId/confirm
func (h *ProjectHandler) ConfirmTask(c *gin.Context) {
	projectID := c.Param("id")
	taskID := c.Param("taskId")
	if projectID == "" || taskID == "" {
		BadRequest(c, "Project ID and Task ID are required")
		return
	}

	userID := GetUserID(c)

	if err := h.svc.ConfirmTask(c.Request.Context(), projectID, taskID, userID); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "任务已确认"})
}

// RejectTask 驳回任务
// POST /projects/:id/tasks/:taskId/reject
func (h *ProjectHandler) RejectTask(c *gin.Context) {
	projectID := c.Param("id")
	taskID := c.Param("taskId")
	if projectID == "" || taskID == "" {
		BadRequest(c, "Project ID and Task ID are required")
		return
	}

	userID := GetUserID(c)

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	if err := h.svc.RejectTask(c.Request.Context(), projectID, taskID, userID, req.Reason); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "任务已驳回"})
}

// AssignRoles 批量角色分配
// POST /api/v1/projects/:id/assign-roles
func (h *ProjectHandler) AssignRoles(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		BadRequest(c, "Project ID is required")
		return
	}

	userID := GetUserID(c)

	var req struct {
		Assignments []struct {
			Role   string `json:"role"`
			UserID string `json:"user_id"`
		} `json:"assignments"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	if len(req.Assignments) == 0 {
		BadRequest(c, "至少需要一个角色分配")
		return
	}

	assignments := make([]service.RoleAssignment, len(req.Assignments))
	for i, a := range req.Assignments {
		assignments[i] = service.RoleAssignment{
			RoleCode: a.Role,
			UserID:   a.UserID,
		}
	}

	if err := h.svc.AssignRoles(c.Request.Context(), projectID, userID, assignments); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "角色分配成功"})
}

// GetOverdueTasks 获取逾期任务
func (h *ProjectHandler) GetOverdueTasks(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		BadRequest(c, "Project ID is required")
		return
	}

	tasks, err := h.svc.GetOverdueTasks(c.Request.Context(), projectID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, tasks)
}
