package entity

import "time"

// Settlement 对账单
type Settlement struct {
	ID             string     `json:"id" gorm:"primaryKey;size:32"`
	SettlementCode string    `json:"settlement_code" gorm:"size:32;uniqueIndex;not null"`
	SupplierID     string    `json:"supplier_id" gorm:"size:32;not null;index"`
	PeriodStart    *time.Time `json:"period_start"`
	PeriodEnd      *time.Time `json:"period_end"`
	Status         string     `json:"status" gorm:"size:20;default:draft"` // draft/confirmed/invoiced/paid

	// 金额
	POAmount       *float64 `json:"po_amount" gorm:"type:decimal(15,2)"`
	ReceivedAmount *float64 `json:"received_amount" gorm:"type:decimal(15,2)"`
	Deduction      *float64 `json:"deduction" gorm:"type:decimal(15,2);default:0"`
	FinalAmount    *float64 `json:"final_amount" gorm:"type:decimal(15,2)"`
	Currency       string   `json:"currency" gorm:"size:10;default:CNY"`

	// 发票
	InvoiceNo     string   `json:"invoice_no" gorm:"size:100"`
	InvoiceAmount *float64 `json:"invoice_amount" gorm:"type:decimal(15,2)"`
	InvoiceURL    string   `json:"invoice_url" gorm:"size:500"`

	// 确认
	ConfirmedByBuyer    bool       `json:"confirmed_by_buyer" gorm:"default:false"`
	ConfirmedBySupplier bool       `json:"confirmed_by_supplier" gorm:"default:false"`
	ConfirmedAt         *time.Time `json:"confirmed_at"`

	CreatedBy string    `json:"created_by" gorm:"size:32"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Notes     string    `json:"notes" gorm:"type:text"`

	// 关联
	Disputes []SettlementDispute `json:"disputes,omitempty" gorm:"foreignKey:SettlementID"`
	Supplier *Supplier           `json:"supplier,omitempty" gorm:"foreignKey:SupplierID"`
}

func (Settlement) TableName() string {
	return "srm_settlements"
}

// SettlementDispute 对账差异记录
type SettlementDispute struct {
	ID           string    `json:"id" gorm:"primaryKey;size:32"`
	SettlementID string    `json:"settlement_id" gorm:"size:32;not null;index"`
	DisputeType  string    `json:"dispute_type" gorm:"size:50"` // price_diff/quantity_diff/quality_deduction/other
	Description  string    `json:"description" gorm:"type:text"`
	AmountDiff   *float64  `json:"amount_diff" gorm:"type:decimal(12,2)"`
	Status       string    `json:"status" gorm:"size:20;default:open"` // open/resolved
	Resolution   string    `json:"resolution" gorm:"type:text"`
	CreatedAt    time.Time `json:"created_at"`
}

func (SettlementDispute) TableName() string {
	return "srm_settlement_disputes"
}
