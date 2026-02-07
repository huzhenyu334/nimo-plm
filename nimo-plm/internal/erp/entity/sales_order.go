package entity

import (
	"time"
)

// SalesOrderStatus 销售订单状态
const (
	SOStatusPending   = "PENDING"
	SOStatusConfirmed = "CONFIRMED"
	SOStatusPicking   = "PICKING"
	SOStatusShipped   = "SHIPPED"
	SOStatusDelivered = "DELIVERED"
	SOStatusCompleted = "COMPLETED"
	SOStatusCancelled = "CANCELLED"
)

// SalesChannel 销售渠道
const (
	ChannelDirect    = "DIRECT"     // 官网直销
	ChannelEcommerce = "ECOMMERCE"  // 电商平台
	ChannelDealer    = "DEALER"     // 线下代理
	ChannelRetail    = "RETAIL"     // 线下门店
)

// SalesOrder 销售订单
type SalesOrder struct {
	ID              string     `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	SOCode          string     `json:"so_code" gorm:"size:50;not null;uniqueIndex"`
	CustomerID      string     `json:"customer_id" gorm:"type:uuid;not null;index"`
	Channel         string     `json:"channel" gorm:"size:20;not null;default:DIRECT"`
	Status          string     `json:"status" gorm:"size:20;not null;default:PENDING"`
	TotalAmount     float64    `json:"total_amount" gorm:"type:decimal(12,2);default:0"`
	Currency        string     `json:"currency" gorm:"size:10;not null;default:CNY"`
	OrderDate       *time.Time `json:"order_date"`
	ShippingDate    *time.Time `json:"shipping_date"`
	DeliveredDate   *time.Time `json:"delivered_date"`
	ShippingAddress string     `json:"shipping_address" gorm:"size:500"`
	TrackingNo      string     `json:"tracking_no" gorm:"size:100"`
	Notes           string     `json:"notes" gorm:"type:text"`
	CreatedBy       string     `json:"created_by" gorm:"size:64;not null"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at" gorm:"index"`

	Customer *Customer   `json:"customer,omitempty" gorm:"foreignKey:CustomerID"`
	Items    []SOItem    `json:"items,omitempty" gorm:"foreignKey:SOID"`
}

func (SalesOrder) TableName() string {
	return "erp_sales_orders"
}

// SOItemStatus 销售订单行状态
const (
	SOItemStatusOpen     = "OPEN"
	SOItemStatusPicking  = "PICKING"
	SOItemStatusShipped  = "SHIPPED"
	SOItemStatusClosed   = "CLOSED"
)

// SOItem 销售订单明细
type SOItem struct {
	ID          string     `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	SOID        string     `json:"so_id" gorm:"type:uuid;not null;index"`
	ProductID   string     `json:"product_id" gorm:"size:32;not null"`
	ProductCode string     `json:"product_code" gorm:"size:64"`
	ProductName string     `json:"product_name" gorm:"size:128"`
	Quantity    float64    `json:"quantity" gorm:"type:decimal(12,4);not null"`
	UnitPrice   float64    `json:"unit_price" gorm:"type:decimal(12,4);not null"`
	Amount      float64    `json:"amount" gorm:"type:decimal(12,2);not null"`
	ShippedQty  float64    `json:"shipped_qty" gorm:"type:decimal(12,4);default:0"`
	Status      string     `json:"status" gorm:"size:20;not null;default:OPEN"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	SalesOrder *SalesOrder `json:"sales_order,omitempty" gorm:"foreignKey:SOID"`
}

func (SOItem) TableName() string {
	return "erp_so_items"
}
