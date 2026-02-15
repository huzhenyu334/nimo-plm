package entity

import "time"

// SupplierEvaluation 供应商评估
type SupplierEvaluation struct {
	ID         string `json:"id" gorm:"primaryKey;size:32"`
	SupplierID string `json:"supplier_id" gorm:"size:32;not null;index"`
	Period     string `json:"period" gorm:"size:20;not null"` // e.g. 2026-Q1, 2026-01
	EvalType   string `json:"eval_type" gorm:"size:20;default:quarterly"` // quarterly/monthly/annual

	// 评分（0-100）
	QualityScore  *float64 `json:"quality_score" gorm:"type:decimal(5,2)"`
	DeliveryScore *float64 `json:"delivery_score" gorm:"type:decimal(5,2)"`
	PriceScore    *float64 `json:"price_score" gorm:"type:decimal(5,2)"`
	ServiceScore  *float64 `json:"service_score" gorm:"type:decimal(5,2)"`
	TotalScore    *float64 `json:"total_score" gorm:"type:decimal(5,2)"`

	// 权重
	QualityWeight  float64 `json:"quality_weight" gorm:"type:decimal(3,2);default:0.30"`
	DeliveryWeight float64 `json:"delivery_weight" gorm:"type:decimal(3,2);default:0.25"`
	PriceWeight    float64 `json:"price_weight" gorm:"type:decimal(3,2);default:0.25"`
	ServiceWeight  float64 `json:"service_weight" gorm:"type:decimal(3,2);default:0.20"`

	// 等级
	Grade string `json:"grade" gorm:"size:10"` // A/B/C/D

	// 统计
	TotalPOs      int `json:"total_pos" gorm:"default:0"`
	OnTimePOs     int `json:"on_time_pos" gorm:"default:0"`
	QualityPassed int `json:"quality_passed" gorm:"default:0"`
	QualityTotal  int `json:"quality_total" gorm:"default:0"`

	// 其他
	Remarks     string `json:"remarks" gorm:"type:text"`
	EvaluatorID string `json:"evaluator_id" gorm:"size:32"`
	Status      string `json:"status" gorm:"size:20;default:draft"` // draft/submitted/approved

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 关联
	Supplier *Supplier `json:"supplier,omitempty" gorm:"foreignKey:SupplierID"`
}

func (SupplierEvaluation) TableName() string {
	return "srm_supplier_evaluations"
}

// 评估状态
const (
	EvalStatusDraft     = "draft"
	EvalStatusSubmitted = "submitted"
	EvalStatusApproved  = "approved"
)

// 评估等级
func CalcGrade(score float64) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 75:
		return "B"
	case score >= 60:
		return "C"
	default:
		return "D"
	}
}
