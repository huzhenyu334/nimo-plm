package service

import (
	"context"
	"fmt"
	"time"

	"github.com/bitfantasy/nimo-plm/internal/model/entity"
	"github.com/bitfantasy/nimo-plm/internal/repository"
	"github.com/google/uuid"
)

// ProjectService 项目服务
type ProjectService struct {
	projectRepo *repository.ProjectRepository
	taskRepo    *repository.TaskRepository
	productRepo *repository.ProductRepository
	feishuSvc   *FeishuIntegrationService
}

// NewProjectService 创建项目服务
func NewProjectService(
	projectRepo *repository.ProjectRepository,
	taskRepo *repository.TaskRepository,
	productRepo *repository.ProductRepository,
	feishuSvc *FeishuIntegrationService,
) *ProjectService {
	return &ProjectService{
		projectRepo: projectRepo,
		taskRepo:    taskRepo,
		productRepo: productRepo,
		feishuSvc:   feishuSvc,
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
	project := &entity.Project{
		ID:           uuid.New().String()[:32],
		Code:         code,
		Name:         req.Name,
		ProductID:    req.ProductID,
		Status:       entity.ProjectStatusPlanning,
		CurrentPhase: "evt",
		Description:  req.Description,
		OwnerID:      ownerID,
		PlannedStart: req.PlannedStart,
		PlannedEnd:   req.PlannedEnd,
		Progress:     0,
		CreatedBy:    userID,
		CreatedAt:    now,
		UpdatedAt:    now,
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
		project.OwnerID = req.OwnerID
	}
	if req.PlannedStart != nil {
		project.PlannedStart = req.PlannedStart
	}
	if req.PlannedEnd != nil {
		project.PlannedEnd = req.PlannedEnd
	}
	if req.CurrentPhase != "" {
		project.CurrentPhase = req.CurrentPhase
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
	task := &entity.Task{
		ID:             uuid.New().String()[:32],
		ProjectID:      projectID,
		PhaseID:        req.PhaseID,
		ParentTaskID:   req.ParentTaskID,
		Code:           code,
		Name:           req.Name,
		Description:    req.Description,
		TaskType:       taskType,
		Status:         entity.TaskStatusPending,
		Priority:       priority,
		AssigneeID:     req.AssigneeID,
		ReviewerID:     req.ReviewerID,
		PlannedStart:   req.PlannedStart,
		PlannedEnd:     req.PlannedEnd,
		DueDate:        req.DueDate,
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
		task.AssigneeID = req.AssigneeID
	}
	if req.ReviewerID != "" {
		task.ReviewerID = req.ReviewerID
	}
	if req.PlannedStart != nil {
		task.PlannedStart = req.PlannedStart
	}
	if req.PlannedEnd != nil {
		task.PlannedEnd = req.PlannedEnd
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
		task.ActualEnd = &now
		task.Progress = 100
	}

	task.UpdatedAt = now

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	// 更新项目进度
	go s.updateProjectProgress(context.Background(), task.ProjectID)

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
		DependsOnTaskID: dependsOnID,
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
