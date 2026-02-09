package entity

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// JSONB JSONB类型
type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan JSONB: %v", value)
	}
	return json.Unmarshal(bytes, j)
}

// JSONBArray JSONB数组类型
type JSONBArray []interface{}

func (j JSONBArray) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONBArray) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan JSONBArray: %v", value)
	}
	return json.Unmarshal(bytes, j)
}

// Supplier 供应商
type Supplier struct {
	ID        string    `json:"id" gorm:"primaryKey;size:32"`
	Code      string    `json:"code" gorm:"size:32;uniqueIndex;not null"`
	Name      string    `json:"name" gorm:"size:200;not null"`
	ShortName string    `json:"short_name" gorm:"size:50"`
	Category  string    `json:"category" gorm:"size:50;not null"` // structural/electronic/optical/packaging/other
	Level     string    `json:"level" gorm:"size:20;default:potential"`
	Status    string    `json:"status" gorm:"size:20;default:pending"`

	// 基本信息
	Country  string `json:"country" gorm:"size:50"`
	Province string `json:"province" gorm:"size:50"`
	City     string `json:"city" gorm:"size:50"`
	Address  string `json:"address" gorm:"size:500"`
	Website  string `json:"website" gorm:"size:200"`

	// 业务信息
	BusinessScope  string     `json:"business_scope" gorm:"type:text"`
	AnnualRevenue  *float64   `json:"annual_revenue" gorm:"type:decimal(15,2)"`
	EmployeeCount  *int       `json:"employee_count"`
	FactoryArea    *float64   `json:"factory_area" gorm:"type:decimal(10,2)"`
	Certifications *JSONBArray `json:"certifications" gorm:"type:jsonb"`

	// 付款信息
	BankName     string `json:"bank_name" gorm:"size:200"`
	BankAccount  string `json:"bank_account" gorm:"size:50"`
	TaxID        string `json:"tax_id" gorm:"size:50"`
	PaymentTerms string `json:"payment_terms" gorm:"size:100"`

	// 360画像标签
	Tags            *JSONBArray `json:"tags" gorm:"type:jsonb"`
	TechCapability  string      `json:"tech_capability" gorm:"size:20"`
	Cooperation     string      `json:"cooperation" gorm:"size:20"`
	CapacityLimit   string      `json:"capacity_limit" gorm:"size:200"`

	// 绩效指标
	QualityScore  *float64 `json:"quality_score" gorm:"type:decimal(5,2)"`
	DeliveryScore *float64 `json:"delivery_score" gorm:"type:decimal(5,2)"`
	PriceScore    *float64 `json:"price_score" gorm:"type:decimal(5,2)"`
	OverallScore  *float64 `json:"overall_score" gorm:"type:decimal(5,2)"`

	// 管理信息
	CreatedBy  string     `json:"created_by" gorm:"size:32"`
	ApprovedBy *string    `json:"approved_by" gorm:"size:32"`
	ApprovedAt *time.Time `json:"approved_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Notes      string     `json:"notes" gorm:"type:text"`

	// 关联
	Contacts []SupplierContact  `json:"contacts,omitempty" gorm:"foreignKey:SupplierID"`
	Materials []SupplierMaterial `json:"materials,omitempty" gorm:"foreignKey:SupplierID"`
}

func (Supplier) TableName() string {
	return "srm_suppliers"
}

// 供应商分类
const (
	SupplierCategoryStructural = "structural"
	SupplierCategoryElectronic = "electronic"
	SupplierCategoryOptical    = "optical"
	SupplierCategoryPackaging  = "packaging"
	SupplierCategoryOther      = "other"
)

// 供应商等级
const (
	SupplierLevelPotential = "potential"
	SupplierLevelQualified = "qualified"
	SupplierLevelPreferred = "preferred"
	SupplierLevelStrategic = "strategic"
)

// 供应商状态
const (
	SupplierStatusPending     = "pending"
	SupplierStatusActive      = "active"
	SupplierStatusSuspended   = "suspended"
	SupplierStatusBlacklisted = "blacklisted"
)

// SupplierContact 供应商联系人
type SupplierContact struct {
	ID         string    `json:"id" gorm:"primaryKey;size:32"`
	SupplierID string    `json:"supplier_id" gorm:"size:32;not null;index"`
	Name       string    `json:"name" gorm:"size:100;not null"`
	Title      string    `json:"title" gorm:"size:100"`
	Phone      string    `json:"phone" gorm:"size:50"`
	Email      string    `json:"email" gorm:"size:200"`
	Wechat     string    `json:"wechat" gorm:"size:100"`
	IsPrimary  bool      `json:"is_primary" gorm:"default:false"`
	CreatedAt  time.Time `json:"created_at"`
}

func (SupplierContact) TableName() string {
	return "srm_supplier_contacts"
}

// SupplierMaterial 供应商可供物料
type SupplierMaterial struct {
	ID           string   `json:"id" gorm:"primaryKey;size:32"`
	SupplierID   string   `json:"supplier_id" gorm:"size:32;not null;index"`
	CategoryID   *string  `json:"category_id" gorm:"size:32"`
	MaterialID   *string  `json:"material_id" gorm:"size:32"`
	LeadTimeDays *int     `json:"lead_time_days"`
	MOQ          *int     `json:"moq"`
	UnitPrice    *float64 `json:"unit_price" gorm:"type:decimal(12,4)"`
	Currency     string   `json:"currency" gorm:"size:10;default:CNY"`
	Notes        string   `json:"notes" gorm:"type:text"`
	CreatedAt    time.Time `json:"created_at"`
}

func (SupplierMaterial) TableName() string {
	return "srm_supplier_materials"
}
