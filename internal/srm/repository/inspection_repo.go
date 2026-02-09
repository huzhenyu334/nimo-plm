package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bitfantasy/nimo/internal/srm/entity"
	"gorm.io/gorm"
)

// InspectionRepository 检验仓库
type InspectionRepository struct {
	db *gorm.DB
}

func NewInspectionRepository(db *gorm.DB) *InspectionRepository {
	return &InspectionRepository{db: db}
}

// FindAll 查询检验列表
func (r *InspectionRepository) FindAll(ctx context.Context, page, pageSize int, filters map[string]string) ([]entity.Inspection, int64, error) {
	var items []entity.Inspection
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.Inspection{})

	if supplierID := filters["supplier_id"]; supplierID != "" {
		query = query.Where("supplier_id = ?", supplierID)
	}
	if status := filters["status"]; status != "" {
		query = query.Where("status = ?", status)
	}
	if result := filters["result"]; result != "" {
		query = query.Where("result = ?", result)
	}
	if poID := filters["po_id"]; poID != "" {
		query = query.Where("po_id = ?", poID)
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

// FindByID 根据ID查找检验
func (r *InspectionRepository) FindByID(ctx context.Context, id string) (*entity.Inspection, error) {
	var inspection entity.Inspection
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&inspection).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &inspection, nil
}

// Create 创建检验
func (r *InspectionRepository) Create(ctx context.Context, inspection *entity.Inspection) error {
	return r.db.WithContext(ctx).Create(inspection).Error
}

// Update 更新检验
func (r *InspectionRepository) Update(ctx context.Context, inspection *entity.Inspection) error {
	return r.db.WithContext(ctx).Save(inspection).Error
}

// GenerateCode 生成检验编码 IQC-{year}-{4位}
func (r *InspectionRepository) GenerateCode(ctx context.Context) (string, error) {
	year := time.Now().Format("2006")
	prefix := fmt.Sprintf("IQC-%s-", year)

	var maxCode string
	err := r.db.WithContext(ctx).
		Model(&entity.Inspection{}).
		Select("COALESCE(MAX(inspection_code), '')").
		Where("inspection_code LIKE ?", prefix+"%").
		Scan(&maxCode).Error
	if err != nil {
		return "", err
	}

	var seq int
	if maxCode != "" {
		fmt.Sscanf(maxCode, "IQC-"+year+"-%04d", &seq)
	}
	seq++
	return fmt.Sprintf("IQC-%s-%04d", year, seq), nil
}
