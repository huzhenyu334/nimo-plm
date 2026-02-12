package entity

import "time"

// SamplingRequest 打样请求
type SamplingRequest struct {
	ID           string     `json:"id" gorm:"primaryKey;size:32"`
	PRItemID     string     `json:"pr_item_id" gorm:"size:32;index"`
	Round        int        `json:"round"`                        // 打样轮次
	SupplierID   string     `json:"supplier_id" gorm:"size:32"`
	SupplierName string     `json:"supplier_name" gorm:"-"`       // 关联查询
	SampleQty    int        `json:"sample_qty"`                   // 样品数量
	Status       string     `json:"status" gorm:"size:20"`        // preparing/shipping/arrived/verifying/passed/failed
	RequestedBy  string     `json:"requested_by" gorm:"size:32"`
	ArrivedAt    *time.Time `json:"arrived_at"`
	VerifiedBy   string     `json:"verified_by" gorm:"size:32"`
	VerifiedAt   *time.Time `json:"verified_at"`
	VerifyResult string     `json:"verify_result" gorm:"size:20"` // passed/failed
	RejectReason string     `json:"reject_reason" gorm:"size:500"`
	ApprovalID   string     `json:"approval_id" gorm:"size:100"` // 飞书审批实例ID
	Notes        string     `json:"notes" gorm:"size:500"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (SamplingRequest) TableName() string {
	return "srm_sampling_requests"
}

// 打样状态
const (
	SamplingStatusPreparing = "preparing" // 供应商制样
	SamplingStatusShipping  = "shipping"  // 运输中
	SamplingStatusArrived   = "arrived"   // 已到货
	SamplingStatusVerifying = "verifying" // 验证中（已发飞书审批）
	SamplingStatusPassed    = "passed"    // 验证通过
	SamplingStatusFailed    = "failed"    // 验证不通过
)

// ValidSamplingTransitions 合法的打样状态流转
var ValidSamplingTransitions = map[string][]string{
	SamplingStatusPreparing: {SamplingStatusShipping},
	SamplingStatusShipping:  {SamplingStatusArrived},
	SamplingStatusArrived:   {SamplingStatusVerifying},
	SamplingStatusVerifying: {SamplingStatusPassed, SamplingStatusFailed},
}
