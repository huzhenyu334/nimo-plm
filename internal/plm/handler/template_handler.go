package handler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bitfantasy/nimo/internal/plm/entity"
	"github.com/bitfantasy/nimo/internal/plm/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// isVersionGreater 比较版本号 a > b（格式: "1.0", "2.1" 等）
func isVersionGreater(a, b string) bool {
	parseVer := func(v string) (int, int) {
		v = strings.TrimPrefix(v, "v")
		v = strings.TrimPrefix(v, "V")
		parts := strings.SplitN(v, ".", 2)
		major, _ := strconv.Atoi(parts[0])
		minor := 0
		if len(parts) > 1 {
			minor, _ = strconv.Atoi(parts[1])
		}
		return major, minor
	}
	aMajor, aMinor := parseVer(a)
	bMajor, bMinor := parseVer(b)
	if aMajor != bMajor {
		return aMajor > bMajor
	}
	return aMinor > bMinor
}

// nextVersion 自动递增版本号
func nextVersion(current string) string {
	current = strings.TrimPrefix(current, "v")
	current = strings.TrimPrefix(current, "V")
	parts := strings.SplitN(current, ".", 2)
	major, _ := strconv.Atoi(parts[0])
	minor := 0
	if len(parts) > 1 {
		minor, _ = strconv.Atoi(parts[1])
	}
	minor++
	if minor >= 10 {
		major++
		minor = 0
	}
	return fmt.Sprintf("%d.%d", major, minor)
}

// TemplateHandler 模板处理器
type TemplateHandler struct {
	svc *service.TemplateService
}

// NewTemplateHandler 创建模板处理器
func NewTemplateHandler(svc *service.TemplateService) *TemplateHandler {
	return &TemplateHandler{svc: svc}
}

// List 获取模板列表
func (h *TemplateHandler) List(c *gin.Context) {
	templateType := c.Query("type")
	productType := c.Query("product_type")
	activeOnly := c.Query("active_only") != "false"

	templates, err := h.svc.ListTemplates(c.Request.Context(), templateType, productType, activeOnly)
	if err != nil {
		InternalError(c, "Failed to list templates")
		return
	}

	Success(c, templates)
}

// Get 获取模板详情
func (h *TemplateHandler) Get(c *gin.Context) {
	id := c.Param("id")

	template, err := h.svc.GetTemplate(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "Template not found")
		return
	}

	Success(c, template)
}

// CreateTemplateRequest 创建模板请求
type CreateTemplateRequest struct {
	Code          string   `json:"code"`
	Name          string   `json:"name" binding:"required"`
	Description   string   `json:"description"`
	ProductType   string   `json:"product_type"`
	Phases        []string `json:"phases"`
	EstimatedDays int      `json:"estimated_days"`
}

// Create 创建模板
func (h *TemplateHandler) Create(c *gin.Context) {
	var req CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body")
		return
	}

	userID := GetUserID(c)

	// Auto-generate code if not provided
	code := req.Code
	if code == "" {
		code = fmt.Sprintf("TPL-%d", time.Now().UnixMilli()%100000)
	}

	template := &entity.ProjectTemplate{
		Code:          code,
		Name:          req.Name,
		Description:   req.Description,
		TemplateType:  "CUSTOM",
		ProductType:   req.ProductType,
		EstimatedDays: req.EstimatedDays,
		IsActive:      true,
		CreatedBy:     userID,
	}

	if err := h.svc.CreateTemplate(c.Request.Context(), template); err != nil {
		InternalError(c, "Failed to create template")
		return
	}

	Created(c, template)
}

// UpdateTemplateRequest 更新模板请求
type UpdateTemplateRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	ProductType   string `json:"product_type"`
	Version       string `json:"version"`
	EstimatedDays int    `json:"estimated_days"`
	IsActive      *bool  `json:"is_active"`
}

// Update 更新模板
func (h *TemplateHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body")
		return
	}

	template, err := h.svc.GetTemplate(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "Template not found")
		return
	}

	// 已发布的流程不能修改
	if template.Status == "published" {
		BadRequest(c, "已发布的流程不能修改，请先升级版本创建新的草稿")
		return
	}

	if req.Name != "" {
		template.Name = req.Name
	}
	if req.Description != "" {
		template.Description = req.Description
	}
	if req.ProductType != "" {
		template.ProductType = req.ProductType
	}
	if req.Version != "" {
		// 版本号必须比当前版本大
		if !isVersionGreater(req.Version, template.Version) {
			BadRequest(c, "新版本号必须大于当前版本 "+template.Version)
			return
		}
		template.Version = req.Version
	}
	if req.EstimatedDays > 0 {
		template.EstimatedDays = req.EstimatedDays
	}
	if req.IsActive != nil {
		template.IsActive = *req.IsActive
	}

	if err := h.svc.UpdateTemplate(c.Request.Context(), template); err != nil {
		InternalError(c, "Failed to update template")
		return
	}

	Success(c, template)
}

// Delete 删除模板
func (h *TemplateHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.svc.DeleteTemplate(c.Request.Context(), id); err != nil {
		if err.Error() == "cannot delete system template" {
			Forbidden(c, err.Error())
			return
		}
		InternalError(c, "Failed to delete template")
		return
	}

	Success(c, nil)
}

// DuplicateTemplateRequest 复制模板请求
type DuplicateTemplateRequest struct {
	NewCode string `json:"new_code" binding:"required"`
	NewName string `json:"new_name" binding:"required"`
}

// Duplicate 复制模板
func (h *TemplateHandler) Duplicate(c *gin.Context) {
	id := c.Param("id")

	var req DuplicateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body")
		return
	}

	userID := GetUserID(c)

	newTemplate, err := h.svc.DuplicateTemplate(c.Request.Context(), id, req.NewCode, req.NewName, userID)
	if err != nil {
		InternalError(c, "Failed to duplicate template: "+err.Error())
		return
	}

	Created(c, newTemplate)
}

// CreateTemplateTaskRequest 创建模板任务请求
type CreateTemplateTaskRequest struct {
	TaskCode             string   `json:"task_code" binding:"required"`
	Name                 string   `json:"name" binding:"required"`
	Description          string   `json:"description"`
	Phase                string   `json:"phase" binding:"required"`
	ParentTaskCode       string   `json:"parent_task_code"`
	TaskType             string   `json:"task_type"`
	DefaultAssigneeRole  string   `json:"default_assignee_role"`
	EstimatedDays        int      `json:"estimated_days"`
	IsCritical           bool     `json:"is_critical"`
	RequiresApproval     bool     `json:"requires_approval"`
	ApprovalType         string   `json:"approval_type"`
	AutoCreateFeishuTask bool     `json:"auto_create_feishu_task"`
	FeishuApprovalCode   string   `json:"feishu_approval_code"`
	SortOrder            int      `json:"sort_order"`
	DependsOn            []string `json:"depends_on"` // 前置任务 task_code 列表
}

// CreateTask 创建模板任务
func (h *TemplateHandler) CreateTask(c *gin.Context) {
	templateID := c.Param("id")

	var req CreateTemplateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body")
		return
	}

	task := &entity.TemplateTask{
		TemplateID:           templateID,
		TaskCode:             req.TaskCode,
		Name:                 req.Name,
		Description:          req.Description,
		Phase:                req.Phase,
		ParentTaskCode:       req.ParentTaskCode,
		TaskType:             req.TaskType,
		DefaultAssigneeRole:  req.DefaultAssigneeRole,
		EstimatedDays:        req.EstimatedDays,
		IsCritical:           req.IsCritical,
		RequiresApproval:     req.RequiresApproval,
		ApprovalType:         req.ApprovalType,
		AutoCreateFeishuTask: req.AutoCreateFeishuTask,
		FeishuApprovalCode:   req.FeishuApprovalCode,
		SortOrder:            req.SortOrder,
	}

	if task.TaskType == "" {
		task.TaskType = "TASK"
	}
	if task.EstimatedDays == 0 {
		task.EstimatedDays = 1
	}

	if err := h.svc.CreateTemplateTask(c.Request.Context(), task); err != nil {
		InternalError(c, "Failed to create task")
		return
	}

	Created(c, task)
}

// UpdateTask 更新模板任务
func (h *TemplateHandler) UpdateTask(c *gin.Context) {
	templateID := c.Param("id")
	taskCode := c.Param("taskCode")

	var req CreateTemplateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body")
		return
	}

	task := &entity.TemplateTask{
		TemplateID:           templateID,
		TaskCode:             taskCode,
		Name:                 req.Name,
		Description:          req.Description,
		Phase:                req.Phase,
		ParentTaskCode:       req.ParentTaskCode,
		TaskType:             req.TaskType,
		DefaultAssigneeRole:  req.DefaultAssigneeRole,
		EstimatedDays:        req.EstimatedDays,
		IsCritical:           req.IsCritical,
		RequiresApproval:     req.RequiresApproval,
		ApprovalType:         req.ApprovalType,
		AutoCreateFeishuTask: req.AutoCreateFeishuTask,
		FeishuApprovalCode:   req.FeishuApprovalCode,
		SortOrder:            req.SortOrder,
	}

	if err := h.svc.UpdateTemplateTask(c.Request.Context(), task); err != nil {
		InternalError(c, "Failed to update task")
		return
	}

	Success(c, task)
}

// DeleteTask 删除模板任务
func (h *TemplateHandler) DeleteTask(c *gin.Context) {
	templateID := c.Param("id")
	taskCode := c.Param("taskCode")

	if err := h.svc.DeleteTemplateTask(c.Request.Context(), templateID, taskCode); err != nil {
		InternalError(c, "Failed to delete task")
		return
	}

	Success(c, nil)
}

// CreateProjectFromTemplateRequest 从模板创建项目请求
type CreateProjectFromTemplateRequest struct {
	TemplateID      string            `json:"template_id" binding:"required"`
	ProjectName     string            `json:"project_name" binding:"required"`
	ProjectCode     string            `json:"project_code" binding:"required"`
	ProductID       string            `json:"product_id"`
	StartDate       string            `json:"start_date" binding:"required"`
	PMID            string            `json:"pm_user_id" binding:"required"`
	SkipWeekends    bool              `json:"skip_weekends"`
	RoleAssignments map[string]string `json:"role_assignments"`
}

// CreateProjectFromTemplate 从模板创建项目
func (h *TemplateHandler) CreateProjectFromTemplate(c *gin.Context) {
	var req CreateProjectFromTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		BadRequest(c, "Invalid start_date format, use YYYY-MM-DD")
		return
	}

	userID := GetUserID(c)

	input := &service.CreateProjectFromTemplateInput{
		TemplateID:      req.TemplateID,
		ProjectName:     req.ProjectName,
		ProjectCode:     req.ProjectCode,
		ProductID:       req.ProductID,
		StartDate:       startDate,
		PMID:            req.PMID,
		SkipWeekends:    req.SkipWeekends,
		RoleAssignments: req.RoleAssignments,
	}

	project, err := h.svc.CreateProjectFromTemplate(c.Request.Context(), input, userID)
	if err != nil {
		InternalError(c, "Failed to create project: "+err.Error())
		return
	}

	Created(c, project)
}

// BatchSaveTasksRequest 批量保存任务请求
type BatchSaveTasksRequest struct {
	Tasks   []CreateTemplateTaskRequest `json:"tasks" binding:"required"`
	Version string                      `json:"version"` // 可选：同时升级版本
}

// BatchSaveTasks 批量保存任务（删除旧任务，插入新任务）
func (h *TemplateHandler) BatchSaveTasks(c *gin.Context) {
	templateID := c.Param("id")

	var req BatchSaveTasksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	// 获取模板
	template, err := h.svc.GetTemplate(c.Request.Context(), templateID)
	if err != nil {
		NotFound(c, "Template not found")
		return
	}

	// 已发布的流程不能直接修改
	if template.Status == "published" {
		BadRequest(c, "已发布的流程不能修改，请先升级版本创建新的草稿")
		return
	}

	// 如果指定了版本号，验证
	if req.Version != "" {
		if !isVersionGreater(req.Version, template.Version) {
			BadRequest(c, fmt.Sprintf("版本号 %s 必须大于当前版本 %s", req.Version, template.Version))
			return
		}
	}

	// 转换为entity
	var tasks []entity.TemplateTask
	for i, t := range req.Tasks {
		task := entity.TemplateTask{
			ID:                   uuid.New().String(),
			TemplateID:           templateID,
			TaskCode:             t.TaskCode,
			Name:                 t.Name,
			Description:          t.Description,
			Phase:                t.Phase,
			ParentTaskCode:       t.ParentTaskCode,
			TaskType:             t.TaskType,
			DefaultAssigneeRole:  t.DefaultAssigneeRole,
			EstimatedDays:        t.EstimatedDays,
			IsCritical:           t.IsCritical,
			RequiresApproval:     t.RequiresApproval,
			ApprovalType:         t.ApprovalType,
			AutoCreateFeishuTask: t.AutoCreateFeishuTask,
			FeishuApprovalCode:   t.FeishuApprovalCode,
			SortOrder:            i,
		}
		if task.TaskType == "" {
			task.TaskType = "TASK"
		}
		if task.EstimatedDays == 0 {
			task.EstimatedDays = 1
		}
		tasks = append(tasks, task)
	}

	// 构建依赖关系
	var dependencies []entity.TemplateTaskDependency
	for _, t := range req.Tasks {
		for _, depCode := range t.DependsOn {
			dependencies = append(dependencies, entity.TemplateTaskDependency{
				ID:                uuid.New().String(),
				TemplateID:        templateID,
				TaskCode:          t.TaskCode,
				DependsOnTaskCode: depCode,
				DependencyType:    "FS",
				LagDays:           0,
			})
		}
	}

	// 批量保存
	if err := h.svc.BatchSaveTasks(c.Request.Context(), templateID, tasks, dependencies); err != nil {
		InternalError(c, "Failed to save tasks: "+err.Error())
		return
	}

	// 如果指定了版本号，更新模板版本
	if req.Version != "" {
		template.Version = req.Version
	} else {
		// 自动递增版本
		template.Version = nextVersion(template.Version)
	}
	template.EstimatedDays = calcEstimatedDays(tasks)
	if err := h.svc.UpdateTemplate(c.Request.Context(), template); err != nil {
		InternalError(c, "Failed to update template version")
		return
	}

	Success(c, gin.H{
		"task_count": len(tasks),
		"version":    template.Version,
	})
}

// calcEstimatedDays 计算模板预估工期（取各阶段最长任务路径之和）
func calcEstimatedDays(tasks []entity.TemplateTask) int {
	phaseMax := make(map[string]int)
	for _, t := range tasks {
		if t.TaskType == "SUBTASK" {
			continue
		}
		if t.EstimatedDays > phaseMax[t.Phase] {
			phaseMax[t.Phase] = t.EstimatedDays
		}
	}
	total := 0
	for _, d := range phaseMax {
		total += d
	}
	return total
}

// Publish 发布流程（锁定当前版本）
func (h *TemplateHandler) Publish(c *gin.Context) {
	id := c.Param("id")

	template, err := h.svc.GetTemplate(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "Template not found")
		return
	}

	if template.Status == "published" {
		BadRequest(c, "该流程已经发布")
		return
	}

	// 检查是否有任务
	if len(template.Tasks) == 0 {
		BadRequest(c, "流程至少需要包含一个任务才能发布")
		return
	}

	now := time.Now()
	template.Status = "published"
	template.PublishedAt = &now
	if template.BaseCode == "" {
		template.BaseCode = template.Code
	}

	if err := h.svc.UpdateTemplate(c.Request.Context(), template); err != nil {
		InternalError(c, "发布失败")
		return
	}

	Success(c, template)
}

// UpgradeVersion 升级版本（从已发布版本创建新草稿）
func (h *TemplateHandler) UpgradeVersion(c *gin.Context) {
	id := c.Param("id")

	template, err := h.svc.GetTemplate(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "Template not found")
		return
	}

	if template.Status != "published" {
		BadRequest(c, "只有已发布的流程才能升级版本")
		return
	}

	// 支持前端传入版本号
	var req struct {
		Version string `json:"version"`
	}
	c.ShouldBindJSON(&req)

	userID := GetUserID(c)
	newVersion := req.Version
	if newVersion == "" {
		newVersion = nextVersion(template.Version)
	}

	// 版本号校验
	if !isVersionGreater(newVersion, template.Version) {
		BadRequest(c, fmt.Sprintf("新版本号 %s 必须大于当前版本 %s", newVersion, template.Version))
		return
	}

	// 创建新的草稿版本（复制任务）
	newTemplate, err := h.svc.CreateNewVersion(c.Request.Context(), template, newVersion, userID)
	if err != nil {
		InternalError(c, "升级版本失败: "+err.Error())
		return
	}

	Created(c, newTemplate)
}

// Revert 撤销草稿，回退到上一个已发布版本
func (h *TemplateHandler) Revert(c *gin.Context) {
	id := c.Param("id")

	prevVersion, err := h.svc.RevertTemplate(c.Request.Context(), id)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, prevVersion)
}

// ListVersions 获取流程版本历史
func (h *TemplateHandler) ListVersions(c *gin.Context) {
	id := c.Param("id")

	template, err := h.svc.GetTemplate(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "Template not found")
		return
	}

	baseCode := template.BaseCode
	if baseCode == "" {
		baseCode = template.Code
	}

	versions, err := h.svc.ListVersions(c.Request.Context(), baseCode)
	if err != nil {
		InternalError(c, "获取版本历史失败")
		return
	}

	Success(c, versions)
}
