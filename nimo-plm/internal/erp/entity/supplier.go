package entity

import (
	"time"
)

// SupplierType 供应商类型
const (
	SupplierTypeRawMaterial = "RAW_MATERIAL"  // 原材料
	SupplierTypeComponent   = "COMPONENT"     // 元器件
	SupplierTypeStructural  = "STRUCTURAL"    // 结构件
	SupplierTypePackaging   = "PACKAGING"     // 包装
	SupplierTypeOther       = "OTHER"         // 其他
)

// SupplierRating 供应商评级
const (
	SupplierRatingA = "A"
	SupplierRatingB = "B"
	SupplierRatingC = "C"
	SupplierRatingD = "D"
)

// SupplierStatus 供应商状态
const (
	SupplierStatusActive    = "ACTIVE"
	SupplierStatusInactive  = "INACTIVE"
	SupplierStatusBlacklist = "BLACKLIST"
)

// Supplier 供应商实体
type Supplier struct {
	ID            string     `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	SupplierCode  string     `json:"supplier_code" gorm:"size:50;not null;uniqueIndex"`
	Name          string     `json:"name" gorm:"size:200;not null"`
	Type          string     `json:"type" gorm:"size:20;not null"`
	ContactName   string     `json:"contact_name" gorm:"size:100;not null"`
	Phone         string     `json:"phone" gorm:"size:20;not null"`
	Email         string     `json:"email" gorm:"size:100"`
	Address       string     `json:"address" gorm:"size:500;not null"`
	PaymentTerms  string     `json:"payment_terms" gorm:"size:100"`
	Rating        string     `json:"rating" gorm:"size:1"`
	Status        string     `json:"status" gorm:"size:20;not null;default:ACTIVE"`
	QualityScore  float64    `json:"quality_score" gorm:"type:decimal(5,2);default:0"`
	DeliveryScore float64    `json:"delivery_score" gorm:"type:decimal(5,2);default:0"`
	PriceScore    float64    `json:"price_score" gorm:"type:decimal(5,2);default:0"`
	ServiceScore  float64    `json:"service_score" gorm:"type:decimal(5,2);default:0"`
	OverallScore  float64    `json:"overall_score" gorm:"type:decimal(5,2);default:0"`
	Notes         string     `json:"notes" gorm:"type:text"`
	CreatedBy     string     `json:"created_by" gorm:"size:64"`
	UpdatedBy     string     `json:"updated_by" gorm:"size:64"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at" gorm:"index"`
}

func (Supplier) TableName() string {
	return "erp_suppliers"
}

// CalculateOverallScore 计算综合评分
func (s *Supplier) CalculateOverallScore() {
	s.OverallScore = s.QualityScore*0.4 + s.DeliveryScore*0.3 + s.PriceScore*0.2 + s.ServiceScore*0.1
}

// DetermineRating 根据综合评分确定评级
func (s *Supplier) DetermineRating() {
	s.CalculateOverallScore()
	switch {
	case s.OverallScore >= 90:
		s.Rating = SupplierRatingA
	case s.OverallScore >= 75:
		s.Rating = SupplierRatingB
	case s.OverallScore >= 60:
		s.Rating = SupplierRatingC
	default:
		s.Rating = SupplierRatingD
	}
}
