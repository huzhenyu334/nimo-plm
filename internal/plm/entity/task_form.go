package entity

import (
	"encoding/json"
	"time"
)

// TaskForm 任务表单定义
type TaskForm struct {
	ID          string          `json:"id" gorm:"primaryKey;size:32"`
	TaskID      string          `json:"task_id" gorm:"size:32;not null;uniqueIndex"`
	Name        string          `json:"name" gorm:"size:128;not null;default:'完成表单'"`
	Description string          `json:"description" gorm:"type:text"`
	Fields      json.RawMessage `json:"fields" gorm:"type:jsonb;not null;default:'[]'"`
	CreatedBy   string          `json:"created_by" gorm:"size:32;not null"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func (TaskForm) TableName() string {
	return "task_forms"
}

// TaskFormSubmission 表单提交记录
type TaskFormSubmission struct {
	ID          string          `json:"id" gorm:"primaryKey;size:32"`
	FormID      string          `json:"form_id" gorm:"size:32;not null"`
	TaskID      string          `json:"task_id" gorm:"size:32;not null"`
	Data        JSONB           `json:"data" gorm:"type:jsonb;not null;default:'{}'"`
	Files       json.RawMessage `json:"files" gorm:"type:jsonb;default:'[]'"`
	SubmittedBy string          `json:"submitted_by" gorm:"size:32;not null"`
	SubmittedAt time.Time       `json:"submitted_at"`
	Version     int             `json:"version" gorm:"not null;default:0"`
}

func (TaskFormSubmission) TableName() string {
	return "task_form_submissions"
}

// TemplateTaskForm 模板任务表单
type TemplateTaskForm struct {
	ID         string          `json:"id" gorm:"primaryKey;size:32"`
	TemplateID string          `json:"template_id" gorm:"size:36;not null;uniqueIndex:idx_template_task_code"`
	TaskCode   string          `json:"task_code" gorm:"size:64;not null;uniqueIndex:idx_template_task_code"`
	Name       string          `json:"name" gorm:"size:128;not null;default:'完成表单'"`
	Fields     json.RawMessage `json:"fields" gorm:"type:jsonb;not null;default:'[]'"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

func (TemplateTaskForm) TableName() string {
	return "template_task_forms"
}
