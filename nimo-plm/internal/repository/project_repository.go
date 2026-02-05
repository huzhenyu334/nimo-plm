package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bitfantasy/nimo-plm/internal/model/entity"
	"gorm.io/gorm"
)

// ProjectRepository 项目仓库
type ProjectRepository struct {
	db *gorm.DB
}

// NewProjectRepository 创建项目仓库
func NewProjectRepository(db *gorm.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// FindByID 根据ID查找项目
func (r *ProjectRepository) FindByID(ctx context.Context, id string) (*entity.Project, error) {
	var project entity.Project
	err := r.db.WithContext(ctx).
		Preload("Product").
		Preload("Owner").
		Preload("Phases", func(db *gorm.DB) *gorm.DB {
			return db.Order("sequence ASC")
		}).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&project).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &project, nil
}

// Create 创建项目
func (r *ProjectRepository) Create(ctx context.Context, project *entity.Project) error {
	return r.db.WithContext(ctx).Create(project).Error
}

// Update 更新项目
func (r *ProjectRepository) Update(ctx context.Context, project *entity.Project) error {
	return r.db.WithContext(ctx).Save(project).Error
}

// Delete 软删除项目
func (r *ProjectRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Model(&entity.Project{}).
		Where("id = ?", id).
		Update("deleted_at", time.Now()).Error
}

// List 获取项目列表
func (r *ProjectRepository) List(ctx context.Context, page, pageSize int, filters map[string]interface{}) ([]entity.Project, int64, error) {
	var projects []entity.Project
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.Project{}).Where("deleted_at IS NULL")

	if keyword, ok := filters["keyword"].(string); ok && keyword != "" {
		query = query.Where("name ILIKE ? OR code ILIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if productID, ok := filters["product_id"].(string); ok && productID != "" {
		query = query.Where("product_id = ?", productID)
	}
	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where("status = ?", status)
	}
	if ownerID, ok := filters["owner_id"].(string); ok && ownerID != "" {
		query = query.Where("owner_id = ?", ownerID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.
		Preload("Product").
		Preload("Owner").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&projects).Error

	return projects, total, err
}

// GenerateCode 生成项目编码
func (r *ProjectRepository) GenerateCode(ctx context.Context) (string, error) {
	var seq int64
	err := r.db.WithContext(ctx).Raw("SELECT nextval('project_code_seq')").Scan(&seq).Error
	if err != nil {
		return "", err
	}
	year := time.Now().Year()
	return fmt.Sprintf("PROJ-%d-%04d", year, seq), nil
}

// CreatePhase 创建项目阶段
func (r *ProjectRepository) CreatePhase(ctx context.Context, phase *entity.ProjectPhase) error {
	return r.db.WithContext(ctx).Create(phase).Error
}

// GetPhases 获取项目阶段列表
func (r *ProjectRepository) GetPhases(ctx context.Context, projectID string) ([]entity.ProjectPhase, error) {
	var phases []entity.ProjectPhase
	err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("sequence ASC").
		Find(&phases).Error
	return phases, err
}

// GetTaskStats 获取项目任务统计
func (r *ProjectRepository) GetTaskStats(ctx context.Context, projectID string) (total, completed, inProgress, pending, blocked int64, err error) {
	err = r.db.WithContext(ctx).
		Model(&entity.Task{}).
		Where("project_id = ?", projectID).
		Count(&total).Error
	if err != nil {
		return
	}

	r.db.WithContext(ctx).Model(&entity.Task{}).Where("project_id = ? AND status = ?", projectID, entity.TaskStatusCompleted).Count(&completed)
	r.db.WithContext(ctx).Model(&entity.Task{}).Where("project_id = ? AND status = ?", projectID, entity.TaskStatusInProgress).Count(&inProgress)
	r.db.WithContext(ctx).Model(&entity.Task{}).Where("project_id = ? AND status = ?", projectID, entity.TaskStatusPending).Count(&pending)
	r.db.WithContext(ctx).Model(&entity.Task{}).Where("project_id = ? AND status = ?", projectID, entity.TaskStatusBlocked).Count(&blocked)
	
	return
}

// ListPhases 获取项目阶段列表
func (r *ProjectRepository) ListPhases(ctx context.Context, projectID string) ([]entity.ProjectPhase, error) {
	var phases []entity.ProjectPhase
	err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("sequence ASC").
		Find(&phases).Error
	return phases, err
}

// FindPhaseByID 根据ID查找阶段
func (r *ProjectRepository) FindPhaseByID(ctx context.Context, phaseID string) (*entity.ProjectPhase, error) {
	var phase entity.ProjectPhase
	err := r.db.WithContext(ctx).
		Where("id = ?", phaseID).
		First(&phase).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &phase, nil
}

// UpdatePhase 更新阶段
func (r *ProjectRepository) UpdatePhase(ctx context.Context, phase *entity.ProjectPhase) error {
	return r.db.WithContext(ctx).Save(phase).Error
}

// UpdateProgress 更新项目进度
func (r *ProjectRepository) UpdateProgress(ctx context.Context, projectID string, progress int) error {
	return r.db.WithContext(ctx).
		Model(&entity.Project{}).
		Where("id = ?", projectID).
		Updates(map[string]interface{}{
			"progress":   progress,
			"updated_at": time.Now(),
		}).Error
}
