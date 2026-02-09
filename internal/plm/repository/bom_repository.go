package repository

import (
	"context"
	"github.com/bitfantasy/nimo/internal/plm/entity"

	"gorm.io/gorm"
)

type ProjectBOMRepository struct {
	db *gorm.DB
}

func NewProjectBOMRepository(db *gorm.DB) *ProjectBOMRepository {
	return &ProjectBOMRepository{db: db}
}

func (r *ProjectBOMRepository) DB() *gorm.DB {
	return r.db
}

// Create 创建BOM
func (r *ProjectBOMRepository) Create(ctx context.Context, bom *entity.ProjectBOM) error {
	return r.db.WithContext(ctx).Create(bom).Error
}

// FindByID 根据ID查找BOM
func (r *ProjectBOMRepository) FindByID(ctx context.Context, id string) (*entity.ProjectBOM, error) {
	var bom entity.ProjectBOM
	err := r.db.WithContext(ctx).
		Preload("Phase").
		Preload("Items").
		Preload("Items.Material").
		Preload("Submitter").
		Preload("Reviewer").
		Preload("Creator").
		First(&bom, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &bom, nil
}

// ListByProject 获取项目的BOM列表
func (r *ProjectBOMRepository) ListByProject(ctx context.Context, projectID string, bomType, status string) ([]entity.ProjectBOM, error) {
	var boms []entity.ProjectBOM
	query := r.db.WithContext(ctx).
		Preload("Phase").
		Preload("Creator").
		Preload("Submitter").
		Preload("Reviewer").
		Where("project_id = ?", projectID)

	if bomType != "" {
		query = query.Where("bom_type = ?", bomType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	err := query.Order("created_at DESC").Find(&boms).Error
	return boms, err
}

// Update 更新BOM
func (r *ProjectBOMRepository) Update(ctx context.Context, bom *entity.ProjectBOM) error {
	return r.db.WithContext(ctx).Save(bom).Error
}

// Delete 删除BOM
func (r *ProjectBOMRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.ProjectBOM{}, "id = ?", id).Error
}

// CreateItem 创建BOM行项
func (r *ProjectBOMRepository) CreateItem(ctx context.Context, item *entity.ProjectBOMItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// FindItemByID 根据ID查找BOM行项
func (r *ProjectBOMRepository) FindItemByID(ctx context.Context, id string) (*entity.ProjectBOMItem, error) {
	var item entity.ProjectBOMItem
	err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// UpdateItem 更新BOM行项
func (r *ProjectBOMRepository) UpdateItem(ctx context.Context, item *entity.ProjectBOMItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}

// DeleteItem 删除BOM行项
func (r *ProjectBOMRepository) DeleteItem(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.ProjectBOMItem{}, "id = ?", id).Error
}

// DeleteItemsByBOM 删除BOM所有行项
func (r *ProjectBOMRepository) DeleteItemsByBOM(ctx context.Context, bomID string) error {
	return r.db.WithContext(ctx).Delete(&entity.ProjectBOMItem{}, "bom_id = ?", bomID).Error
}

// CountItems 统计BOM行项数
func (r *ProjectBOMRepository) CountItems(ctx context.Context, bomID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.ProjectBOMItem{}).Where("bom_id = ?", bomID).Count(&count).Error
	return count, err
}

// BatchCreateItems 批量创建BOM行项
func (r *ProjectBOMRepository) BatchCreateItems(ctx context.Context, items []entity.ProjectBOMItem) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&items).Error
}

// ListItemsByBOM 获取BOM的所有行项（按item_number排序）
func (r *ProjectBOMRepository) ListItemsByBOM(ctx context.Context, bomID string) ([]entity.ProjectBOMItem, error) {
	var items []entity.ProjectBOMItem
	err := r.db.WithContext(ctx).
		Where("bom_id = ?", bomID).
		Order("item_number ASC").
		Find(&items).Error
	return items, err
}

// MatchMaterialByNameAndPN 通过名称+制造商料号匹配物料库
func (r *ProjectBOMRepository) MatchMaterialByNameAndPN(ctx context.Context, name, manufacturerPN string) (*entity.Material, error) {
	var material entity.Material
	query := r.db.WithContext(ctx).Where("deleted_at IS NULL")
	if manufacturerPN != "" {
		query = query.Where("name ILIKE ? OR code ILIKE ?", "%"+name+"%", "%"+manufacturerPN+"%")
	} else {
		query = query.Where("name ILIKE ?", "%"+name+"%")
	}
	err := query.First(&material).Error
	if err != nil {
		return nil, err
	}
	return &material, nil
}

// CreateRelease 创建BOM发布快照
func (r *ProjectBOMRepository) CreateRelease(ctx context.Context, release *entity.BOMRelease) error {
	return r.db.WithContext(ctx).Create(release).Error
}

// ListPendingReleases 获取待同步的发布快照
func (r *ProjectBOMRepository) ListPendingReleases(ctx context.Context) ([]entity.BOMRelease, error) {
	var releases []entity.BOMRelease
	err := r.db.WithContext(ctx).
		Where("status = ?", "pending").
		Order("created_at ASC").
		Find(&releases).Error
	return releases, err
}

// FindReleaseByID 根据ID查找发布快照
func (r *ProjectBOMRepository) FindReleaseByID(ctx context.Context, id string) (*entity.BOMRelease, error) {
	var release entity.BOMRelease
	err := r.db.WithContext(ctx).First(&release, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &release, nil
}

// UpdateRelease 更新发布快照
func (r *ProjectBOMRepository) UpdateRelease(ctx context.Context, release *entity.BOMRelease) error {
	return r.db.WithContext(ctx).Save(release).Error
}
