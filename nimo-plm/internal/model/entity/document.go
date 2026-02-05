package entity

import (
	"time"
)

// DocumentCategory 文档分类
type DocumentCategory struct {
	ID        string    `json:"id" gorm:"primaryKey;size:32"`
	Code      string    `json:"code" gorm:"size:32;not null;uniqueIndex"`
	Name      string    `json:"name" gorm:"size:64;not null"`
	ParentID  string    `json:"parent_id" gorm:"size:32"`
	SortOrder int       `json:"sort_order" gorm:"not null;default:0"`
	CreatedAt time.Time `json:"created_at"`

	// 关联
	Parent   *DocumentCategory  `json:"parent,omitempty" gorm:"foreignKey:ParentID"`
	Children []DocumentCategory `json:"children,omitempty" gorm:"foreignKey:ParentID"`
}

func (DocumentCategory) TableName() string {
	return "document_categories"
}

// Document 文档实体
type Document struct {
	ID             string     `json:"id" gorm:"primaryKey;size:32"`
	Code           string     `json:"code" gorm:"size:64;not null;uniqueIndex"`
	Title          string     `json:"title" gorm:"size:256;not null"`
	CategoryID     string     `json:"category_id" gorm:"size:32"`
	RelatedType    string     `json:"related_type" gorm:"size:32"`
	RelatedID      string     `json:"related_id" gorm:"size:32"`
	Status         string     `json:"status" gorm:"size:16;not null;default:draft"`
	Version        string     `json:"version" gorm:"size:16;not null;default:1.0"`
	Description    string     `json:"description" gorm:"type:text"`
	FileName       string     `json:"file_name" gorm:"size:256;not null"`
	FilePath       string     `json:"file_path" gorm:"size:512;not null"`
	FileSize       int64      `json:"file_size" gorm:"not null"`
	MimeType       string     `json:"mime_type" gorm:"size:128"`
	FeishuDocToken string     `json:"feishu_doc_token" gorm:"size:64"`
	FeishuDocURL   string     `json:"feishu_doc_url" gorm:"size:512"`
	UploadedBy     string     `json:"uploaded_by" gorm:"size:32;not null"`
	ReleasedBy     string     `json:"released_by" gorm:"size:32"`
	ReleasedAt     *time.Time `json:"released_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at" gorm:"index"`

	// 关联
	Category *DocumentCategory  `json:"category,omitempty" gorm:"foreignKey:CategoryID"`
	Uploader *User              `json:"uploader,omitempty" gorm:"foreignKey:UploadedBy"`
	Releaser *User              `json:"releaser,omitempty" gorm:"foreignKey:ReleasedBy"`
	Versions []DocumentVersion  `json:"versions,omitempty" gorm:"foreignKey:DocumentID"`
}

func (Document) TableName() string {
	return "documents"
}

// DocumentVersion 文档版本历史
type DocumentVersion struct {
	ID            string    `json:"id" gorm:"primaryKey;size:32"`
	DocumentID    string    `json:"document_id" gorm:"size:32;not null"`
	Version       string    `json:"version" gorm:"size:16;not null"`
	FileName      string    `json:"file_name" gorm:"size:256;not null"`
	FilePath      string    `json:"file_path" gorm:"size:512;not null"`
	FileSize      int64     `json:"file_size" gorm:"not null"`
	ChangeSummary string    `json:"change_summary" gorm:"type:text"`
	CreatedBy     string    `json:"created_by" gorm:"size:32;not null"`
	CreatedAt     time.Time `json:"created_at"`

	// 关联
	Document *Document `json:"document,omitempty" gorm:"foreignKey:DocumentID"`
	Creator  *User     `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
}

func (DocumentVersion) TableName() string {
	return "document_versions"
}

// TaskAttachment 任务附件
type TaskAttachment struct {
	ID         string    `json:"id" gorm:"primaryKey;size:32"`
	TaskID     string    `json:"task_id" gorm:"size:32;not null"`
	FileName   string    `json:"file_name" gorm:"size:256;not null"`
	FilePath   string    `json:"file_path" gorm:"size:512;not null"`
	FileSize   int64     `json:"file_size" gorm:"not null"`
	MimeType   string    `json:"mime_type" gorm:"size:128"`
	UploadedBy string    `json:"uploaded_by" gorm:"size:32;not null"`
	CreatedAt  time.Time `json:"created_at"`

	// 关联
	Task     *Task `json:"task,omitempty" gorm:"foreignKey:TaskID"`
	Uploader *User `json:"uploader,omitempty" gorm:"foreignKey:UploadedBy"`
}

func (TaskAttachment) TableName() string {
	return "task_attachments"
}

// OperationLog 操作日志
type OperationLog struct {
	ID            string    `json:"id" gorm:"primaryKey;size:32"`
	UserID        string    `json:"user_id" gorm:"size:32"`
	UserName      string    `json:"user_name" gorm:"size:64"`
	Module        string    `json:"module" gorm:"size:32;not null"`
	Action        string    `json:"action" gorm:"size:32;not null"`
	TargetType    string    `json:"target_type" gorm:"size:32"`
	TargetID      string    `json:"target_id" gorm:"size:32"`
	TargetName    string    `json:"target_name" gorm:"size:256"`
	Before        JSONB     `json:"before" gorm:"type:jsonb"`
	After         JSONB     `json:"after" gorm:"type:jsonb"`
	IP            string    `json:"ip" gorm:"size:64"`
	UserAgent     string    `json:"user_agent" gorm:"size:512"`
	CreatedAt     time.Time `json:"created_at"`

	// 关联
	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (OperationLog) TableName() string {
	return "operation_logs"
}

// SystemConfig 系统配置
type SystemConfig struct {
	ID          string    `json:"id" gorm:"primaryKey;size:32"`
	Key         string    `json:"key" gorm:"size:128;not null;uniqueIndex"`
	Value       string    `json:"value" gorm:"type:text;not null"`
	ValueType   string    `json:"value_type" gorm:"size:16;not null;default:string"`
	Module      string    `json:"module" gorm:"size:32"`
	Description string    `json:"description" gorm:"type:text"`
	IsPublic    bool      `json:"is_public" gorm:"not null;default:false"`
	UpdatedBy   string    `json:"updated_by" gorm:"size:32"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (SystemConfig) TableName() string {
	return "system_configs"
}

// CodeRule 编码规则
type CodeRule struct {
	ID          string    `json:"id" gorm:"primaryKey;size:32"`
	EntityType  string    `json:"entity_type" gorm:"size:32;not null;uniqueIndex"`
	Prefix      string    `json:"prefix" gorm:"size:16;not null"`
	Separator   string    `json:"separator" gorm:"size:4;not null;default:-"`
	DateFormat  string    `json:"date_format" gorm:"size:16"`
	SeqLength   int       `json:"seq_length" gorm:"not null;default:4"`
	CurrentSeq  int       `json:"current_seq" gorm:"not null;default:0"`
	ResetCycle  string    `json:"reset_cycle" gorm:"size:16"`
	Example     string    `json:"example" gorm:"size:64"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (CodeRule) TableName() string {
	return "code_rules"
}

// 文档状态常量
const (
	DocumentStatusDraft    = "draft"
	DocumentStatusReleased = "released"
	DocumentStatusObsolete = "obsolete"
)

// 文档关联类型常量
const (
	DocumentRelatedTypeProduct  = "product"
	DocumentRelatedTypeProject  = "project"
	DocumentRelatedTypeECN      = "ecn"
	DocumentRelatedTypeMaterial = "material"
)

// 文档分类ID常量
const (
	DocumentCategoryDesign  = "dcat_design"
	DocumentCategorySpec    = "dcat_spec"
	DocumentCategoryDrawing = "dcat_drawing"
	DocumentCategoryTest    = "dcat_test"
	DocumentCategoryQuality = "dcat_quality"
	DocumentCategoryOther   = "dcat_other"
)
