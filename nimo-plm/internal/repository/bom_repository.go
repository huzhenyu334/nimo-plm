package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bitfantasy/nimo-plm/internal/model/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// generateID 生成32位ID
func generateID() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")[:32]
}

// BOMRepository BOM仓库
type BOMRepository struct {
	db *gorm.DB
}

// NewBOMRepository 创建BOM仓库
func NewBOMRepository(db *gorm.DB) *BOMRepository {
	return &BOMRepository{db: db}
}

// GetBOMByProductID 获取产品的BOM
func (r *BOMRepository) GetByProductID(ctx context.Context, productID, version string) (*entity.BOMHeader, error) {
	var bom entity.BOMHeader
	query := r.db.WithContext(ctx).
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("level ASC, sequence ASC")
		}).
		Preload("Items.Material").
		Where("product_id = ?", productID)

	if version != "" {
		query = query.Where("version = ?", version)
	} else {
		// 获取最新版本
		query = query.Order("created_at DESC")
	}

	err := query.First(&bom).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &bom, nil
}

// GetBOMByID 根据ID获取BOM
func (r *BOMRepository) GetByID(ctx context.Context, id string) (*entity.BOMHeader, error) {
	var bom entity.BOMHeader
	err := r.db.WithContext(ctx).
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("level ASC, sequence ASC")
		}).
		Preload("Items.Material").
		Where("id = ?", id).
		First(&bom).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &bom, nil
}

// ListBOMVersions 获取产品的BOM版本列表
func (r *BOMRepository) ListVersions(ctx context.Context, productID string) ([]entity.BOMHeader, error) {
	var boms []entity.BOMHeader
	err := r.db.WithContext(ctx).
		Where("product_id = ?", productID).
		Order("created_at DESC").
		Find(&boms).Error
	return boms, err
}

// CreateBOM 创建BOM
func (r *BOMRepository) Create(ctx context.Context, bom *entity.BOMHeader) error {
	return r.db.WithContext(ctx).Create(bom).Error
}

// UpdateBOM 更新BOM
func (r *BOMRepository) Update(ctx context.Context, bom *entity.BOMHeader) error {
	return r.db.WithContext(ctx).Save(bom).Error
}

// CreateBOMItem 创建BOM行项
func (r *BOMRepository) CreateItem(ctx context.Context, item *entity.BOMItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// UpdateBOMItem 更新BOM行项
func (r *BOMRepository) UpdateItem(ctx context.Context, item *entity.BOMItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}

// DeleteBOMItem 删除BOM行项
func (r *BOMRepository) DeleteItem(ctx context.Context, id string) error {
	// 先删除子项
	if err := r.db.WithContext(ctx).
		Where("parent_item_id = ?", id).
		Delete(&entity.BOMItem{}).Error; err != nil {
		return err
	}
	// 删除本项
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&entity.BOMItem{}).Error
}

// GetBOMItem 获取BOM行项
func (r *BOMRepository) GetItem(ctx context.Context, id string) (*entity.BOMItem, error) {
	var item entity.BOMItem
	err := r.db.WithContext(ctx).
		Preload("Material").
		Where("id = ?", id).
		First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

// GetBOMItems 获取BOM的所有行项
func (r *BOMRepository) GetItems(ctx context.Context, bomHeaderID string) ([]entity.BOMItem, error) {
	var items []entity.BOMItem
	err := r.db.WithContext(ctx).
		Preload("Material").
		Where("bom_header_id = ?", bomHeaderID).
		Order("level ASC, sequence ASC").
		Find(&items).Error
	return items, err
}

// GetMaxSequence 获取同级最大序号
func (r *BOMRepository) GetMaxSequence(ctx context.Context, bomHeaderID, parentItemID string) (int, error) {
	var maxSeq int
	query := r.db.WithContext(ctx).
		Model(&entity.BOMItem{}).
		Select("COALESCE(MAX(sequence), 0)").
		Where("bom_header_id = ?", bomHeaderID)

	if parentItemID == "" {
		query = query.Where("parent_item_id IS NULL")
	} else {
		query = query.Where("parent_item_id = ?", parentItemID)
	}

	err := query.Scan(&maxSeq).Error
	return maxSeq, err
}

// UpdateBOMStats 更新BOM统计信息
func (r *BOMRepository) UpdateStats(ctx context.Context, bomHeaderID string) error {
	// 更新总物料数
	var totalItems int64
	if err := r.db.WithContext(ctx).
		Model(&entity.BOMItem{}).
		Where("bom_header_id = ?", bomHeaderID).
		Count(&totalItems).Error; err != nil {
		return err
	}

	// 更新最大层级
	var maxLevel int
	if err := r.db.WithContext(ctx).
		Model(&entity.BOMItem{}).
		Select("COALESCE(MAX(level), 0)").
		Where("bom_header_id = ?", bomHeaderID).
		Scan(&maxLevel).Error; err != nil {
		return err
	}

	// 更新总成本
	var totalCost float64
	if err := r.db.WithContext(ctx).
		Model(&entity.BOMItem{}).
		Select("COALESCE(SUM(extended_cost), 0)").
		Where("bom_header_id = ? AND level = 0", bomHeaderID).
		Scan(&totalCost).Error; err != nil {
		return err
	}

	return r.db.WithContext(ctx).
		Model(&entity.BOMHeader{}).
		Where("id = ?", bomHeaderID).
		Updates(map[string]interface{}{
			"total_items": totalItems,
			"max_level":   maxLevel,
			"total_cost":  totalCost,
			"updated_at":  time.Now(),
		}).Error
}

// GetDraftBOM 获取产品的草稿BOM
func (r *BOMRepository) GetDraftBOM(ctx context.Context, productID string) (*entity.BOMHeader, error) {
	var bom entity.BOMHeader
	err := r.db.WithContext(ctx).
		Where("product_id = ? AND status = ?", productID, entity.BOMStatusDraft).
		First(&bom).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &bom, nil
}

// CreateDraftBOM 创建草稿BOM
func (r *BOMRepository) CreateDraftBOM(ctx context.Context, productID, userID string) (*entity.BOMHeader, error) {
	// 查找最新版本号
	var latestVersion string
	r.db.WithContext(ctx).
		Model(&entity.BOMHeader{}).
		Select("version").
		Where("product_id = ?", productID).
		Order("created_at DESC").
		Limit(1).
		Scan(&latestVersion)

	// 计算新版本号
	newVersion := "1.0"
	if latestVersion != "" {
		// 简单的版本号递增逻辑
		var major, minor int
		fmt.Sscanf(latestVersion, "%d.%d", &major, &minor)
		newVersion = fmt.Sprintf("%d.%d", major, minor+1)
	}

	now := time.Now()
	bom := &entity.BOMHeader{
		ID:        generateID(),
		ProductID: productID,
		Version:   newVersion,
		Status:    entity.BOMStatusDraft,
		CreatedBy: userID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := r.db.WithContext(ctx).Create(bom).Error; err != nil {
		return nil, err
	}

	return bom, nil
}

// CompareBOM 对比两个BOM版本
func (r *BOMRepository) Compare(ctx context.Context, productID, versionA, versionB string) (added, removed, modified []entity.BOMItem, err error) {
	// 获取版本A的物料
	var itemsA []entity.BOMItem
	err = r.db.WithContext(ctx).
		Joins("JOIN bom_headers ON bom_items.bom_header_id = bom_headers.id").
		Preload("Material").
		Where("bom_headers.product_id = ? AND bom_headers.version = ?", productID, versionA).
		Find(&itemsA).Error
	if err != nil {
		return
	}

	// 获取版本B的物料
	var itemsB []entity.BOMItem
	err = r.db.WithContext(ctx).
		Joins("JOIN bom_headers ON bom_items.bom_header_id = bom_headers.id").
		Preload("Material").
		Where("bom_headers.product_id = ? AND bom_headers.version = ?", productID, versionB).
		Find(&itemsB).Error
	if err != nil {
		return
	}

	// 建立物料映射
	mapA := make(map[string]entity.BOMItem)
	for _, item := range itemsA {
		mapA[item.MaterialID] = item
	}

	mapB := make(map[string]entity.BOMItem)
	for _, item := range itemsB {
		mapB[item.MaterialID] = item
	}

	// 对比
	for materialID, itemB := range mapB {
		if itemA, exists := mapA[materialID]; exists {
			// 检查是否有修改
			if itemA.Quantity != itemB.Quantity || itemA.Position != itemB.Position {
				modified = append(modified, itemB)
			}
			delete(mapA, materialID)
		} else {
			// 新增
			added = append(added, itemB)
		}
	}

	// 剩余的是删除的
	for _, item := range mapA {
		removed = append(removed, item)
	}

	return
}
