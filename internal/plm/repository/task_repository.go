package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bitfantasy/nimo/internal/plm/entity"
	"gorm.io/gorm"
)

// TaskRepository 任务仓库
type TaskRepository struct {
	db *gorm.DB
}

// NewTaskRepository 创建任务仓库
func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// FindByID 根据ID查找任务
func (r *TaskRepository) FindByID(ctx context.Context, id string) (*entity.Task, error) {
	var task entity.Task
	err := r.db.WithContext(ctx).
		Preload("Project").
		Preload("Phase").
		Preload("Assignee").
		Preload("ParentTask").
		Where("id = ?", id).
		First(&task).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &task, nil
}

// Create 创建任务
func (r *TaskRepository) Create(ctx context.Context, task *entity.Task) error {
	return r.db.WithContext(ctx).Create(task).Error
}

// Update 更新任务
func (r *TaskRepository) Update(ctx context.Context, task *entity.Task) error {
	return r.db.WithContext(ctx).Save(task).Error
}

// Delete 删除任务
func (r *TaskRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&entity.Task{}).Error
}

// ListByProject 获取项目任务列表
func (r *TaskRepository) ListByProject(ctx context.Context, projectID string, filters map[string]interface{}) ([]entity.Task, error) {
	var tasks []entity.Task

	query := r.db.WithContext(ctx).
		Where("project_id = ?", projectID)

	if phaseID, ok := filters["phase_id"].(string); ok && phaseID != "" {
		query = query.Where("phase_id = ?", phaseID)
	}
	if phase, ok := filters["phase"].(string); ok && phase != "" {
		query = query.Joins("JOIN project_phases ON tasks.phase_id = project_phases.id").
			Where("project_phases.phase = ?", phase)
	}
	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where("status = ?", status)
	}
	if assigneeID, ok := filters["assignee_id"].(string); ok && assigneeID != "" {
		query = query.Where("assignee_id = ?", assigneeID)
	}
	if overdueOnly, ok := filters["overdue_only"].(bool); ok && overdueOnly {
		query = query.Where("due_date < ? AND status NOT IN ?", time.Now(), []string{entity.TaskStatusCompleted, entity.TaskStatusCancelled})
	}

	err := query.
		Preload("Assignee").
		Preload("Creator").
		Order("level ASC, sequence ASC").
		Find(&tasks).Error

	return tasks, err
}

// ListByAssignee 获取指派给用户的任务
func (r *TaskRepository) ListByAssignee(ctx context.Context, userID string, filters map[string]interface{}) ([]entity.Task, error) {
	var tasks []entity.Task

	query := r.db.WithContext(ctx).
		Where("assignee_id = ?", userID)

	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where("status = ?", status)
	}
	if overdueOnly, ok := filters["overdue_only"].(bool); ok && overdueOnly {
		query = query.Where("due_date < ? AND status NOT IN ?", time.Now(), []string{entity.TaskStatusCompleted, entity.TaskStatusCancelled})
	}

	err := query.
		Preload("Project").
		Preload("Assignee").
		Order("created_at DESC").
		Find(&tasks).Error

	return tasks, err
}

// GetSubTasks 获取子任务
func (r *TaskRepository) GetSubTasks(ctx context.Context, parentTaskID string) ([]entity.Task, error) {
	var tasks []entity.Task
	err := r.db.WithContext(ctx).
		Where("parent_task_id = ?", parentTaskID).
		Order("sequence ASC").
		Find(&tasks).Error
	return tasks, err
}

// CreateBatch 批量创建任务
func (r *TaskRepository) CreateBatch(ctx context.Context, tasks []entity.Task) error {
	return r.db.WithContext(ctx).Create(&tasks).Error
}

// GetTaskTemplates 获取任务模板
func (r *TaskRepository) GetTaskTemplates(ctx context.Context, templateName, phase string) ([]TaskTemplate, error) {
	var templates []TaskTemplate
	query := r.db.WithContext(ctx).
		Table("task_templates").
		Where("template_name = ?", templateName)
	
	if phase != "" {
		query = query.Where("phase = ?", phase)
	}

	err := query.Order("phase ASC, sequence ASC").Find(&templates).Error
	return templates, err
}

// TaskTemplate 任务模板
type TaskTemplate struct {
	ID                 string `gorm:"column:id"`
	TemplateName       string `gorm:"column:template_name"`
	Phase              string `gorm:"column:phase"`
	TaskCode           string `gorm:"column:task_code"`
	TaskName           string `gorm:"column:task_name"`
	TaskType           string `gorm:"column:task_type"`
	ParentCode         string `gorm:"column:parent_code"`
	Sequence           int    `gorm:"column:sequence"`
	Level              int    `gorm:"column:level"`
	DefaultDurationDays int   `gorm:"column:default_duration_days"`
	Description        string `gorm:"column:description"`
	IsRequired         bool   `gorm:"column:is_required"`
}

// UpdateProgress 更新任务进度
func (r *TaskRepository) UpdateProgress(ctx context.Context, taskID string, progress int) error {
	updates := map[string]interface{}{
		"progress":   progress,
		"updated_at": time.Now(),
	}

	if progress >= 100 {
		updates["status"] = entity.TaskStatusCompleted
		now := time.Now()
		updates["actual_end"] = now
	}

	return r.db.WithContext(ctx).
		Model(&entity.Task{}).
		Where("id = ?", taskID).
		Updates(updates).Error
}

// List 获取任务列表（分页）
func (r *TaskRepository) List(ctx context.Context, page, pageSize int, filters map[string]interface{}) ([]entity.Task, int64, error) {
	var tasks []entity.Task
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.Task{})

	if projectID, ok := filters["project_id"].(string); ok && projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}
	if phaseID, ok := filters["phase_id"].(string); ok && phaseID != "" {
		query = query.Where("phase_id = ?", phaseID)
	}
	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where("status = ?", status)
	}
	if assigneeID, ok := filters["assignee_id"].(string); ok && assigneeID != "" {
		query = query.Where("assignee_id = ?", assigneeID)
	}
	if priority, ok := filters["priority"].(string); ok && priority != "" {
		query = query.Where("priority = ?", priority)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.
		Preload("Assignee").
		Preload("Phase").
		Order("level ASC, sequence ASC").
		Offset(offset).
		Limit(pageSize).
		Find(&tasks).Error

	return tasks, total, err
}

// GenerateCode 生成任务编码
func (r *TaskRepository) GenerateCode(ctx context.Context, projectID string) (string, error) {
	var count int64
	r.db.WithContext(ctx).Model(&entity.Task{}).Where("project_id = ?", projectID).Count(&count)
	return fmt.Sprintf("T-%04d", count+1), nil
}

// ListByParent 获取子任务列表
func (r *TaskRepository) ListByParent(ctx context.Context, parentID string) ([]entity.Task, error) {
	var tasks []entity.Task
	err := r.db.WithContext(ctx).
		Where("parent_task_id = ?", parentID).
		Preload("Assignee").
		Order("sequence ASC").
		Find(&tasks).Error
	return tasks, err
}

// AddComment 添加评论
func (r *TaskRepository) AddComment(ctx context.Context, comment *entity.TaskComment) error {
	return r.db.WithContext(ctx).Create(comment).Error
}

// ListComments 获取评论列表
func (r *TaskRepository) ListComments(ctx context.Context, taskID string) ([]entity.TaskComment, error) {
	var comments []entity.TaskComment
	err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Preload("User").
		Order("created_at DESC").
		Find(&comments).Error
	return comments, err
}

// AddDependency 添加依赖
func (r *TaskRepository) AddDependency(ctx context.Context, dep *entity.TaskDependency) error {
	return r.db.WithContext(ctx).Create(dep).Error
}

// RemoveDependency 移除依赖
func (r *TaskRepository) RemoveDependency(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.TaskDependency{}, "id = ?", id).Error
}

// ListDependencies 获取依赖列表
func (r *TaskRepository) ListDependencies(ctx context.Context, taskID string) ([]entity.TaskDependency, error) {
	var deps []entity.TaskDependency
	err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Find(&deps).Error
	return deps, err
}

// ListDependenciesByTaskIDs 批量获取多个任务的依赖
func (r *TaskRepository) ListDependenciesByTaskIDs(ctx context.Context, taskIDs []string) ([]entity.TaskDependency, error) {
	var deps []entity.TaskDependency
	if len(taskIDs) == 0 {
		return deps, nil
	}
	err := r.db.WithContext(ctx).
		Where("task_id IN (?)", taskIDs).
		Find(&deps).Error
	return deps, err
}

// FindStatusByIDs 批量查询任务状态
func (r *TaskRepository) FindStatusByIDs(ctx context.Context, ids []string) (map[string]string, error) {
	if len(ids) == 0 {
		return make(map[string]string), nil
	}
	var results []struct {
		ID     string `gorm:"column:id"`
		Status string `gorm:"column:status"`
	}
	err := r.db.WithContext(ctx).
		Model(&entity.Task{}).
		Select("id, status").
		Where("id IN (?)", ids).
		Find(&results).Error
	if err != nil {
		return nil, err
	}
	statusMap := make(map[string]string, len(results))
	for _, r := range results {
		statusMap[r.ID] = r.Status
	}
	return statusMap, nil
}

// ListByAssigneeWithPaging 带分页获取指派任务
func (r *TaskRepository) ListByAssigneeWithPaging(ctx context.Context, userID string, page, pageSize int, filters map[string]interface{}) ([]entity.Task, int64, error) {
	var tasks []entity.Task
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.Task{}).Where("assignee_id = ?", userID)

	// 如果没有指定status筛选，默认排除pending任务（前置依赖未完成的不显示）
	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where("status = ?", status)
	} else {
		query = query.Where("status NOT IN ?", []string{"pending", "cancelled"})
	}
	if priority, ok := filters["priority"].(string); ok && priority != "" {
		query = query.Where("priority = ?", priority)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.
		Preload("Project").
		Preload("Assignee").
		Preload("Creator").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&tasks).Error

	return tasks, total, err
}

// ListOverdue 获取逾期任务
func (r *TaskRepository) ListOverdue(ctx context.Context, projectID string) ([]entity.Task, error) {
	var tasks []entity.Task
	query := r.db.WithContext(ctx).
		Where("due_date < ? AND status NOT IN ?", time.Now(), []string{entity.TaskStatusCompleted, entity.TaskStatusCancelled})
	
	if projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}

	err := query.
		Preload("Assignee").
		Preload("Project").
		Order("due_date ASC").
		Find(&tasks).Error
	return tasks, err
}

// CalculateProjectProgress 计算项目整体进度
func (r *TaskRepository) CalculateProjectProgress(ctx context.Context, projectID string) (int, error) {
	var result struct {
		TotalTasks int64
		TotalProgress int64
	}

	err := r.db.WithContext(ctx).
		Model(&entity.Task{}).
		Select("COUNT(*) as total_tasks, COALESCE(SUM(progress), 0) as total_progress").
		Where("project_id = ? AND parent_task_id IS NULL OR parent_task_id = ''", projectID).
		Scan(&result).Error
	
	if err != nil {
		return 0, err
	}

	if result.TotalTasks == 0 {
		return 0, nil
	}

	return int(result.TotalProgress / result.TotalTasks), nil
}

// ListDependentTasks 查询依赖指定任务的所有下游任务（即 depends_on_task_id = taskID 的任务）
func (r *TaskRepository) ListDependentTasks(ctx context.Context, taskID string) ([]entity.Task, error) {
	var tasks []entity.Task
	err := r.db.WithContext(ctx).
		Where("id IN (SELECT task_id FROM task_dependencies WHERE depends_on_task_id = ?)", taskID).
		Find(&tasks).Error
	return tasks, err
}

// FindByProjectAndCode 按项目ID + 任务Code查找任务
func (r *TaskRepository) FindByProjectAndCode(ctx context.Context, projectID, code string) (*entity.Task, error) {
	var task entity.Task
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND code = ?", projectID, code).
		First(&task).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &task, nil
}

// UpdateAssigneeByRole 按角色批量更新任务的 assignee_id（大小写不敏感匹配）
func (r *TaskRepository) UpdateAssigneeByRole(ctx context.Context, projectID, role, userID string) error {
	return r.db.WithContext(ctx).
		Model(&entity.Task{}).
		Where("project_id = ? AND LOWER(default_assignee_role) = LOWER(?)", projectID, role).
		Update("assignee_id", userID).Error
}

// FindRoleAssignment 查询项目角色分配（按角色代码，大小写不敏感）
func (r *TaskRepository) FindRoleAssignment(ctx context.Context, projectID, roleCode string) (*entity.ProjectRoleAssignment, error) {
	var assignment entity.ProjectRoleAssignment
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND LOWER(role_code) = LOWER(?)", projectID, roleCode).
		First(&assignment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &assignment, nil
}

// UpsertRoleAssignment 插入或更新项目角色分配
func (r *TaskRepository) UpsertRoleAssignment(ctx context.Context, assignment *entity.ProjectRoleAssignment) error {
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO project_role_assignments (id, project_id, phase, role_code, user_id, assigned_by, assigned_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW())
		ON CONFLICT (project_id, phase, role_code) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			assigned_by = EXCLUDED.assigned_by,
			assigned_at = NOW()
	`, assignment.ID, assignment.ProjectID, assignment.Phase, assignment.RoleCode, assignment.UserID, assignment.AssignedBy).Error
}
