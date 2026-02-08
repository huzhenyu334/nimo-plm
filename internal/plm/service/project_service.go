package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bitfantasy/nimo/internal/plm/entity"
	"github.com/bitfantasy/nimo/internal/plm/sse"
	"github.com/bitfantasy/nimo/internal/plm/repository"
	"github.com/google/uuid"
)

// ProjectService 项目服务
type ProjectService struct {
	projectRepo  *repository.ProjectRepository
	taskRepo     *repository.TaskRepository
	productRepo  *repository.ProductRepository
	feishuSvc    *FeishuIntegrationService
	taskFormRepo *repository.TaskFormRepository
}

// NewProjectService 创建项目服务
func NewProjectService(
	projectRepo *repository.ProjectRepository,
	taskRepo *repository.TaskRepository,
	productRepo *repository.ProductRepository,
	feishuSvc *FeishuIntegrationService,
	taskFormRepo *repository.TaskFormRepository,
) *ProjectService {
	return &ProjectService{
		projectRepo:  projectRepo,
		taskRepo:     taskRepo,
		productRepo:  productRepo,
		feishuSvc:    feishuSvc,
		taskFormRepo: taskFormRepo,
	}
}

// CreateProjectRequest 创建项目请求
type CreateProjectRequest struct {
	Name         string     `json:"name" binding:"required"`
	ProductID    string     `json:"product_id"`
	Description  string     `json:"description"`
	OwnerID      string     `json:"owner_id"`
	PlannedStart *time.Time `json:"planned_start"`
	PlannedEnd   *time.Time `json:"planned_end"`
}

// UpdateProjectRequest 更新项目请求
type UpdateProjectRequest struct {
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	OwnerID      string     `json:"owner_id"`
	PlannedStart *time.Time `json:"planned_start"`
	PlannedEnd   *time.Time `json:"planned_end"`
	CurrentPhase string     `json:"current_phase"`
}

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	Name           string     `json:"name" binding:"required"`
	PhaseID        string     `json:"phase_id"`
	ParentTaskID   string     `json:"parent_task_id"`
	Description    string     `json:"description"`
	TaskType       string     `json:"task_type"`
	Priority       string     `json:"priority"`
	AssigneeID     string     `json:"assignee_id"`
	ReviewerID     string     `json:"reviewer_id"`
	PlannedStart   *time.Time `json:"planned_start"`
	PlannedEnd     *time.Time `json:"planned_end"`
	DueDate        *time.Time `json:"due_date"`
	EstimatedHours float64    `json:"estimated_hours"`
}

// UpdateTaskRequest 更新任务请求
type UpdateTaskRequest struct {
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	Priority       string     `json:"priority"`
	AssigneeID     string     `json:"assignee_id"`
	ReviewerID     string     `json:"reviewer_id"`
	PlannedStart   *time.Time `json:"planned_start"`
	PlannedEnd     *time.Time `json:"planned_end"`
	DueDate        *time.Time `json:"due_date"`
	EstimatedHours float64    `json:"estimated_hours"`
	ActualHours    float64    `json:"actual_hours"`
	Progress       int        `json:"progress"`
}

// ProjectListResult 项目列表结果
type ProjectListResult struct {
	Items      []entity.Project `json:"items"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

// TaskListResult 任务列表结果
type TaskListResult struct {
	Items      []entity.Task `json:"items"`
	Total      int64         `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int           `json:"total_pages"`
}

// ============================================================
// 项目相关操作
// ============================================================

// ListProjects 获取项目列表
func (s *ProjectService) ListProjects(ctx context.Context, page, pageSize int, filters map[string]interface{}) (*ProjectListResult, error) {
	projects, total, err := s.projectRepo.List(ctx, page, pageSize, filters)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &ProjectListResult{
		Items:      projects,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// GetProject 获取项目详情
func (s *ProjectService) GetProject(ctx context.Context, id string) (*entity.Project, error) {
	project, err := s.projectRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find project: %w", err)
	}
	return project, nil
}

// CreateProject 创建项目
func (s *ProjectService) CreateProject(ctx context.Context, userID string, req *CreateProjectRequest) (*entity.Project, error) {
	// 如果关联产品，验证产品存在
	if req.ProductID != "" {
		_, err := s.productRepo.FindByID(ctx, req.ProductID)
		if err != nil {
			return nil, fmt.Errorf("product not found: %w", err)
		}
	}

	// 生成项目编码
	code, err := s.projectRepo.GenerateCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generate code: %w", err)
	}

	ownerID := req.OwnerID
	if ownerID == "" {
		ownerID = userID
	}

	now := time.Now()
	var productID *string
	if req.ProductID != "" {
		productID = &req.ProductID
	}
	project := &entity.Project{
		ID:          uuid.New().String()[:32],
		Code:        code,
		Name:        req.Name,
		ProductID:   productID,
		Status:      entity.ProjectStatusPlanning,
		Phase:       "evt",
		Description: req.Description,
		ManagerID:   ownerID,
		StartDate:   req.PlannedStart,
		PlannedEnd:  req.PlannedEnd,
		Progress:    0,
		CreatedBy:   userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.projectRepo.Create(ctx, project); err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}

	// 创建默认阶段
	phases := []struct {
		Phase    string
		Name     string
		Sequence int
	}{
		{"evt", "工程验证", 1},
		{"dvt", "设计验证", 2},
		{"pvt", "产品验证", 3},
		{"mp", "量产", 4},
	}

	for _, p := range phases {
		phase := &entity.ProjectPhase{
			ID:        uuid.New().String()[:32],
			ProjectID: project.ID,
			Phase:     p.Phase,
			Name:      p.Name,
			Status:    "pending",
			Sequence:  p.Sequence,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := s.projectRepo.CreatePhase(ctx, phase); err != nil {
			return nil, fmt.Errorf("create phase: %w", err)
		}
	}

	return project, nil
}

// UpdateProject 更新项目
func (s *ProjectService) UpdateProject(ctx context.Context, id string, req *UpdateProjectRequest) (*entity.Project, error) {
	project, err := s.projectRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find project: %w", err)
	}

	if req.Name != "" {
		project.Name = req.Name
	}
	if req.Description != "" {
		project.Description = req.Description
	}
	if req.OwnerID != "" {
		project.ManagerID = req.OwnerID
	}
	if req.PlannedStart != nil {
		project.StartDate = req.PlannedStart
	}
	if req.PlannedEnd != nil {
		project.PlannedEnd = req.PlannedEnd
	}
	if req.CurrentPhase != "" {
		project.Phase = req.CurrentPhase
	}

	project.UpdatedAt = time.Now()

	if err := s.projectRepo.Update(ctx, project); err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}

	return project, nil
}

// DeleteProject 删除项目
func (s *ProjectService) DeleteProject(ctx context.Context, id string) error {
	return s.projectRepo.Delete(ctx, id)
}

// UpdateProjectStatus 更新项目状态
func (s *ProjectService) UpdateProjectStatus(ctx context.Context, id string, status string) (*entity.Project, error) {
	project, err := s.projectRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find project: %w", err)
	}

	project.Status = status
	if status == entity.ProjectStatusCompleted {
		now := time.Now()
		project.ActualEnd = &now
	}
	project.UpdatedAt = time.Now()

	if err := s.projectRepo.Update(ctx, project); err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}

	return project, nil
}

// ListPhases 获取项目阶段列表
func (s *ProjectService) ListPhases(ctx context.Context, projectID string) ([]entity.ProjectPhase, error) {
	return s.projectRepo.ListPhases(ctx, projectID)
}

// UpdatePhaseStatus 更新阶段状态
func (s *ProjectService) UpdatePhaseStatus(ctx context.Context, phaseID string, status string) (*entity.ProjectPhase, error) {
	phase, err := s.projectRepo.FindPhaseByID(ctx, phaseID)
	if err != nil {
		return nil, fmt.Errorf("find phase: %w", err)
	}

	phase.Status = status
	if status == "in_progress" {
		now := time.Now()
		phase.ActualStart = &now
	} else if status == "completed" {
		now := time.Now()
		phase.ActualEnd = &now
	}
	phase.UpdatedAt = time.Now()

	if err := s.projectRepo.UpdatePhase(ctx, phase); err != nil {
		return nil, fmt.Errorf("update phase: %w", err)
	}

	return phase, nil
}

// ============================================================
// 任务相关操作
// ============================================================

// ListTasks 获取任务列表
func (s *ProjectService) ListTasks(ctx context.Context, projectID string, page, pageSize int, filters map[string]interface{}) (*TaskListResult, error) {
	filters["project_id"] = projectID
	tasks, total, err := s.taskRepo.List(ctx, page, pageSize, filters)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	// 加载任务依赖信息
	if len(tasks) > 0 {
		taskIDs := make([]string, len(tasks))
		for i, t := range tasks {
			taskIDs[i] = t.ID
		}

		// 批量查询依赖
		deps, err := s.taskRepo.ListDependenciesByTaskIDs(ctx, taskIDs)
		if err == nil && len(deps) > 0 {
			// 收集所有被依赖的任务ID
			depTaskIDs := make([]string, 0, len(deps))
			for _, d := range deps {
				depTaskIDs = append(depTaskIDs, d.DependsOnID)
			}

			// 批量查询被依赖任务的状态
			statusMap, _ := s.taskRepo.FindStatusByIDs(ctx, depTaskIDs)

			// 填充 DependsOnStatus
			for i := range deps {
				if status, ok := statusMap[deps[i].DependsOnID]; ok {
					deps[i].DependsOnStatus = status
				}
			}

			// 按 task_id 分组并附加到任务
			depMap := make(map[string][]entity.TaskDependency)
			for _, d := range deps {
				depMap[d.TaskID] = append(depMap[d.TaskID], d)
			}
			for i := range tasks {
				if taskDeps, ok := depMap[tasks[i].ID]; ok {
					tasks[i].Dependencies = taskDeps
				}
			}
		}

		// 补偿逻辑：自动启动所有前置已完成的 pending 任务
		for i := range tasks {
			if tasks[i].Status != "pending" || len(tasks[i].Dependencies) == 0 {
				continue
			}

			allCompleted := true
			for _, dep := range tasks[i].Dependencies {
				if dep.DependsOnStatus != "completed" && dep.DependsOnStatus != "confirmed" {
					allCompleted = false
					break
				}
			}

			if allCompleted {
				now := time.Now()
				tasks[i].Status = "in_progress"
				tasks[i].ActualStart = &now
				_ = s.taskRepo.Update(ctx, &tasks[i])
			}
		}
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &TaskListResult{
		Items:      tasks,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// GetTask 获取任务详情
func (s *ProjectService) GetTask(ctx context.Context, id string) (*entity.Task, error) {
	task, err := s.taskRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find task: %w", err)
	}
	return task, nil
}

// CreateTask 创建任务
func (s *ProjectService) CreateTask(ctx context.Context, projectID string, userID string, req *CreateTaskRequest) (*entity.Task, error) {
	// 验证项目存在
	_, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	// 生成任务编码
	code, err := s.taskRepo.GenerateCode(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("generate code: %w", err)
	}

	taskType := req.TaskType
	if taskType == "" {
		taskType = entity.TaskTypeTask
	}

	priority := req.Priority
	if priority == "" {
		priority = entity.TaskPriorityMedium
	}

	// 计算层级
	level := 0
	if req.ParentTaskID != "" {
		parent, err := s.taskRepo.FindByID(ctx, req.ParentTaskID)
		if err == nil {
			level = parent.Level + 1
		}
	}

	now := time.Now()
	var phaseID *string
	if req.PhaseID != "" {
		phaseID = &req.PhaseID
	}
	var assigneeID, reviewerID *string
	if req.AssigneeID != "" {
		assigneeID = &req.AssigneeID
	}
	if req.ReviewerID != "" {
		reviewerID = &req.ReviewerID
	}

	var parentTaskID *string
	if req.ParentTaskID != "" {
		parentTaskID = &req.ParentTaskID
	}

	// Use DueDate if provided, otherwise fall back to PlannedEnd
	dueDate := req.DueDate
	if dueDate == nil {
		dueDate = req.PlannedEnd
	}

	task := &entity.Task{
		ID:             uuid.New().String()[:32],
		ProjectID:      projectID,
		PhaseID:        phaseID,
		ParentTaskID:   parentTaskID,
		Code:           code,
		Name:           req.Name,
		Description:    req.Description,
		TaskType:       taskType,
		Status:         entity.TaskStatusPending,
		Priority:       priority,
		AssigneeID:     assigneeID,
		ReviewerID:     reviewerID,
		StartDate:      req.PlannedStart,
		DueDate:        dueDate,
		Progress:       0,
		EstimatedHours: req.EstimatedHours,
		Level:          level,
		CreatedBy:      userID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	return task, nil
}

// UpdateTask 更新任务
func (s *ProjectService) UpdateTask(ctx context.Context, id string, req *UpdateTaskRequest) (*entity.Task, error) {
	task, err := s.taskRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find task: %w", err)
	}

	if req.Name != "" {
		task.Name = req.Name
	}
	if req.Description != "" {
		task.Description = req.Description
	}
	if req.Priority != "" {
		task.Priority = req.Priority
	}
	if req.AssigneeID != "" {
		task.AssigneeID = &req.AssigneeID
	}
	if req.ReviewerID != "" {
		task.ReviewerID = &req.ReviewerID
	}
	if req.PlannedStart != nil {
		task.StartDate = req.PlannedStart
	}
	if req.PlannedEnd != nil {
		task.DueDate = req.PlannedEnd
	}
	if req.DueDate != nil {
		task.DueDate = req.DueDate
	}
	if req.EstimatedHours > 0 {
		task.EstimatedHours = req.EstimatedHours
	}
	if req.ActualHours > 0 {
		task.ActualHours = req.ActualHours
	}
	if req.Progress >= 0 && req.Progress <= 100 {
		task.Progress = req.Progress
	}

	task.UpdatedAt = time.Now()

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	return task, nil
}

// DeleteTask 删除任务
func (s *ProjectService) DeleteTask(ctx context.Context, id string) error {
	return s.taskRepo.Delete(ctx, id)
}

// UpdateTaskStatus 更新任务状态
func (s *ProjectService) UpdateTaskStatus(ctx context.Context, id string, status string) (*entity.Task, error) {
	task, err := s.taskRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find task: %w", err)
	}

	task.Status = status
	now := time.Now()

	if status == entity.TaskStatusInProgress && task.ActualStart == nil {
		task.ActualStart = &now
	} else if status == entity.TaskStatusCompleted {
		task.CompletedAt = &now
		task.Progress = 100
	}

	task.UpdatedAt = now

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	// 更新项目进度
	go s.updateProjectProgress(context.Background(), task.ProjectID)

	// SSE: 通知前端任务状态变更
	go sse.PublishTaskUpdate(task.ProjectID, task.ID, "status_change")

	return task, nil
}

// updateProjectProgress 更新项目整体进度
func (s *ProjectService) updateProjectProgress(ctx context.Context, projectID string) {
	progress, err := s.taskRepo.CalculateProjectProgress(ctx, projectID)
	if err != nil {
		return
	}

	s.projectRepo.UpdateProgress(ctx, projectID, progress)
}

// ListSubTasks 获取子任务列表
func (s *ProjectService) ListSubTasks(ctx context.Context, parentID string) ([]entity.Task, error) {
	return s.taskRepo.ListByParent(ctx, parentID)
}

// AddTaskComment 添加任务评论
func (s *ProjectService) AddTaskComment(ctx context.Context, taskID string, userID string, content string) (*entity.TaskComment, error) {
	comment := &entity.TaskComment{
		ID:        uuid.New().String()[:32],
		TaskID:    taskID,
		UserID:    userID,
		Content:   content,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.taskRepo.AddComment(ctx, comment); err != nil {
		return nil, fmt.Errorf("add comment: %w", err)
	}

	return comment, nil
}

// ListTaskComments 获取任务评论列表
func (s *ProjectService) ListTaskComments(ctx context.Context, taskID string) ([]entity.TaskComment, error) {
	return s.taskRepo.ListComments(ctx, taskID)
}

// AddTaskDependency 添加任务依赖
func (s *ProjectService) AddTaskDependency(ctx context.Context, taskID, dependsOnID, dependencyType string, lagDays int) (*entity.TaskDependency, error) {
	dep := &entity.TaskDependency{
		ID:              uuid.New().String()[:32],
		TaskID:          taskID,
		DependsOnID: dependsOnID,
		DependencyType:  dependencyType,
		LagDays:         lagDays,
		CreatedAt:       time.Now(),
	}

	if err := s.taskRepo.AddDependency(ctx, dep); err != nil {
		return nil, fmt.Errorf("add dependency: %w", err)
	}

	return dep, nil
}

// RemoveTaskDependency 移除任务依赖
func (s *ProjectService) RemoveTaskDependency(ctx context.Context, id string) error {
	return s.taskRepo.RemoveDependency(ctx, id)
}

// ListTaskDependencies 获取任务依赖列表
func (s *ProjectService) ListTaskDependencies(ctx context.Context, taskID string) ([]entity.TaskDependency, error) {
	return s.taskRepo.ListDependencies(ctx, taskID)
}

// GetMyTasks 获取我的任务
func (s *ProjectService) GetMyTasks(ctx context.Context, userID string, page, pageSize int, filters map[string]interface{}) (*TaskListResult, error) {
	tasks, total, err := s.taskRepo.ListByAssigneeWithPaging(ctx, userID, page, pageSize, filters)
	if err != nil {
		return nil, fmt.Errorf("list my tasks: %w", err)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &TaskListResult{
		Items:      tasks,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// GetOverdueTasks 获取逾期任务
func (s *ProjectService) GetOverdueTasks(ctx context.Context, projectID string) ([]entity.Task, error) {
	return s.taskRepo.ListOverdue(ctx, projectID)
}

// CompleteMyTask 工程师完成任务（含表单提交）
func (s *ProjectService) CompleteMyTask(ctx context.Context, taskID, userID string, formData map[string]interface{}) error {
	// 1. 查找任务
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("任务不存在")
	}

	// 2. 验证 assignee
	if task.AssigneeID == nil || *task.AssigneeID != userID {
		return fmt.Errorf("只有任务负责人才能完成任务")
	}

	// 3. 验证状态
	if task.Status != entity.TaskStatusInProgress {
		return fmt.Errorf("只有进行中的任务才能完成，当前状态: %s", task.Status)
	}

	// 4. 检查表单
	if s.taskFormRepo != nil {
		form, err := s.taskFormRepo.FindByTaskID(ctx, taskID)
		if err != nil {
			return fmt.Errorf("查询表单失败: %w", err)
		}

		if form != nil {
			// 有表单，必须提交 form_data
			if formData == nil {
				return fmt.Errorf("此任务需要填写表单才能完成")
			}

			// 验证 required 字段
			var fields []struct {
				Key      string `json:"key"`
				Label    string `json:"label"`
				Required bool   `json:"required"`
			}
			if err := json.Unmarshal(form.Fields, &fields); err == nil {
				for _, f := range fields {
					if f.Required {
						val, ok := formData[f.Key]
						if !ok || val == nil || val == "" {
							return fmt.Errorf("必填字段 [%s] 未填写", f.Label)
						}
					}
				}
			}

			// 计算版本号
			version := 1
			latestSubmission, _ := s.taskFormRepo.FindLatestSubmission(ctx, taskID)
			if latestSubmission != nil {
				version = latestSubmission.Version + 1
			}

			// 创建提交记录
			submission := &entity.TaskFormSubmission{
				ID:          uuid.New().String()[:32],
				FormID:      form.ID,
				TaskID:      taskID,
				Data:        entity.JSONB(formData),
				SubmittedBy: userID,
				SubmittedAt: time.Now(),
				Version:     version,
			}
			if err := s.taskFormRepo.CreateSubmission(ctx, submission); err != nil {
				return fmt.Errorf("保存表单提交失败: %w", err)
			}
		}
	}

	// 5. 更新任务状态
	now := time.Now()
	task.Status = entity.TaskStatusCompleted
	task.CompletedAt = &now
	task.Progress = 100
	task.UpdatedAt = now

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 6. 更新项目进度
	go s.updateProjectProgress(context.Background(), task.ProjectID)

	// SSE: 通知前端任务完成
	go sse.PublishTaskUpdate(task.ProjectID, task.ID, "task_completed")

	return nil
}

// ConfirmTask 项目经理确认任务
func (s *ProjectService) ConfirmTask(ctx context.Context, projectID, taskID, userID string) error {
	// 1. 查找项目，验证 manager_id
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return fmt.Errorf("项目不存在")
	}
	if project.ManagerID != userID {
		return fmt.Errorf("只有项目经理才能确认任务")
	}

	// 2. 查找任务，验证状态
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("任务不存在")
	}
	if task.Status != entity.TaskStatusCompleted {
		return fmt.Errorf("只有已完成的任务才能确认，当前状态: %s", task.Status)
	}

	// 3. 更新状态
	task.Status = entity.TaskStatusConfirmed
	task.UpdatedAt = time.Now()

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	// SSE: 通知前端任务确认
	go sse.PublishTaskUpdate(task.ProjectID, task.ID, "task_confirmed")

	return nil
}

// RejectTask 项目经理驳回任务
func (s *ProjectService) RejectTask(ctx context.Context, projectID, taskID, userID, reason string) error {
	// 1. 查找项目，验证 manager_id
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return fmt.Errorf("项目不存在")
	}
	if project.ManagerID != userID {
		return fmt.Errorf("只有项目经理才能驳回任务")
	}

	// 2. 查找任务，验证状态
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("任务不存在")
	}
	if task.Status != entity.TaskStatusCompleted {
		return fmt.Errorf("只有已完成的任务才能驳回，当前状态: %s", task.Status)
	}

	// 3. 更新状态回 in_progress
	task.Status = entity.TaskStatusInProgress
	task.CompletedAt = nil
	task.Progress = 50 // 驳回后重置进度
	task.UpdatedAt = time.Now()

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 4. 添加驳回评论
	if reason != "" {
		comment := &entity.TaskComment{
			ID:        uuid.New().String()[:32],
			TaskID:    taskID,
			UserID:    userID,
			Content:   fmt.Sprintf("[驳回] %s", reason),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		s.taskRepo.AddComment(ctx, comment)
	}

	// 5. 更新项目进度
	go s.updateProjectProgress(context.Background(), task.ProjectID)

	// SSE: 通知前端任务驳回
	go sse.PublishTaskUpdate(task.ProjectID, task.ID, "task_rejected")

	return nil
}
