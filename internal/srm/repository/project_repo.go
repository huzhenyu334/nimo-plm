package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bitfantasy/nimo/internal/srm/entity"
	"gorm.io/gorm"
)

// ProjectRepository 采购项目仓库
type ProjectRepository struct {
	db *gorm.DB
}

func NewProjectRepository(db *gorm.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// FindAll 查询采购项目列表
func (r *ProjectRepository) FindAll(ctx context.Context, page, pageSize int, filters map[string]string) ([]entity.SRMProject, int64, error) {
	var items []entity.SRMProject
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.SRMProject{})

	if status := filters["status"]; status != "" {
		query = query.Where("status = ?", status)
	}
	if projectType := filters["type"]; projectType != "" {
		query = query.Where("type = ?", projectType)
	}
	if phase := filters["phase"]; phase != "" {
		query = query.Where("phase = ?", phase)
	}
	if plmProjectID := filters["plm_project_id"]; plmProjectID != "" {
		query = query.Where("plm_project_id = ?", plmProjectID)
	}
	if search := filters["search"]; search != "" {
		query = query.Where("name ILIKE ? OR code ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&items).Error

	return items, total, err
}

// FindByID 根据ID查找采购项目
func (r *ProjectRepository) FindByID(ctx context.Context, id string) (*entity.SRMProject, error) {
	var project entity.SRMProject
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&project).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &project, nil
}

// FindByPLMBOMID 根据PLM BOM ID查找采购项目（防重复）
func (r *ProjectRepository) FindByPLMBOMID(ctx context.Context, bomID string) (*entity.SRMProject, error) {
	var project entity.SRMProject
	err := r.db.WithContext(ctx).Where("plm_bom_id = ?", bomID).First(&project).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &project, nil
}

// Create 创建采购项目
func (r *ProjectRepository) Create(ctx context.Context, project *entity.SRMProject) error {
	return r.db.WithContext(ctx).Create(project).Error
}

// Update 更新采购项目
func (r *ProjectRepository) Update(ctx context.Context, project *entity.SRMProject) error {
	return r.db.WithContext(ctx).Save(project).Error
}

// UpdateProgress 更新采购项目进度计数
func (r *ProjectRepository) UpdateProgress(ctx context.Context, id string, counts map[string]int) error {
	return r.db.WithContext(ctx).Model(&entity.SRMProject{}).Where("id = ?", id).Updates(counts).Error
}

// GenerateCode 生成采购项目编码 SRMP-{year}-{4位}
func (r *ProjectRepository) GenerateCode(ctx context.Context) (string, error) {
	year := time.Now().Format("2006")
	prefix := fmt.Sprintf("SRMP-%s-", year)

	var maxCode string
	err := r.db.WithContext(ctx).
		Model(&entity.SRMProject{}).
		Select("COALESCE(MAX(code), '')").
		Where("code LIKE ?", prefix+"%").
		Scan(&maxCode).Error
	if err != nil {
		return "", err
	}

	var seq int
	if maxCode != "" {
		fmt.Sscanf(maxCode, "SRMP-"+year+"-%04d", &seq)
	}
	seq++
	return fmt.Sprintf("SRMP-%s-%04d", year, seq), nil
}
