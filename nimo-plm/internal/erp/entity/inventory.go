package entity

import (
	"time"
)

// InventoryType 库存类型
const (
	InventoryTypeRaw   = "RAW"   // 原材料
	InventoryTypeWIP   = "WIP"   // 半成品
	InventoryTypeFG    = "FG"    // 成品
	InventoryTypeSpare = "SPARE" // 备品备件
)

// TransactionType 库存交易类型
const (
	TxTypePurchaseIn    = "PURCHASE_IN"    // 采购入库
	TxTypeProductionIn  = "PRODUCTION_IN"  // 生产入库
	TxTypeReturnIn      = "RETURN_IN"      // 退货入库
	TxTypeProductionOut = "PRODUCTION_OUT" // 生产领料
	TxTypeSalesOut      = "SALES_OUT"      // 销售出库
	TxTypeScrapOut      = "SCRAP_OUT"      // 报废出库
	TxTypeAdjust        = "ADJUST"         // 库存调整
	TxTypeTransfer      = "TRANSFER"       // 库存调拨
)

// Inventory 库存记录
type Inventory struct {
	ID            string     `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	MaterialID    string     `json:"material_id" gorm:"size:32;not null;index"`
	MaterialCode  string     `json:"material_code" gorm:"size:64"`
	MaterialName  string     `json:"material_name" gorm:"size:128"`
	WarehouseID   string     `json:"warehouse_id" gorm:"type:uuid;not null;index"`
	LocationID    string     `json:"location_id" gorm:"type:uuid"`
	BatchNo       string     `json:"batch_no" gorm:"size:50;index"`
	SerialNo      string     `json:"serial_no" gorm:"size:100"`
	InventoryType string     `json:"inventory_type" gorm:"size:10;not null;default:RAW"`
	Quantity      float64    `json:"quantity" gorm:"type:decimal(12,4);not null;default:0"`
	ReservedQty   float64    `json:"reserved_qty" gorm:"type:decimal(12,4);default:0"`
	AvailableQty  float64    `json:"available_qty" gorm:"type:decimal(12,4);default:0"`
	UnitCost      float64    `json:"unit_cost" gorm:"type:decimal(12,4);default:0"`
	Unit          string     `json:"unit" gorm:"size:20;not null;default:pcs"`
	SafetyStock   float64    `json:"safety_stock" gorm:"type:decimal(12,4);default:0"`
	MaxStock      float64    `json:"max_stock" gorm:"type:decimal(12,4);default:0"`
	ExpiryDate    *time.Time `json:"expiry_date"`
	LastMovedAt   *time.Time `json:"last_moved_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at" gorm:"index"`

	Warehouse *Warehouse `json:"warehouse,omitempty" gorm:"foreignKey:WarehouseID"`
}

func (Inventory) TableName() string {
	return "erp_inventory"
}

// InventoryTransaction 库存交易记录
type InventoryTransaction struct {
	ID              string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	MaterialID      string    `json:"material_id" gorm:"size:32;not null;index"`
	MaterialCode    string    `json:"material_code" gorm:"size:64"`
	MaterialName    string    `json:"material_name" gorm:"size:128"`
	WarehouseID     string    `json:"warehouse_id" gorm:"type:uuid;not null;index"`
	TransactionType string    `json:"transaction_type" gorm:"size:20;not null"`
	Quantity        float64   `json:"quantity" gorm:"type:decimal(12,4);not null"` // 正=入，负=出
	BatchNo         string    `json:"batch_no" gorm:"size:50"`
	SerialNo        string    `json:"serial_no" gorm:"size:100"`
	UnitCost        float64   `json:"unit_cost" gorm:"type:decimal(12,4);default:0"`
	ReferenceType   string    `json:"reference_type" gorm:"size:50;not null"` // PO, WO, SO
	ReferenceID     string    `json:"reference_id" gorm:"size:64;not null"`
	ReferenceCode   string    `json:"reference_code" gorm:"size:50"`
	Notes           string    `json:"notes" gorm:"type:text"`
	CreatedBy       string    `json:"created_by" gorm:"size:64;not null"`
	CreatedAt       time.Time `json:"created_at"`
}

func (InventoryTransaction) TableName() string {
	return "erp_inventory_transactions"
}
