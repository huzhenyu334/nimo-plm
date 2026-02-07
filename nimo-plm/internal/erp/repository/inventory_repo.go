package repository

import (
	"github.com/bitfantasy/nimo-plm/internal/erp/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type InventoryRepository struct {
	db *gorm.DB
}

func NewInventoryRepository(db *gorm.DB) *InventoryRepository {
	return &InventoryRepository{db: db}
}

// GetByMaterialAndWarehouse 获取指定物料在指定仓库的库存
func (r *InventoryRepository) GetByMaterialAndWarehouse(materialID, warehouseID string) (*entity.Inventory, error) {
	var inv entity.Inventory
	err := r.db.Where("material_id = ? AND warehouse_id = ? AND deleted_at IS NULL", materialID, warehouseID).
		First(&inv).Error
	return &inv, err
}

// GetTotalStock 获取物料总库存（所有仓库）
func (r *InventoryRepository) GetTotalStock(materialID string) (float64, error) {
	var result struct{ Total float64 }
	err := r.db.Raw(`
		SELECT COALESCE(SUM(available_qty), 0) as total 
		FROM erp_inventory 
		WHERE material_id = ? AND deleted_at IS NULL
	`, materialID).Scan(&result).Error
	return result.Total, err
}

// UpsertInventory 更新或创建库存记录
func (r *InventoryRepository) UpsertInventory(inv *entity.Inventory) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "material_id"}, {Name: "warehouse_id"}, {Name: "batch_no"}},
		DoUpdates: clause.AssignmentColumns([]string{"quantity", "available_qty", "reserved_qty", "unit_cost", "last_moved_at", "updated_at"}),
	}).Create(inv).Error
}

func (r *InventoryRepository) Update(inv *entity.Inventory) error {
	return r.db.Save(inv).Error
}

func (r *InventoryRepository) CreateTransaction(tx *entity.InventoryTransaction) error {
	return r.db.Create(tx).Error
}

type InventoryListParams struct {
	MaterialID    string
	WarehouseID   string
	InventoryType string
	Keyword       string
	LowStock      bool
	Page          int
	Size          int
}

func (r *InventoryRepository) List(params InventoryListParams) ([]entity.Inventory, int64, error) {
	query := r.db.Model(&entity.Inventory{}).Where("deleted_at IS NULL")
	if params.MaterialID != "" {
		query = query.Where("material_id = ?", params.MaterialID)
	}
	if params.WarehouseID != "" {
		query = query.Where("warehouse_id = ?", params.WarehouseID)
	}
	if params.InventoryType != "" {
		query = query.Where("inventory_type = ?", params.InventoryType)
	}
	if params.Keyword != "" {
		kw := "%" + params.Keyword + "%"
		query = query.Where("material_code ILIKE ? OR material_name ILIKE ?", kw, kw)
	}
	if params.LowStock {
		query = query.Where("available_qty < safety_stock AND safety_stock > 0")
	}
	var total int64
	query.Count(&total)
	if params.Page <= 0 { params.Page = 1 }
	if params.Size <= 0 { params.Size = 20 }
	var items []entity.Inventory
	err := query.Preload("Warehouse").Order("updated_at DESC").
		Offset((params.Page-1)*params.Size).Limit(params.Size).Find(&items).Error
	return items, total, err
}

func (r *InventoryRepository) ListTransactions(materialID string, page, size int) ([]entity.InventoryTransaction, int64, error) {
	query := r.db.Model(&entity.InventoryTransaction{})
	if materialID != "" {
		query = query.Where("material_id = ?", materialID)
	}
	var total int64
	query.Count(&total)
	if page <= 0 { page = 1 }
	if size <= 0 { size = 20 }
	var txs []entity.InventoryTransaction
	err := query.Order("created_at DESC").Offset((page-1)*size).Limit(size).Find(&txs).Error
	return txs, total, err
}

// GetAlerts 获取库存预警列表
func (r *InventoryRepository) GetAlerts() ([]entity.Inventory, error) {
	var alerts []entity.Inventory
	err := r.db.Where("available_qty < safety_stock AND safety_stock > 0 AND deleted_at IS NULL").
		Find(&alerts).Error
	return alerts, err
}

// GetByMaterial 获取某物料所有仓库的库存
func (r *InventoryRepository) GetByMaterial(materialID string) ([]entity.Inventory, error) {
	var items []entity.Inventory
	err := r.db.Preload("Warehouse").
		Where("material_id = ? AND deleted_at IS NULL", materialID).
		Find(&items).Error
	return items, err
}

// DB 返回底层db用于事务
func (r *InventoryRepository) DB() *gorm.DB {
	return r.db
}
