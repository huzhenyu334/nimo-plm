package entity

import (
	"encoding/json"
	"time"
)

// ProjectTemplate 项目模板
type ProjectTemplate struct {
	ID               string          `json:"id" gorm:"primaryKey;size:36"`
	Code             string          `json:"code" gorm:"uniqueIndex;size:50;not null"`
	Name             string          `json:"name" gorm:"size:200;not null"`
	Description      string          `json:"description" gorm:"type:text"`
	TemplateType     string          `json:"template_type" gorm:"size:20;not null;default:'CUSTOM'"` // SYSTEM/CUSTOM
	ProductType      string          `json:"product_type" gorm:"size:50"`
	Phases           json.RawMessage `json:"phases" gorm:"type:jsonb"`
	EstimatedDays    int             `json:"estimated_days"`
	IsActive         bool            `json:"is_active" gorm:"default:true"`
	ParentTemplateID *string         `json:"parent_template_id" gorm:"size:36"`
	Version          string          `json:"version" gorm:"size:20;default:'1.0'"`
	Status           string          `json:"status" gorm:"size:20;default:'draft'"` // draft/published
	PublishedAt      *time.Time      `json:"published_at"`
	BaseCode         string          `json:"base_code" gorm:"size:50"` // 同一流程的不同版本共享base_code
	CreatedBy        string          `json:"created_by" gorm:"size:64;not null"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`

	// 关联
	Tasks        []TemplateTask `json:"tasks,omitempty" gorm:"foreignKey:TemplateID"`
	Dependencies []TemplateTaskDependency `json:"dependencies,omitempty" gorm:"foreignKey:TemplateID"`
}

func (ProjectTemplate) TableName() string {
	return "project_templates"
}

// TemplateTask 模板任务
type TemplateTask struct {
	ID                    string          `json:"id" gorm:"primaryKey;size:36"`
	TemplateID            string          `json:"template_id" gorm:"size:36;not null;index"`
	TaskCode              string          `json:"task_code" gorm:"size:50;not null"`
	Name                  string          `json:"name" gorm:"size:200;not null"`
	Description           string          `json:"description" gorm:"type:text"`
	Phase                 string          `json:"phase" gorm:"size:20;not null"`
	ParentTaskCode        string          `json:"parent_task_code" gorm:"size:50"`
	TaskType              string          `json:"task_type" gorm:"size:20;default:'TASK'"` // MILESTONE/TASK/SUBTASK
	DefaultAssigneeRole   string          `json:"default_assignee_role" gorm:"size:50"`
	EstimatedDays         int             `json:"estimated_days" gorm:"default:1"`
	IsCritical            bool            `json:"is_critical" gorm:"default:false"`
	Deliverables          json.RawMessage `json:"deliverables" gorm:"type:jsonb"`
	Checklist             json.RawMessage `json:"checklist" gorm:"type:jsonb"`
	RequiresApproval      bool            `json:"requires_approval" gorm:"default:false"`
	ApprovalType          string          `json:"approval_type" gorm:"size:50"`
	AutoCreateFeishuTask  bool            `json:"auto_create_feishu_task" gorm:"default:false"`
	FeishuApprovalCode    string          `json:"feishu_approval_code" gorm:"size:100"`
	SortOrder             int             `json:"sort_order" gorm:"default:0"`
	IsLocked              bool            `json:"is_locked" gorm:"default:false"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`

	// 关联
	SubTasks     []TemplateTask            `json:"sub_tasks,omitempty" gorm:"-"`
	Dependencies []TemplateTaskDependency  `json:"dependencies,omitempty" gorm:"-"`
}

func (TemplateTask) TableName() string {
	return "template_tasks"
}

// TemplateTaskDependency 模板任务依赖
type TemplateTaskDependency struct {
	ID                 string `json:"id" gorm:"primaryKey;size:36"`
	TemplateID         string `json:"template_id" gorm:"size:36;not null;index"`
	TaskCode           string `json:"task_code" gorm:"size:50;not null"`
	DependsOnTaskCode  string `json:"depends_on_task_code" gorm:"size:50;not null"`
	DependencyType     string `json:"dependency_type" gorm:"size:10;default:'FS'"` // FS/SS/FF/SF
	LagDays            int    `json:"lag_days" gorm:"default:0"`
}

func (TemplateTaskDependency) TableName() string {
	return "template_task_dependencies"
}

// FeishuTaskSync 飞书任务同步记录
type FeishuTaskSync struct {
	ID            string     `json:"id" gorm:"primaryKey;size:36"`
	TaskID        string     `json:"task_id" gorm:"size:32;uniqueIndex;not null"`
	FeishuTaskID  string     `json:"feishu_task_id" gorm:"size:100;uniqueIndex;not null"`
	FeishuTaskGUID string    `json:"feishu_task_guid" gorm:"size:100"`
	SyncStatus    string     `json:"sync_status" gorm:"size:20;default:'SYNCED'"`
	LastSyncAt    *time.Time `json:"last_sync_at"`
	SyncError     string     `json:"sync_error" gorm:"type:text"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (FeishuTaskSync) TableName() string {
	return "feishu_task_sync"
}

// NOTE: ApprovalDefinition 已移动到 approval_definition.go

// ApprovalInstance 审批实例
type ApprovalInstance struct {
	ID                  string          `json:"id" gorm:"primaryKey;size:36"`
	ApprovalDefID       *string         `json:"approval_def_id" gorm:"size:36"`
	FeishuInstanceCode  string          `json:"feishu_instance_code" gorm:"size:100;uniqueIndex"`
	BusinessType        string          `json:"business_type" gorm:"size:20;not null"`
	BusinessID          string          `json:"business_id" gorm:"size:64;not null;index"`
	Status              string          `json:"status" gorm:"size:20;default:'PENDING'"` // PENDING/APPROVED/REJECTED/CANCELLED
	ApplicantID         string          `json:"applicant_id" gorm:"size:64;not null"`
	FormData            json.RawMessage `json:"form_data" gorm:"type:jsonb"`
	Approvers           json.RawMessage `json:"approvers" gorm:"type:jsonb"`
	CurrentApproverID   string          `json:"current_approver_id" gorm:"size:64"`
	Comments            string          `json:"comments" gorm:"type:text"`
	CreatedAt           time.Time       `json:"created_at"`
	CompletedAt         *time.Time      `json:"completed_at"`
}

func (ApprovalInstance) TableName() string {
	return "approval_instances"
}

// ReviewMeeting 评审会议
type ReviewMeeting struct {
	ID                     string          `json:"id" gorm:"primaryKey;size:36"`
	Title                  string          `json:"title" gorm:"size:200;not null"`
	MeetingType            string          `json:"meeting_type" gorm:"size:20;not null"` // DESIGN/PHASE/BOM/ECN
	ProjectID              string          `json:"project_id" gorm:"size:32;index"`
	TaskID                 string          `json:"task_id" gorm:"size:32;index"`
	FeishuCalendarEventID  string          `json:"feishu_calendar_event_id" gorm:"size:100"`
	FeishuMeetingID        string          `json:"feishu_meeting_id" gorm:"size:100"`
	ScheduledAt            time.Time       `json:"scheduled_at" gorm:"not null"`
	DurationMinutes        int             `json:"duration_minutes" gorm:"default:60"`
	Location               string          `json:"location" gorm:"size:200"`
	OrganizerID            string          `json:"organizer_id" gorm:"size:64;not null"`
	Attendees              json.RawMessage `json:"attendees" gorm:"type:jsonb"`
	Agenda                 string          `json:"agenda" gorm:"type:text"`
	Documents              json.RawMessage `json:"documents" gorm:"type:jsonb"`
	Status                 string          `json:"status" gorm:"size:20;default:'SCHEDULED'"` // SCHEDULED/IN_PROGRESS/COMPLETED/CANCELLED
	Conclusion             string          `json:"conclusion" gorm:"type:text"`
	ActionItems            json.RawMessage `json:"action_items" gorm:"type:jsonb"`
	MinutesDocID           *string         `json:"minutes_doc_id" gorm:"size:36"`
	CreatedAt              time.Time       `json:"created_at"`
	UpdatedAt              time.Time       `json:"updated_at"`
}

func (ReviewMeeting) TableName() string {
	return "review_meetings"
}

// AutomationRule 自动化规则
type AutomationRule struct {
	ID               string          `json:"id" gorm:"primaryKey;size:36"`
	RuleType         string          `json:"rule_type" gorm:"size:30;not null;index"` // TASK_START/TASK_COMPLETE/OVERDUE/PHASE_COMPLETE
	TriggerCondition json.RawMessage `json:"trigger_condition" gorm:"type:jsonb;not null"`
	ActionType       string          `json:"action_type" gorm:"size:30;not null"` // UPDATE_STATUS/SEND_NOTIFICATION/CREATE_TASK/CREATE_APPROVAL
	ActionConfig     json.RawMessage `json:"action_config" gorm:"type:jsonb;not null"`
	IsActive         bool            `json:"is_active" gorm:"default:true;index"`
	ProjectID        string          `json:"project_id" gorm:"size:32;index"`
	TemplateID       *string         `json:"template_id" gorm:"size:36"`
	Priority         int             `json:"priority" gorm:"default:0"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

func (AutomationRule) TableName() string {
	return "automation_rules"
}

// AutomationLog 自动化执行日志
type AutomationLog struct {
	ID           string          `json:"id" gorm:"primaryKey;size:36"`
	RuleID       *string         `json:"rule_id" gorm:"size:36;index"`
	TriggerEvent json.RawMessage `json:"trigger_event" gorm:"type:jsonb;not null"`
	ActionResult json.RawMessage `json:"action_result" gorm:"type:jsonb"`
	Status       string          `json:"status" gorm:"size:20;not null"` // SUCCESS/FAILED/SKIPPED
	ErrorMessage string          `json:"error_message" gorm:"type:text"`
	ExecutedAt   time.Time       `json:"executed_at"`
}

func (AutomationLog) TableName() string {
	return "automation_logs"
}
