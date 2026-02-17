package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bitfantasy/nimo/internal/srm/entity"
	"gorm.io/gorm"
)

// PORepository 采购订单仓库
type PORepository struct {
	db *gorm.DB
}

func NewPORepository(db *gorm.DB) *PORepository {
	return &PORepository{db: db}
}

// FindAll 查询采购订单列表
func (r *PORepository) FindAll(ctx context.Context, page, pageSize int, filters map[string]string) ([]entity.PurchaseOrder, int64, error) {
	var items []entity.PurchaseOrder
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.PurchaseOrder{})

	if supplierID := filters["supplier_id"]; supplierID != "" {
		query = query.Where("supplier_id = ?", supplierID)
	}
	if status := filters["status"]; status != "" {
		query = query.Where("status = ?", status)
	}
	if poType := filters["type"]; poType != "" {
		query = query.Where("type = ?", poType)
	}
	if search := filters["search"]; search != "" {
		query = query.Where("po_code ILIKE ?", "%"+search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.
		Preload("Supplier").
		Preload("Items").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&items).Error

	return items, total, err
}

// FindByID 根据ID查找采购订单（含行项）
func (r *PORepository) FindByID(ctx context.Context, id string) (*entity.PurchaseOrder, error) {
	var po entity.PurchaseOrder
	err := r.db.WithContext(ctx).
		Preload("Supplier").
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC")
		}).
		Where("id = ?", id).
		First(&po).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &po, nil
}

// Create 创建采购订单
func (r *PORepository) Create(ctx context.Context, po *entity.PurchaseOrder) error {
	return r.db.WithContext(ctx).Create(po).Error
}

// Update 更新采购订单
func (r *PORepository) Update(ctx context.Context, po *entity.PurchaseOrder) error {
	return r.db.WithContext(ctx).Save(po).Error
}

// Delete 删除采购订单及行项
func (r *PORepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("po_id = ?", id).Delete(&entity.POItem{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", id).Delete(&entity.PurchaseOrder{}).Error
	})
}

// ReceiveItem 收货（更新行项收货数量和状态）
func (r *PORepository) ReceiveItem(ctx context.Context, itemID string, receivedQty float64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item entity.POItem
		if err := tx.Where("id = ?", itemID).First(&item).Error; err != nil {
			return err
		}

		item.ReceivedQty += receivedQty
		if item.ReceivedQty >= item.Quantity {
			item.Status = entity.POItemStatusReceived
		} else if item.ReceivedQty > 0 {
			item.Status = entity.POItemStatusPartial
		}

		return tx.Save(&item).Error
	})
}

// FindItemByID 查找PO行项
func (r *PORepository) FindItemByID(ctx context.Context, itemID string) (*entity.POItem, error) {
	var item entity.POItem
	err := r.db.WithContext(ctx).Where("id = ?", itemID).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

// GenerateCode 生成PO编码 PO-{year}-{4位}
func (r *PORepository) GenerateCode(ctx context.Context) (string, error) {
	year := time.Now().Format("2006")
	prefix := fmt.Sprintf("PO-%s-", year)

	var maxCode string
	err := r.db.WithContext(ctx).
		Model(&entity.PurchaseOrder{}).
		Select("COALESCE(MAX(po_code), '')").
		Where("po_code LIKE ?", prefix+"%").
		Scan(&maxCode).Error
	if err != nil {
		return "", err
	}

	var seq int
	if maxCode != "" {
		fmt.Sscanf(maxCode, "PO-"+year+"-%04d", &seq)
	}
	seq++
	return fmt.Sprintf("PO-%s-%04d", year, seq), nil
}
