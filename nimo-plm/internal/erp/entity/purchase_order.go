package entity

import (
	"time"
)

// PurchaseOrderStatus 采购订单状态
const (
	POStatusDraft     = "DRAFT"
	POStatusPending   = "PENDING"
	POStatusApproved  = "APPROVED"
	POStatusSent      = "SENT"
	POStatusPartial   = "PARTIAL"
	POStatusReceived  = "RECEIVED"
	POStatusClosed    = "CLOSED"
	POStatusCancelled = "CANCELLED"
)

// PurchaseRequisitionStatus 采购需求状态
const (
	PRStatusDraft    = "DRAFT"
	PRStatusPending  = "PENDING"
	PRStatusApproved = "APPROVED"
	PRStatusOrdered  = "ORDERED"
	PRStatusClosed   = "CLOSED"
)

// PurchaseRequisition 采购需求（PR）
type PurchaseRequisition struct {
	ID           string     `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	PRCode       string     `json:"pr_code" gorm:"size:50;not null;uniqueIndex"`
	MaterialID   string     `json:"material_id" gorm:"size:32;not null;index"`
	MaterialCode string     `json:"material_code" gorm:"size:64"`
	MaterialName string     `json:"material_name" gorm:"size:128"`
	Quantity     float64    `json:"quantity" gorm:"type:decimal(12,4);not null"`
	Unit         string     `json:"unit" gorm:"size:20;not null;default:pcs"`
	RequiredDate *time.Time `json:"required_date"`
	Status       string     `json:"status" gorm:"size:20;not null;default:DRAFT"`
	Source       string     `json:"source" gorm:"size:20"` // MRP, MANUAL
	SourceID     string     `json:"source_id" gorm:"size:64"` // MRP运行ID或其他来源ID
	POID         *string    `json:"po_id" gorm:"type:uuid"` // 关联的PO
	Notes        string     `json:"notes" gorm:"type:text"`
	CreatedBy    string     `json:"created_by" gorm:"size:64"`
	ApprovedBy   string     `json:"approved_by" gorm:"size:64"`
	ApprovedAt   *time.Time `json:"approved_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at" gorm:"index"`
}

func (PurchaseRequisition) TableName() string {
	return "erp_purchase_requisitions"
}

// PurchaseOrder 采购订单（PO）
type PurchaseOrder struct {
	ID           string     `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	POCode       string     `json:"po_code" gorm:"size:50;not null;uniqueIndex"`
	SupplierID   string     `json:"supplier_id" gorm:"type:uuid;not null;index"`
	Status       string     `json:"status" gorm:"size:20;not null;default:DRAFT"`
	TotalAmount  float64    `json:"total_amount" gorm:"type:decimal(12,2);default:0"`
	Currency     string     `json:"currency" gorm:"size:10;not null;default:CNY"`
	OrderDate    *time.Time `json:"order_date"`
	ExpectedDate *time.Time `json:"expected_date"`
	ReceivedDate *time.Time `json:"received_date"`
	Notes        string     `json:"notes" gorm:"type:text"`
	CreatedBy    string     `json:"created_by" gorm:"size:64;not null"`
	ApprovedBy   string     `json:"approved_by" gorm:"size:64"`
	ApprovedAt   *time.Time `json:"approved_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at" gorm:"index"`

	Supplier *Supplier       `json:"supplier,omitempty" gorm:"foreignKey:SupplierID"`
	Items    []POItem        `json:"items,omitempty" gorm:"foreignKey:POID"`
}

func (PurchaseOrder) TableName() string {
	return "erp_purchase_orders"
}

// POItemStatus 采购订单行状态
const (
	POItemStatusOpen     = "OPEN"
	POItemStatusPartial  = "PARTIAL"
	POItemStatusReceived = "RECEIVED"
	POItemStatusClosed   = "CLOSED"
)

// POItem 采购订单明细
type POItem struct {
	ID          string     `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	POID        string     `json:"po_id" gorm:"type:uuid;not null;index"`
	MaterialID  string     `json:"material_id" gorm:"size:32;not null"`
	MaterialCode string    `json:"material_code" gorm:"size:64"`
	MaterialName string    `json:"material_name" gorm:"size:128"`
	Quantity    float64    `json:"quantity" gorm:"type:decimal(12,4);not null"`
	Unit        string     `json:"unit" gorm:"size:20;not null;default:pcs"`
	UnitPrice   float64    `json:"unit_price" gorm:"type:decimal(12,4);not null"`
	Amount      float64    `json:"amount" gorm:"type:decimal(12,2);not null"`
	ReceivedQty float64    `json:"received_qty" gorm:"type:decimal(12,4);default:0"`
	Status      string     `json:"status" gorm:"size:20;not null;default:OPEN"`
	PRID        *string    `json:"pr_id" gorm:"type:uuid"` // 关联的PR
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	PurchaseOrder *PurchaseOrder `json:"purchase_order,omitempty" gorm:"foreignKey:POID"`
}

func (POItem) TableName() string {
	return "erp_po_items"
}
