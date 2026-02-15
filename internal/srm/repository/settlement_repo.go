package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bitfantasy/nimo/internal/srm/entity"
	"gorm.io/gorm"
)

// SettlementRepository 结算仓库
type SettlementRepository struct {
	db *gorm.DB
}

func NewSettlementRepository(db *gorm.DB) *SettlementRepository {
	return &SettlementRepository{db: db}
}

// FindAll 查询对账单列表
func (r *SettlementRepository) FindAll(ctx context.Context, page, pageSize int, filters map[string]string) ([]entity.Settlement, int64, error) {
	var items []entity.Settlement
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.Settlement{})

	if supplierID := filters["supplier_id"]; supplierID != "" {
		query = query.Where("supplier_id = ?", supplierID)
	}
	if status := filters["status"]; status != "" {
		query = query.Where("status = ?", status)
	}
	if startDate := filters["start_date"]; startDate != "" {
		query = query.Where("period_start >= ?", startDate)
	}
	if endDate := filters["end_date"]; endDate != "" {
		query = query.Where("period_end <= ?", endDate)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.
		Preload("Supplier").
		Preload("Disputes").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&items).Error

	return items, total, err
}

// FindByID 根据ID查找对账单
func (r *SettlementRepository) FindByID(ctx context.Context, id string) (*entity.Settlement, error) {
	var s entity.Settlement
	err := r.db.WithContext(ctx).
		Preload("Supplier").
		Preload("Disputes").
		Where("id = ?", id).
		First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

// Create 创建对账单
func (r *SettlementRepository) Create(ctx context.Context, s *entity.Settlement) error {
	return r.db.WithContext(ctx).Create(s).Error
}

// Update 更新对账单
func (r *SettlementRepository) Update(ctx context.Context, s *entity.Settlement) error {
	return r.db.WithContext(ctx).Save(s).Error
}

// Delete 删除对账单（仅草稿状态）
func (r *SettlementRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ? AND status = 'draft'", id).Delete(&entity.Settlement{}).Error
}

// GenerateCode 生成对账单编码 STL-YYYYMM-XXXX
func (r *SettlementRepository) GenerateCode(ctx context.Context) (string, error) {
	prefix := fmt.Sprintf("STL-%s", time.Now().Format("200601"))
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.Settlement{}).
		Where("settlement_code LIKE ?", prefix+"%").
		Count(&count).Error
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%04d", prefix, count+1), nil
}

// FindReceivedPOs 查询指定供应商在周期内已收货的PO
func (r *SettlementRepository) FindReceivedPOs(ctx context.Context, supplierID string, periodStart, periodEnd time.Time) ([]entity.PurchaseOrder, error) {
	var pos []entity.PurchaseOrder
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("supplier_id = ? AND status IN ? AND created_at BETWEEN ? AND ?",
			supplierID,
			[]string{"received", "completed"},
			periodStart, periodEnd,
		).
		Find(&pos).Error
	return pos, err
}

// CreateDispute 创建差异记录
func (r *SettlementRepository) CreateDispute(ctx context.Context, d *entity.SettlementDispute) error {
	return r.db.WithContext(ctx).Create(d).Error
}

// UpdateDispute 更新差异记录
func (r *SettlementRepository) UpdateDispute(ctx context.Context, d *entity.SettlementDispute) error {
	return r.db.WithContext(ctx).Save(d).Error
}

// FindDisputeByID 查找差异记录
func (r *SettlementRepository) FindDisputeByID(ctx context.Context, id string) (*entity.SettlementDispute, error) {
	var d entity.SettlementDispute
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&d).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &d, nil
}
