package repository

import (
	"context"

	"github.com/bitfantasy/nimo/internal/plm/entity"
	"gorm.io/gorm"
)

type SKURepository struct {
	db *gorm.DB
}

func NewSKURepository(db *gorm.DB) *SKURepository {
	return &SKURepository{db: db}
}

func (r *SKURepository) DB() *gorm.DB {
	return r.db
}

// ========== ProductSKU ==========

func (r *SKURepository) Create(ctx context.Context, sku *entity.ProductSKU) error {
	return r.db.WithContext(ctx).Create(sku).Error
}

func (r *SKURepository) FindByID(ctx context.Context, id string) (*entity.ProductSKU, error) {
	var sku entity.ProductSKU
	err := r.db.WithContext(ctx).
		Preload("CMFConfigs").
		Preload("CMFConfigs.BOMItem").
		Preload("BOMOverrides").
		Preload("BOMOverrides.BaseItem").
		First(&sku, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &sku, nil
}

func (r *SKURepository) ListByProject(ctx context.Context, projectID string) ([]entity.ProductSKU, error) {
	var skus []entity.ProductSKU
	err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("sort_order ASC, created_at ASC").
		Find(&skus).Error
	return skus, err
}

func (r *SKURepository) Update(ctx context.Context, sku *entity.ProductSKU) error {
	return r.db.WithContext(ctx).Save(sku).Error
}

func (r *SKURepository) Delete(ctx context.Context, id string) error {
	// Delete related CMF configs and BOM overrides first
	r.db.WithContext(ctx).Where("sku_id = ?", id).Delete(&entity.SKUCMFConfig{})
	r.db.WithContext(ctx).Where("sku_id = ?", id).Delete(&entity.SKUBOMOverride{})
	return r.db.WithContext(ctx).Delete(&entity.ProductSKU{}, "id = ?", id).Error
}

// ========== SKUCMFConfig ==========

func (r *SKURepository) ListCMFConfigs(ctx context.Context, skuID string) ([]entity.SKUCMFConfig, error) {
	var configs []entity.SKUCMFConfig
	err := r.db.WithContext(ctx).
		Preload("BOMItem").
		Where("sku_id = ?", skuID).
		Find(&configs).Error
	return configs, err
}

func (r *SKURepository) BatchSaveCMFConfigs(ctx context.Context, skuID string, configs []entity.SKUCMFConfig) error {
	tx := r.db.WithContext(ctx).Begin()
	// Delete existing configs for this SKU
	if err := tx.Where("sku_id = ?", skuID).Delete(&entity.SKUCMFConfig{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	// Insert new configs
	if len(configs) > 0 {
		if err := tx.Create(&configs).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

// ========== SKUBOMOverride ==========

func (r *SKURepository) ListBOMOverrides(ctx context.Context, skuID string) ([]entity.SKUBOMOverride, error) {
	var overrides []entity.SKUBOMOverride
	err := r.db.WithContext(ctx).
		Preload("BaseItem").
		Where("sku_id = ?", skuID).
		Find(&overrides).Error
	return overrides, err
}

func (r *SKURepository) CreateBOMOverride(ctx context.Context, override *entity.SKUBOMOverride) error {
	return r.db.WithContext(ctx).Create(override).Error
}

func (r *SKURepository) UpdateBOMOverride(ctx context.Context, override *entity.SKUBOMOverride) error {
	return r.db.WithContext(ctx).Save(override).Error
}

func (r *SKURepository) FindBOMOverrideByID(ctx context.Context, id string) (*entity.SKUBOMOverride, error) {
	var override entity.SKUBOMOverride
	err := r.db.WithContext(ctx).
		Preload("BaseItem").
		First(&override, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &override, nil
}

func (r *SKURepository) DeleteBOMOverride(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.SKUBOMOverride{}, "id = ?", id).Error
}
