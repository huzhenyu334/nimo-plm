package handler

import (
	"encoding/json"
	"time"

	"github.com/bitfantasy/nimo/internal/plm/entity"
	"github.com/bitfantasy/nimo/internal/plm/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TaskFormHandler 任务表单处理器
type TaskFormHandler struct {
	formRepo    *repository.TaskFormRepository
	projectRepo *repository.ProjectRepository
}

// NewTaskFormHandler 创建任务表单处理器
func NewTaskFormHandler(formRepo *repository.TaskFormRepository, projectRepo *repository.ProjectRepository) *TaskFormHandler {
	return &TaskFormHandler{
		formRepo:    formRepo,
		projectRepo: projectRepo,
	}
}

// GetTaskForm 获取任务表单定义
// GET /projects/:id/tasks/:taskId/form
func (h *TaskFormHandler) GetTaskForm(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		BadRequest(c, "Task ID is required")
		return
	}

	form, err := h.formRepo.FindByTaskID(c.Request.Context(), taskID)
	if err != nil {
		InternalError(c, "查询表单失败: "+err.Error())
		return
	}
	if form == nil {
		Success(c, nil)
		return
	}

	Success(c, form)
}

// UpsertTaskForm 创建或更新任务表单
// PUT /projects/:id/tasks/:taskId/form
func (h *TaskFormHandler) UpsertTaskForm(c *gin.Context) {
	projectID := c.Param("id")
	taskID := c.Param("taskId")
	if projectID == "" || taskID == "" {
		BadRequest(c, "Project ID and Task ID are required")
		return
	}

	userID := GetUserID(c)

	// 验证当前用户是项目经理
	project, err := h.projectRepo.FindByID(c.Request.Context(), projectID)
	if err != nil {
		NotFound(c, "项目不存在")
		return
	}
	if project.ManagerID != userID {
		Forbidden(c, "只有项目经理才能配置任务表单")
		return
	}

	var req struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Fields      json.RawMessage `json:"fields"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	ctx := c.Request.Context()
	now := time.Now()

	// 查找已有表单
	existing, err := h.formRepo.FindByTaskID(ctx, taskID)
	if err != nil {
		InternalError(c, "查询表单失败: "+err.Error())
		return
	}

	if existing != nil {
		// 更新
		if req.Name != "" {
			existing.Name = req.Name
		}
		if req.Description != "" {
			existing.Description = req.Description
		}
		if req.Fields != nil {
			existing.Fields = req.Fields
		}
		existing.UpdatedAt = now

		if err := h.formRepo.Update(ctx, existing); err != nil {
			InternalError(c, "更新表单失败: "+err.Error())
			return
		}
		Success(c, existing)
	} else {
		// 创建
		name := req.Name
		if name == "" {
			name = "完成表单"
		}
		fields := req.Fields
		if fields == nil {
			fields = json.RawMessage(`[]`)
		}

		form := &entity.TaskForm{
			ID:          uuid.New().String()[:32],
			TaskID:      taskID,
			Name:        name,
			Description: req.Description,
			Fields:      fields,
			CreatedBy:   userID,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		if err := h.formRepo.Create(ctx, form); err != nil {
			InternalError(c, "创建表单失败: "+err.Error())
			return
		}
		Created(c, form)
	}
}

// GetTemplateTaskForms 获取模板的所有任务表单
// GET /templates/:id/task-forms
func (h *TaskFormHandler) GetTemplateTaskForms(c *gin.Context) {
	templateID := c.Param("id")
	if templateID == "" {
		BadRequest(c, "Template ID is required")
		return
	}

	forms, err := h.formRepo.FindTemplateFormsByTemplateID(c.Request.Context(), templateID)
	if err != nil {
		InternalError(c, "查询模板表单失败: "+err.Error())
		return
	}

	Success(c, forms)
}

// SaveTemplateTaskForm 保存模板任务表单
// POST /templates/:id/task-forms
func (h *TaskFormHandler) SaveTemplateTaskForm(c *gin.Context) {
	templateID := c.Param("id")
	if templateID == "" {
		BadRequest(c, "Template ID is required")
		return
	}

	var req struct {
		TaskCode string          `json:"task_code"`
		Name     string          `json:"name"`
		Fields   json.RawMessage `json:"fields"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}
	if req.TaskCode == "" {
		BadRequest(c, "task_code is required")
		return
	}

	now := time.Now()
	name := req.Name
	if name == "" {
		name = "完成表单"
	}
	fields := req.Fields
	if fields == nil {
		fields = json.RawMessage(`[]`)
	}

	form := &entity.TemplateTaskForm{
		ID:         uuid.New().String()[:32],
		TemplateID: templateID,
		TaskCode:   req.TaskCode,
		Name:       name,
		Fields:     fields,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := h.formRepo.UpsertTemplateForm(c.Request.Context(), form); err != nil {
		InternalError(c, "保存模板表单失败: "+err.Error())
		return
	}

	Success(c, form)
}

// SaveFormDraft 保存表单草稿
// PUT /my/tasks/:taskId/form-draft
func (h *TaskFormHandler) SaveFormDraft(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		BadRequest(c, "Task ID is required")
		return
	}

	userID := GetUserID(c)

	var req struct {
		FormData map[string]interface{} `json:"form_data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	ctx := c.Request.Context()

	// 查找表单定义
	form, err := h.formRepo.FindByTaskID(ctx, taskID)
	if err != nil || form == nil {
		BadRequest(c, "该任务没有表单")
		return
	}

	submission := &entity.TaskFormSubmission{
		ID:          uuid.New().String()[:32],
		FormID:      form.ID,
		TaskID:      taskID,
		Data:        entity.JSONB(req.FormData),
		SubmittedBy: userID,
		SubmittedAt: time.Now(),
	}

	if err := h.formRepo.UpsertDraftSubmission(ctx, submission); err != nil {
		InternalError(c, "保存草稿失败: "+err.Error())
		return
	}

	Success(c, gin.H{"message": "草稿已保存"})
}

// GetFormDraft 获取表单草稿
// GET /my/tasks/:taskId/form-draft
func (h *TaskFormHandler) GetFormDraft(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		BadRequest(c, "Task ID is required")
		return
	}

	submission, err := h.formRepo.FindDraftSubmission(c.Request.Context(), taskID)
	if err != nil {
		InternalError(c, "查询草稿失败: "+err.Error())
		return
	}

	Success(c, submission)
}

// GetFormSubmission 获取最新表单提交
// GET /projects/:id/tasks/:taskId/form/submission
func (h *TaskFormHandler) GetFormSubmission(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		BadRequest(c, "Task ID is required")
		return
	}

	submission, err := h.formRepo.FindLatestSubmission(c.Request.Context(), taskID)
	if err != nil {
		InternalError(c, "查询表单提交失败: "+err.Error())
		return
	}
	if submission == nil {
		Success(c, nil)
		return
	}

	Success(c, submission)
}
