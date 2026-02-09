package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bitfantasy/nimo/internal/srm/entity"
	"gorm.io/gorm"
)

// PRRepository 采购需求仓库
type PRRepository struct {
	db *gorm.DB
}

func NewPRRepository(db *gorm.DB) *PRRepository {
	return &PRRepository{db: db}
}

// FindAll 查询采购需求列表
func (r *PRRepository) FindAll(ctx context.Context, page, pageSize int, filters map[string]string) ([]entity.PurchaseRequest, int64, error) {
	var items []entity.PurchaseRequest
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.PurchaseRequest{})

	if projectID := filters["project_id"]; projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}
	if status := filters["status"]; status != "" {
		query = query.Where("status = ?", status)
	}
	if prType := filters["type"]; prType != "" {
		query = query.Where("type = ?", prType)
	}
	if search := filters["search"]; search != "" {
		query = query.Where("title ILIKE ? OR pr_code ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.
		Preload("Items").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&items).Error

	return items, total, err
}

// FindByID 根据ID查找采购需求（含行项）
func (r *PRRepository) FindByID(ctx context.Context, id string) (*entity.PurchaseRequest, error) {
	var pr entity.PurchaseRequest
	err := r.db.WithContext(ctx).
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC")
		}).
		Where("id = ?", id).
		First(&pr).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &pr, nil
}

// Create 创建采购需求
func (r *PRRepository) Create(ctx context.Context, pr *entity.PurchaseRequest) error {
	return r.db.WithContext(ctx).Create(pr).Error
}

// Update 更新采购需求
func (r *PRRepository) Update(ctx context.Context, pr *entity.PurchaseRequest) error {
	return r.db.WithContext(ctx).Save(pr).Error
}

// CreateItem 创建PR行项
func (r *PRRepository) CreateItem(ctx context.Context, item *entity.PRItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// UpdateItem 更新PR行项
func (r *PRRepository) UpdateItem(ctx context.Context, item *entity.PRItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}

// FindByBOMID 根据BOM ID查找PR（防重复）
func (r *PRRepository) FindByBOMID(ctx context.Context, bomID string) (*entity.PurchaseRequest, error) {
	var pr entity.PurchaseRequest
	err := r.db.WithContext(ctx).
		Where("bom_id = ?", bomID).
		First(&pr).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &pr, nil
}

// GenerateCode 生成PR编码 PR-{year}-{4位}
func (r *PRRepository) GenerateCode(ctx context.Context) (string, error) {
	year := time.Now().Format("2006")
	prefix := fmt.Sprintf("PR-%s-", year)

	var maxCode string
	err := r.db.WithContext(ctx).
		Model(&entity.PurchaseRequest{}).
		Select("COALESCE(MAX(pr_code), '')").
		Where("pr_code LIKE ?", prefix+"%").
		Scan(&maxCode).Error
	if err != nil {
		return "", err
	}

	var seq int
	if maxCode != "" {
		fmt.Sscanf(maxCode, "PR-"+year+"-%04d", &seq)
	}
	seq++
	return fmt.Sprintf("PR-%s-%04d", year, seq), nil
}
