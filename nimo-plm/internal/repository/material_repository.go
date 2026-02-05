package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bitfantasy/nimo-plm/internal/model/entity"
	"gorm.io/gorm"
)

// MaterialRepository 物料仓库
type MaterialRepository struct {
	db *gorm.DB
}

// NewMaterialRepository 创建物料仓库
func NewMaterialRepository(db *gorm.DB) *MaterialRepository {
	return &MaterialRepository{db: db}
}

// MaterialCategoryRepository 物料分类仓库
type MaterialCategoryRepository struct {
	db *gorm.DB
}

// NewMaterialCategoryRepository 创建物料分类仓库
func NewMaterialCategoryRepository(db *gorm.DB) *MaterialCategoryRepository {
	return &MaterialCategoryRepository{db: db}
}

// FindByID 根据ID查找物料
func (r *MaterialRepository) FindByID(ctx context.Context, id string) (*entity.Material, error) {
	var material entity.Material
	err := r.db.WithContext(ctx).
		Preload("Category").
		Where("id = ? AND deleted_at IS NULL", id).
		First(&material).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &material, nil
}

// FindByCode 根据编码查找物料
func (r *MaterialRepository) FindByCode(ctx context.Context, code string) (*entity.Material, error) {
	var material entity.Material
	err := r.db.WithContext(ctx).
		Where("code = ? AND deleted_at IS NULL", code).
		First(&material).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &material, nil
}

// Create 创建物料
func (r *MaterialRepository) Create(ctx context.Context, material *entity.Material) error {
	return r.db.WithContext(ctx).Create(material).Error
}

// Update 更新物料
func (r *MaterialRepository) Update(ctx context.Context, material *entity.Material) error {
	return r.db.WithContext(ctx).Save(material).Error
}

// Delete 软删除物料
func (r *MaterialRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Model(&entity.Material{}).
		Where("id = ?", id).
		Update("deleted_at", time.Now()).Error
}

// List 获取物料列表
func (r *MaterialRepository) List(ctx context.Context, page, pageSize int, filters map[string]interface{}) ([]entity.Material, int64, error) {
	var materials []entity.Material
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.Material{}).Where("deleted_at IS NULL")

	if keyword, ok := filters["keyword"].(string); ok && keyword != "" {
		query = query.Where("name ILIKE ? OR code ILIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if categoryID, ok := filters["category_id"].(string); ok && categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}
	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.
		Preload("Category").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&materials).Error

	return materials, total, err
}

// GetCategories 获取物料类别列表
func (r *MaterialRepository) GetCategories(ctx context.Context) ([]entity.MaterialCategory, error) {
	var categories []entity.MaterialCategory
	err := r.db.WithContext(ctx).
		Order("sort_order ASC, name ASC").
		Find(&categories).Error
	return categories, err
}

// GenerateCode 生成物料编码
func (r *MaterialRepository) GenerateCode(ctx context.Context, categoryPrefix string) (string, error) {
	var seq int64
	err := r.db.WithContext(ctx).Raw("SELECT nextval('material_code_seq')").Scan(&seq).Error
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("MAT-%s-%06d", categoryPrefix, seq), nil
}
