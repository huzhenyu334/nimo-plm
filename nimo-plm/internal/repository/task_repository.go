package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bitfantasy/nimo-plm/internal/model/entity"
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
		Order("due_date ASC, priority DESC").
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

// ListByAssigneeWithPaging 带分页获取指派任务
func (r *TaskRepository) ListByAssigneeWithPaging(ctx context.Context, userID string, page, pageSize int, filters map[string]interface{}) ([]entity.Task, int64, error) {
	var tasks []entity.Task
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.Task{}).Where("assignee_id = ?", userID)

	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where("status = ?", status)
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
		Order("due_date ASC, priority DESC").
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
