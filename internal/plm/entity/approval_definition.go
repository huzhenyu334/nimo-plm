package entity

import (
	"encoding/json"
	"time"
)

// ApprovalDefinition 审批定义（模板）
type ApprovalDefinition struct {
	ID          string          `json:"id" gorm:"primaryKey;size:36"`
	Code        string          `json:"code" gorm:"size:50;uniqueIndex;not null"`
	Name        string          `json:"name" gorm:"size:200;not null"`
	Description string          `json:"description" gorm:"type:text"`
	Icon        string          `json:"icon" gorm:"size:50;default:'approval'"`
	GroupName   string          `json:"group_name" gorm:"size:50;not null;default:'其他'"`
	FormSchema  json.RawMessage `json:"form_schema" gorm:"type:jsonb;not null;default:'[]'"`
	FlowSchema  json.RawMessage `json:"flow_schema" gorm:"type:jsonb;not null;default:'{\"nodes\":[]}'"`
	Visibility  string          `json:"visibility" gorm:"size:200;default:'全员'"`
	Status      string          `json:"status" gorm:"size:20;not null;default:'draft'"`
	AdminUserID string          `json:"admin_user_id" gorm:"size:32"`
	SortOrder   int             `json:"sort_order" gorm:"default:0"`
	CreatedBy   string          `json:"created_by" gorm:"size:32;not null"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func (ApprovalDefinition) TableName() string {
	return "approval_definitions"
}

// ApprovalGroup 审批分组
type ApprovalGroup struct {
	ID        string    `json:"id" gorm:"primaryKey;size:36"`
	Name      string    `json:"name" gorm:"size:50;uniqueIndex;not null"`
	SortOrder int       `json:"sort_order" gorm:"default:0"`
	CreatedAt time.Time `json:"created_at"`
}

func (ApprovalGroup) TableName() string {
	return "approval_groups"
}

// ApprovalDefinition status constants
const (
	ApprovalDefStatusDraft     = "draft"
	ApprovalDefStatusPublished = "published"
)

// FlowSchema 流程定义
type FlowSchema struct {
	Nodes []FlowNode `json:"nodes"`
}

// FlowNode 流程节点
type FlowNode struct {
	Type   string         `json:"type"` // submit, approve, end
	Name   string         `json:"name"`
	Config FlowNodeConfig `json:"config"`
}

// FlowNodeConfig 节点配置
type FlowNodeConfig struct {
	// submit node
	Submitter string   `json:"submitter,omitempty"`
	CCUsers   []string `json:"cc_users,omitempty"`
	// approve node
	ApproverType string   `json:"approver_type,omitempty"` // designated, self_select, supervisor, dept_leader, submitter, role
	ApproverIDs  []string `json:"approver_ids,omitempty"`
	MultiApprove string   `json:"multi_approve,omitempty"` // all, any, sequential
	WhenSelf     string   `json:"when_self,omitempty"`
	SelectRange  string   `json:"select_range,omitempty"`
}
