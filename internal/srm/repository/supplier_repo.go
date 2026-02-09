package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/bitfantasy/nimo/internal/srm/entity"
	"gorm.io/gorm"
)

// SupplierRepository 供应商仓库
type SupplierRepository struct {
	db *gorm.DB
}

func NewSupplierRepository(db *gorm.DB) *SupplierRepository {
	return &SupplierRepository{db: db}
}

// FindAll 查询供应商列表
func (r *SupplierRepository) FindAll(ctx context.Context, page, pageSize int, filters map[string]string) ([]entity.Supplier, int64, error) {
	var items []entity.Supplier
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.Supplier{})

	if search := filters["search"]; search != "" {
		query = query.Where("name ILIKE ? OR code ILIKE ? OR short_name ILIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if category := filters["category"]; category != "" {
		query = query.Where("category = ?", category)
	}
	if level := filters["level"]; level != "" {
		query = query.Where("level = ?", level)
	}
	if status := filters["status"]; status != "" {
		query = query.Where("status = ?", status)
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

// FindByID 根据ID查找供应商
func (r *SupplierRepository) FindByID(ctx context.Context, id string) (*entity.Supplier, error) {
	var supplier entity.Supplier
	err := r.db.WithContext(ctx).
		Preload("Contacts").
		Where("id = ?", id).
		First(&supplier).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &supplier, nil
}

// Create 创建供应商
func (r *SupplierRepository) Create(ctx context.Context, supplier *entity.Supplier) error {
	return r.db.WithContext(ctx).Create(supplier).Error
}

// Update 更新供应商
func (r *SupplierRepository) Update(ctx context.Context, supplier *entity.Supplier) error {
	return r.db.WithContext(ctx).Save(supplier).Error
}

// Delete 删除供应商
func (r *SupplierRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&entity.Supplier{}).Error
}

// FindContacts 查找供应商联系人
func (r *SupplierRepository) FindContacts(ctx context.Context, supplierID string) ([]entity.SupplierContact, error) {
	var contacts []entity.SupplierContact
	err := r.db.WithContext(ctx).
		Where("supplier_id = ?", supplierID).
		Order("is_primary DESC, created_at ASC").
		Find(&contacts).Error
	return contacts, err
}

// CreateContact 创建供应商联系人
func (r *SupplierRepository) CreateContact(ctx context.Context, contact *entity.SupplierContact) error {
	return r.db.WithContext(ctx).Create(contact).Error
}

// DeleteContact 删除供应商联系人
func (r *SupplierRepository) DeleteContact(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&entity.SupplierContact{}).Error
}

// UpdateScores 更新供应商评分
func (r *SupplierRepository) UpdateScores(ctx context.Context, id string, quality, delivery, price, overall float64) error {
	return r.db.WithContext(ctx).
		Model(&entity.Supplier{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"quality_score":  quality,
			"delivery_score": delivery,
			"price_score":    price,
			"overall_score":  overall,
		}).Error
}

// CountByCodePrefix 统计编码前缀数量（用于生成编码）
func (r *SupplierRepository) CountByCodePrefix(ctx context.Context, prefix string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.Supplier{}).
		Where("code LIKE ?", prefix+"%").
		Count(&count).Error
	return count, err
}

// GenerateCode 生成供应商编码 SUP-{4位}
func (r *SupplierRepository) GenerateCode(ctx context.Context) (string, error) {
	var maxCode string
	err := r.db.WithContext(ctx).
		Model(&entity.Supplier{}).
		Select("COALESCE(MAX(code), 'SUP-0000')").
		Scan(&maxCode).Error
	if err != nil {
		return "", err
	}

	var seq int
	fmt.Sscanf(maxCode, "SUP-%04d", &seq)
	seq++
	return fmt.Sprintf("SUP-%04d", seq), nil
}
