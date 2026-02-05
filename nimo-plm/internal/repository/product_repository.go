package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bitfantasy/nimo-plm/internal/model/entity"
	"gorm.io/gorm"
)

// ProductRepository 产品仓库
type ProductRepository struct {
	db *gorm.DB
}

// NewProductRepository 创建产品仓库
func NewProductRepository(db *gorm.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

// ProductCategoryRepository 产品分类仓库
type ProductCategoryRepository struct {
	db *gorm.DB
}

// NewProductCategoryRepository 创建产品分类仓库
func NewProductCategoryRepository(db *gorm.DB) *ProductCategoryRepository {
	return &ProductCategoryRepository{db: db}
}

// FindProductByID 根据ID查找产品
func (r *ProductRepository) FindByID(ctx context.Context, id string) (*entity.Product, error) {
	var product entity.Product
	err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("Creator").
		Where("id = ? AND deleted_at IS NULL", id).
		First(&product).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &product, nil
}

// FindProductByCode 根据编码查找产品
func (r *ProductRepository) FindByCode(ctx context.Context, code string) (*entity.Product, error) {
	var product entity.Product
	err := r.db.WithContext(ctx).
		Where("code = ? AND deleted_at IS NULL", code).
		First(&product).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &product, nil
}

// CreateProduct 创建产品
func (r *ProductRepository) Create(ctx context.Context, product *entity.Product) error {
	return r.db.WithContext(ctx).Create(product).Error
}

// UpdateProduct 更新产品
func (r *ProductRepository) Update(ctx context.Context, product *entity.Product) error {
	return r.db.WithContext(ctx).Save(product).Error
}

// DeleteProduct 软删除产品
func (r *ProductRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Model(&entity.Product{}).
		Where("id = ?", id).
		Update("deleted_at", time.Now()).Error
}

// ListProducts 获取产品列表
func (r *ProductRepository) List(ctx context.Context, page, pageSize int, filters map[string]interface{}) ([]entity.Product, int64, error) {
	var products []entity.Product
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.Product{}).Where("deleted_at IS NULL")

	// 应用过滤条件
	if keyword, ok := filters["keyword"].(string); ok && keyword != "" {
		query = query.Where("name ILIKE ? OR code ILIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if categoryID, ok := filters["category_id"].(string); ok && categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}
	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where("status = ?", status)
	}
	if createdBy, ok := filters["created_by"].(string); ok && createdBy != "" {
		query = query.Where("created_by = ?", createdBy)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Preload("Category").
		Preload("Creator").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&products).Error

	return products, total, err
}

// GetCategories 获取产品类别列表
func (r *ProductRepository) GetCategories(ctx context.Context) ([]entity.ProductCategory, error) {
	var categories []entity.ProductCategory
	err := r.db.WithContext(ctx).
		Order("sort_order ASC, name ASC").
		Find(&categories).Error
	return categories, err
}

// GenerateProductCode 生成产品编码
func (r *ProductRepository) GenerateCode(ctx context.Context, prefix string) (string, error) {
	var seq int64
	err := r.db.WithContext(ctx).Raw("SELECT nextval('product_code_seq')").Scan(&seq).Error
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%06d", prefix, seq), nil
}

// UpdateBOMVersion 更新产品当前BOM版本
func (r *ProductRepository) UpdateBOMVersion(ctx context.Context, productID, version string) error {
	return r.db.WithContext(ctx).
		Model(&entity.Product{}).
		Where("id = ?", productID).
		Update("current_bom_version", version).Error
}
