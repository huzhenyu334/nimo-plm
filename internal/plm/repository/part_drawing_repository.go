package repository

import (
	"context"
	"fmt"

	"github.com/bitfantasy/nimo/internal/plm/entity"
	"gorm.io/gorm"
)

type PartDrawingRepository struct {
	db *gorm.DB
}

func NewPartDrawingRepository(db *gorm.DB) *PartDrawingRepository {
	return &PartDrawingRepository{db: db}
}

// ListByBOMItem 按BOMItem获取图纸列表（按type分组, version降序）
func (r *PartDrawingRepository) ListByBOMItem(ctx context.Context, bomItemID string) ([]entity.PartDrawing, error) {
	var drawings []entity.PartDrawing
	err := r.db.WithContext(ctx).
		Preload("Uploader").
		Where("bom_item_id = ?", bomItemID).
		Order("drawing_type ASC, created_at DESC").
		Find(&drawings).Error
	return drawings, err
}

// ListByBOMItems 批量获取多个BOMItem的图纸（避免N+1）
func (r *PartDrawingRepository) ListByBOMItems(ctx context.Context, bomItemIDs []string) ([]entity.PartDrawing, error) {
	var drawings []entity.PartDrawing
	if len(bomItemIDs) == 0 {
		return drawings, nil
	}
	err := r.db.WithContext(ctx).
		Preload("Uploader").
		Where("bom_item_id IN ?", bomItemIDs).
		Order("drawing_type ASC, created_at DESC").
		Find(&drawings).Error
	return drawings, err
}

// FindByID 根据ID查找图纸
func (r *PartDrawingRepository) FindByID(ctx context.Context, id string) (*entity.PartDrawing, error) {
	var drawing entity.PartDrawing
	err := r.db.WithContext(ctx).
		Preload("Uploader").
		First(&drawing, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &drawing, nil
}

// Create 创建图纸记录
func (r *PartDrawingRepository) Create(ctx context.Context, drawing *entity.PartDrawing) error {
	return r.db.WithContext(ctx).Create(drawing).Error
}

// Delete 删除图纸记录
func (r *PartDrawingRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.PartDrawing{}, "id = ?", id).Error
}

// GetNextVersion 获取下一个版本号
func (r *PartDrawingRepository) GetNextVersion(ctx context.Context, bomItemID, drawingType string) (string, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.PartDrawing{}).
		Where("bom_item_id = ? AND drawing_type = ?", bomItemID, drawingType).
		Count(&count).Error
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("v%d", count+1), nil
}
