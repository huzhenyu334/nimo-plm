package entity

import (
	"time"
)

// Project 项目实体
type Project struct {
	ID              string     `json:"id" gorm:"primaryKey;size:32"`
	Code            string     `json:"code" gorm:"size:64;not null;uniqueIndex"`
	Name            string     `json:"name" gorm:"size:128;not null"`
	ProductID       string     `json:"product_id" gorm:"size:32"`
	Status          string     `json:"status" gorm:"size:16;not null;default:planning"`
	CurrentPhase    string     `json:"current_phase" gorm:"size:16;default:evt"`
	Description     string     `json:"description" gorm:"type:text"`
	OwnerID         string     `json:"owner_id" gorm:"size:32;not null"`
	PlannedStart    *time.Time `json:"planned_start" gorm:"type:date"`
	PlannedEnd      *time.Time `json:"planned_end" gorm:"type:date"`
	ActualStart     *time.Time `json:"actual_start" gorm:"type:date"`
	ActualEnd       *time.Time `json:"actual_end" gorm:"type:date"`
	Progress        int        `json:"progress" gorm:"not null;default:0"`
	FeishuProjectKey string    `json:"feishu_project_key" gorm:"size:64"`
	CreatedBy       string     `json:"created_by" gorm:"size:32;not null"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at" gorm:"index"`

	// 关联
	Product *Product       `json:"product,omitempty" gorm:"foreignKey:ProductID"`
	Owner   *User          `json:"owner,omitempty" gorm:"foreignKey:OwnerID"`
	Creator *User          `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
	Phases  []ProjectPhase `json:"phases,omitempty" gorm:"foreignKey:ProjectID"`
	Tasks   []Task         `json:"tasks,omitempty" gorm:"foreignKey:ProjectID"`
}

func (Project) TableName() string {
	return "projects"
}

// ProjectPhase 项目阶段
type ProjectPhase struct {
	ID            string     `json:"id" gorm:"primaryKey;size:32"`
	ProjectID     string     `json:"project_id" gorm:"size:32;not null"`
	Phase         string     `json:"phase" gorm:"size:16;not null"`
	Name          string     `json:"name" gorm:"size:64;not null"`
	Status        string     `json:"status" gorm:"size:16;not null;default:pending"`
	Sequence      int        `json:"sequence" gorm:"not null"`
	PlannedStart  *time.Time `json:"planned_start" gorm:"type:date"`
	PlannedEnd    *time.Time `json:"planned_end" gorm:"type:date"`
	ActualStart   *time.Time `json:"actual_start" gorm:"type:date"`
	ActualEnd     *time.Time `json:"actual_end" gorm:"type:date"`
	EntryCriteria JSONB      `json:"entry_criteria" gorm:"type:jsonb"`
	ExitCriteria  JSONB      `json:"exit_criteria" gorm:"type:jsonb"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	// 关联
	Project *Project `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	Tasks   []Task   `json:"tasks,omitempty" gorm:"foreignKey:PhaseID"`
}

func (ProjectPhase) TableName() string {
	return "project_phases"
}

// Task 任务实体
type Task struct {
	ID             string     `json:"id" gorm:"primaryKey;size:32"`
	ProjectID      string     `json:"project_id" gorm:"size:32;not null"`
	PhaseID        string     `json:"phase_id" gorm:"size:32"`
	ParentTaskID   string     `json:"parent_task_id" gorm:"size:32"`
	Code           string     `json:"code" gorm:"size:64"`
	Name           string     `json:"name" gorm:"size:256;not null"`
	Description    string     `json:"description" gorm:"type:text"`
	TaskType       string     `json:"task_type" gorm:"size:32;not null;default:task"`
	Status         string     `json:"status" gorm:"size:16;not null;default:pending"`
	Priority       string     `json:"priority" gorm:"size:16;not null;default:medium"`
	AssigneeID     string     `json:"assignee_id" gorm:"size:32"`
	ReviewerID     string     `json:"reviewer_id" gorm:"size:32"`
	PlannedStart   *time.Time `json:"planned_start" gorm:"type:date"`
	PlannedEnd     *time.Time `json:"planned_end" gorm:"type:date"`
	ActualStart    *time.Time `json:"actual_start" gorm:"type:date"`
	ActualEnd      *time.Time `json:"actual_end" gorm:"type:date"`
	DueDate        *time.Time `json:"due_date" gorm:"type:date"`
	Progress       int        `json:"progress" gorm:"not null;default:0"`
	EstimatedHours float64    `json:"estimated_hours" gorm:"type:decimal(8,2)"`
	ActualHours    float64    `json:"actual_hours" gorm:"type:decimal(8,2)"`
	FeishuTaskID   string     `json:"feishu_task_id" gorm:"size:64"`
	Sequence       int        `json:"sequence" gorm:"not null;default:0"`
	Level          int        `json:"level" gorm:"not null;default:0"`
	Path           string     `json:"path" gorm:"size:512"`
	CreatedBy      string     `json:"created_by" gorm:"size:32;not null"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	// 关联
	Project    *Project      `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	Phase      *ProjectPhase `json:"phase,omitempty" gorm:"foreignKey:PhaseID"`
	ParentTask *Task         `json:"parent_task,omitempty" gorm:"foreignKey:ParentTaskID"`
	Assignee   *User         `json:"assignee,omitempty" gorm:"foreignKey:AssigneeID"`
	Reviewer   *User         `json:"reviewer,omitempty" gorm:"foreignKey:ReviewerID"`
	Creator    *User         `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
	SubTasks   []Task        `json:"sub_tasks,omitempty" gorm:"foreignKey:ParentTaskID"`
}

func (Task) TableName() string {
	return "tasks"
}

// TaskDependency 任务依赖
type TaskDependency struct {
	ID               string    `json:"id" gorm:"primaryKey;size:32"`
	TaskID           string    `json:"task_id" gorm:"size:32;not null"`
	DependsOnTaskID  string    `json:"depends_on_task_id" gorm:"size:32;not null"`
	DependencyType   string    `json:"dependency_type" gorm:"size:16;not null;default:finish_to_start"`
	LagDays          int       `json:"lag_days" gorm:"default:0"`
	CreatedAt        time.Time `json:"created_at"`
}

func (TaskDependency) TableName() string {
	return "task_dependencies"
}

// TaskComment 任务评论
type TaskComment struct {
	ID        string    `json:"id" gorm:"primaryKey;size:32"`
	TaskID    string    `json:"task_id" gorm:"size:32;not null"`
	UserID    string    `json:"user_id" gorm:"size:32;not null"`
	Content   string    `json:"content" gorm:"type:text;not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 关联
	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (TaskComment) TableName() string {
	return "task_comments"
}

// ProjectStatus 项目状态
const (
	ProjectStatusPlanning  = "planning"
	ProjectStatusEVT       = "evt"
	ProjectStatusDVT       = "dvt"
	ProjectStatusPVT       = "pvt"
	ProjectStatusMP        = "mp"
	ProjectStatusCompleted = "completed"
	ProjectStatusCancelled = "cancelled"
)

// TaskStatus 任务状态
const (
	TaskStatusPending    = "pending"
	TaskStatusInProgress = "in_progress"
	TaskStatusCompleted  = "completed"
	TaskStatusBlocked    = "blocked"
	TaskStatusCancelled  = "cancelled"
)

// TaskType 任务类型
const (
	TaskTypeTask        = "task"
	TaskTypeMilestone   = "milestone"
	TaskTypeDeliverable = "deliverable"
)

// TaskPriority 任务优先级
const (
	TaskPriorityLow      = "low"
	TaskPriorityMedium   = "medium"
	TaskPriorityHigh     = "high"
	TaskPriorityCritical = "critical"
)
