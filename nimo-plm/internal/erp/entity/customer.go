package entity

import (
	"time"
)

// CustomerType 客户类型
const (
	CustomerTypeRetail      = "RETAIL"       // 零售客户
	CustomerTypeWholesale   = "WHOLESALE"    // 批发客户
	CustomerTypeDistributor = "DISTRIBUTOR"  // 代理商
	CustomerTypeEnterprise  = "ENTERPRISE"   // 企业客户
)

// CustomerStatus 客户状态
const (
	CustomerStatusActive   = "ACTIVE"
	CustomerStatusInactive = "INACTIVE"
)

// Customer 客户实体
type Customer struct {
	ID           string     `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	CustomerCode string     `json:"customer_code" gorm:"size:50;not null;uniqueIndex"`
	Name         string     `json:"name" gorm:"size:200;not null"`
	Type         string     `json:"type" gorm:"size:20;not null;default:RETAIL"`
	ContactName  string     `json:"contact_name" gorm:"size:100"`
	Phone        string     `json:"phone" gorm:"size:20"`
	Email        string     `json:"email" gorm:"size:100"`
	Address      string     `json:"address" gorm:"size:500"`
	Channel      string     `json:"channel" gorm:"size:50"` // 销售渠道
	CreditLimit  float64    `json:"credit_limit" gorm:"type:decimal(12,2);default:0"`
	PaymentTerms string     `json:"payment_terms" gorm:"size:100"`
	Status       string     `json:"status" gorm:"size:20;not null;default:ACTIVE"`
	Notes        string     `json:"notes" gorm:"type:text"`
	CreatedBy    string     `json:"created_by" gorm:"size:64"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at" gorm:"index"`
}

func (Customer) TableName() string {
	return "erp_customers"
}
