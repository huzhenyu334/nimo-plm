package repository

import (
	"context"
	"errors"

	"github.com/bitfantasy/nimo/internal/plm/entity"
	"gorm.io/gorm"
)

// TaskFormRepository 任务表单仓库
type TaskFormRepository struct {
	db *gorm.DB
}

// NewTaskFormRepository 创建任务表单仓库
func NewTaskFormRepository(db *gorm.DB) *TaskFormRepository {
	return &TaskFormRepository{db: db}
}

// FindByTaskID 根据任务ID查找表单定义
func (r *TaskFormRepository) FindByTaskID(ctx context.Context, taskID string) (*entity.TaskForm, error) {
	var form entity.TaskForm
	err := r.db.WithContext(ctx).Where("task_id = ?", taskID).First(&form).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 没有表单不算错误
		}
		return nil, err
	}
	return &form, nil
}

// Create 创建表单定义
func (r *TaskFormRepository) Create(ctx context.Context, form *entity.TaskForm) error {
	return r.db.WithContext(ctx).Create(form).Error
}

// Update 更新表单定义
func (r *TaskFormRepository) Update(ctx context.Context, form *entity.TaskForm) error {
	return r.db.WithContext(ctx).Save(form).Error
}

// Delete 删除表单定义
func (r *TaskFormRepository) Delete(ctx context.Context, taskID string) error {
	return r.db.WithContext(ctx).Where("task_id = ?", taskID).Delete(&entity.TaskForm{}).Error
}

// CreateSubmission 创建表单提交记录
func (r *TaskFormRepository) CreateSubmission(ctx context.Context, submission *entity.TaskFormSubmission) error {
	return r.db.WithContext(ctx).Create(submission).Error
}

// FindLatestSubmission 获取最新的表单提交记录
func (r *TaskFormRepository) FindLatestSubmission(ctx context.Context, taskID string) (*entity.TaskFormSubmission, error) {
	var submission entity.TaskFormSubmission
	err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Order("version DESC").
		First(&submission).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &submission, nil
}

// UpsertDraftSubmission 保存草稿（version=0）
func (r *TaskFormRepository) UpsertDraftSubmission(ctx context.Context, submission *entity.TaskFormSubmission) error {
	var existing entity.TaskFormSubmission
	err := r.db.WithContext(ctx).
		Where("task_id = ? AND version = 0", submission.TaskID).
		First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			submission.Version = 0
			return r.db.WithContext(ctx).Create(submission).Error
		}
		return err
	}
	existing.Data = submission.Data
	existing.SubmittedAt = submission.SubmittedAt
	return r.db.WithContext(ctx).Save(&existing).Error
}

// FindDraftSubmission 获取草稿（version=0）
func (r *TaskFormRepository) FindDraftSubmission(ctx context.Context, taskID string) (*entity.TaskFormSubmission, error) {
	var submission entity.TaskFormSubmission
	err := r.db.WithContext(ctx).
		Where("task_id = ? AND version = 0", taskID).
		First(&submission).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &submission, nil
}

// FindTemplateFormsByTemplateID 根据模板ID查找模板表单
func (r *TaskFormRepository) FindTemplateFormsByTemplateID(ctx context.Context, templateID string) ([]entity.TemplateTaskForm, error) {
	var forms []entity.TemplateTaskForm
	err := r.db.WithContext(ctx).Where("template_id = ?", templateID).Find(&forms).Error
	return forms, err
}

// CreateTemplateForm 创建模板表单
func (r *TaskFormRepository) CreateTemplateForm(ctx context.Context, form *entity.TemplateTaskForm) error {
	return r.db.WithContext(ctx).Create(form).Error
}

// UpsertTemplateForm 创建或更新模板任务表单
func (r *TaskFormRepository) UpsertTemplateForm(ctx context.Context, form *entity.TemplateTaskForm) error {
	var existing entity.TemplateTaskForm
	err := r.db.WithContext(ctx).
		Where("template_id = ? AND task_code = ?", form.TemplateID, form.TaskCode).
		First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.db.WithContext(ctx).Create(form).Error
		}
		return err
	}
	existing.Name = form.Name
	existing.Fields = form.Fields
	existing.UpdatedAt = form.UpdatedAt
	return r.db.WithContext(ctx).Save(&existing).Error
}

// DeleteTemplateForm 删除模板任务表单
func (r *TaskFormRepository) DeleteTemplateForm(ctx context.Context, templateID, taskCode string) error {
	return r.db.WithContext(ctx).
		Where("template_id = ? AND task_code = ?", templateID, taskCode).
		Delete(&entity.TemplateTaskForm{}).Error
}
