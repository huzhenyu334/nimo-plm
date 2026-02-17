package entity

import "time"

// PurchaseOrder 采购订单
type PurchaseOrder struct {
	ID           string     `json:"id" gorm:"primaryKey;size:32"`
	POCode       string     `json:"po_code" gorm:"size:32;uniqueIndex;not null"`
	SupplierID   string     `json:"supplier_id" gorm:"size:32;not null;index"`
	PRID         *string    `json:"pr_id" gorm:"size:32"`
	SRMProjectID *string    `json:"srm_project_id" gorm:"size:32"`
	Type         string     `json:"type" gorm:"size:20;not null"`     // sample/production
	Status       string     `json:"status" gorm:"size:20;default:draft"` // draft/approved/sent/partial/received/completed/cancelled
	Round        int        `json:"round" gorm:"default:1"`
	PrevPOID     *string    `json:"prev_po_id" gorm:"size:32"`     // 上一轮PO
	Related8DID  *string    `json:"related_8d_id" gorm:"size:32"` // 关联8D改进单

	// 金额
	TotalAmount *float64 `json:"total_amount" gorm:"type:decimal(15,2)"`
	Currency    string   `json:"currency" gorm:"size:10;default:CNY"`

	// 交期
	ExpectedDate *time.Time `json:"expected_date"`
	ActualDate   *time.Time `json:"actual_date"`

	// 收货与付款
	ShippingAddress string `json:"shipping_address" gorm:"size:500"`
	PaymentTerms    string `json:"payment_terms" gorm:"size:100"`

	// 管理
	CreatedBy  string     `json:"created_by" gorm:"size:32"`
	ApprovedBy *string    `json:"approved_by" gorm:"size:32"`
	ApprovedAt *time.Time `json:"approved_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Notes      string     `json:"notes" gorm:"type:text"`

	// 关联
	Items    []POItem  `json:"items,omitempty" gorm:"foreignKey:POID"`
	Supplier *Supplier `json:"supplier,omitempty" gorm:"foreignKey:SupplierID"`
}

func (PurchaseOrder) TableName() string {
	return "srm_purchase_orders"
}

// PO状态
const (
	POStatusDraft     = "draft"
	POStatusSubmitted = "submitted"
	POStatusApproved  = "approved"
	POStatusSent      = "sent"
	POStatusPartial   = "partial"
	POStatusReceived  = "received"
	POStatusCompleted = "completed"
	POStatusCancelled = "cancelled"
)

// POItem PO行项
type POItem struct {
	ID            string   `json:"id" gorm:"primaryKey;size:32"`
	POID          string   `json:"po_id" gorm:"size:32;not null;index"`
	PRItemID      *string  `json:"pr_item_id" gorm:"size:32"`
	BOMItemID     *string  `json:"bom_item_id" gorm:"size:32"`   // 关联BOM行项来源
	MaterialID    *string  `json:"material_id" gorm:"size:32"`
	MaterialCode  string   `json:"material_code" gorm:"size:50"`
	MaterialName  string   `json:"material_name" gorm:"size:200;not null"`
	Specification string   `json:"specification" gorm:"size:500"`

	Quantity    float64  `json:"quantity" gorm:"type:decimal(10,2);not null"`
	Unit        string   `json:"unit" gorm:"size:20;default:pcs"`
	UnitPrice   *float64 `json:"unit_price" gorm:"type:decimal(12,4)"`
	TotalAmount *float64 `json:"total_amount" gorm:"type:decimal(15,2)"`

	// 收货
	ReceivedQty float64 `json:"received_qty" gorm:"type:decimal(10,2);default:0"`
	Status      string  `json:"status" gorm:"size:20;default:pending"` // pending/shipped/partial/received

	SortOrder int       `json:"sort_order" gorm:"default:0"`
	Notes     string    `json:"notes" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (POItem) TableName() string {
	return "srm_po_items"
}

// POItem状态
const (
	POItemStatusPending  = "pending"
	POItemStatusShipped  = "shipped"
	POItemStatusPartial  = "partial"
	POItemStatusReceived = "received"
)
