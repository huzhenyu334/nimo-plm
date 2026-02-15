package entity

import "time"

// ProjectBOM 项目BOM（三级BOM: EBOM/PBOM/MBOM）
type ProjectBOM struct {
	ID            string     `json:"id" gorm:"primaryKey;size:32"`
	ProjectID     string     `json:"project_id" gorm:"size:32;not null"`
	PhaseID       *string    `json:"phase_id" gorm:"size:32"`
	TaskID        *string    `json:"task_id" gorm:"size:32;index"`
	BOMType       string     `json:"bom_type" gorm:"size:16;not null;default:EBOM"` // EBOM / PBOM / MBOM
	SourceBOMID   *string    `json:"source_bom_id,omitempty" gorm:"size:32"`        // 上游BOM ID (PBOM→EBOM, MBOM→EBOM)
	SourceVersion string     `json:"source_version,omitempty" gorm:"size:20"`       // 上游BOM版本号
	Version       string     `json:"version" gorm:"size:20;not null;default:''"`
	VersionMajor  int        `json:"version_major" gorm:"default:0"`
	VersionMinor  int        `json:"version_minor" gorm:"default:0"`
	Name          string     `json:"name" gorm:"size:128;not null"`
	Status        string     `json:"status" gorm:"size:16;not null;default:draft"` // draft/released/obsolete
	ReleaseNote   string     `json:"release_note,omitempty" gorm:"type:text;default:''"`
	ReleasedAt    *time.Time `json:"released_at,omitempty"`
	ReleasedBy    *string    `json:"released_by,omitempty" gorm:"size:32"`
	Description   string     `json:"description,omitempty"`
	SubmittedBy   *string    `json:"submitted_by,omitempty" gorm:"size:32"`
	SubmittedAt   *time.Time `json:"submitted_at,omitempty"`
	ReviewedBy    *string    `json:"reviewed_by,omitempty" gorm:"size:32"`
	ReviewedAt    *time.Time `json:"reviewed_at,omitempty"`
	ReviewComment string     `json:"review_comment,omitempty"`
	ApprovedBy    *string    `json:"approved_by,omitempty" gorm:"size:32"`
	ApprovedAt    *time.Time `json:"approved_at,omitempty"`
	FrozenAt      *time.Time `json:"frozen_at,omitempty"`
	FrozenBy      *string    `json:"frozen_by,omitempty" gorm:"size:32"`
	TotalItems    int        `json:"total_items" gorm:"default:0"`
	EstimatedCost *float64   `json:"estimated_cost,omitempty" gorm:"type:numeric(15,4)"`
	CreatedBy     string     `json:"created_by" gorm:"size:32;not null"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	// Relations
	Project   *Project         `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	Phase     *ProjectPhase    `json:"phase,omitempty" gorm:"foreignKey:PhaseID"`
	Items     []ProjectBOMItem `json:"items,omitempty" gorm:"foreignKey:BOMID"`
	Submitter *User            `json:"submitter,omitempty" gorm:"foreignKey:SubmittedBy"`
	Reviewer  *User            `json:"reviewer,omitempty" gorm:"foreignKey:ReviewedBy"`
	Creator   *User            `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
	SourceBOM *ProjectBOM      `json:"source_bom,omitempty" gorm:"foreignKey:SourceBOMID"`
}

func (ProjectBOM) TableName() string {
	return "project_boms"
}

// ProjectBOMItem BOM行项（精简固定列 + JSONB扩展属性）
type ProjectBOMItem struct {
	ID           string  `json:"id" gorm:"primaryKey;size:32"`
	BOMID        string  `json:"bom_id" gorm:"size:32;not null"`
	ItemNumber   int     `json:"item_number" gorm:"default:0"`
	ParentItemID *string `json:"parent_item_id,omitempty" gorm:"size:32"`
	Level        int     `json:"level" gorm:"not null;default:0"`
	MaterialID   *string `json:"material_id,omitempty" gorm:"size:32"`

	// 两层分类
	Category    string `json:"category" gorm:"size:32;not null"`     // 大类: electronic/structural/optical/packaging/tooling/consumable
	SubCategory string `json:"sub_category" gorm:"size:32;not null"` // 小类: component/pcb/connector/cable/housing/internal/fastener/lens/lightguide/light_engine/waveguide/box/document/cushion/mold/fixture/consumable

	// 通用固定列（7个显示列）
	Name         string   `json:"name" gorm:"size:128;not null"`
	Quantity     float64  `json:"quantity" gorm:"type:numeric(15,4);not null;default:1"`
	Unit         string   `json:"unit" gorm:"size:16;not null;default:pcs"`
	Supplier     string   `json:"supplier,omitempty" gorm:"size:128"`
	UnitPrice    *float64 `json:"unit_price,omitempty" gorm:"type:numeric(15,4)"`
	ExtendedCost *float64 `json:"extended_cost,omitempty" gorm:"type:numeric(15,4)"`
	Notes        string   `json:"notes,omitempty"`

	// 品类专属扩展属性（JSONB，按属性模板定义的字段存值）
	ExtendedAttrs JSONB `json:"extended_attrs,omitempty" gorm:"type:jsonb;default:'{}'"`

	// PBOM/MBOM专用
	ProcessStepID *string    `json:"process_step_id,omitempty" gorm:"size:32"`
	ScrapRate     *float64   `json:"scrap_rate,omitempty" gorm:"type:numeric(5,4)"`
	EffectiveDate *time.Time `json:"effective_date,omitempty"`
	ExpireDate    *time.Time `json:"expire_date,omitempty"`

	// 替代料
	IsAlternative  bool    `json:"is_alternative" gorm:"default:false"`
	AlternativeFor *string `json:"alternative_for,omitempty" gorm:"size:32"`

	// 附件
	Attachments  string `json:"attachments,omitempty" gorm:"type:jsonb;default:'[]'"`
	ThumbnailURL string `json:"thumbnail_url,omitempty" gorm:"size:512"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relations
	Material     *Material           `json:"material,omitempty" gorm:"foreignKey:MaterialID"`
	ParentItem   *ProjectBOMItem     `json:"parent_item,omitempty" gorm:"foreignKey:ParentItemID"`
	Children     []ProjectBOMItem    `json:"children,omitempty" gorm:"foreignKey:ParentItemID"`
	Drawings     []PartDrawing       `json:"drawings,omitempty" gorm:"foreignKey:BOMItemID"`
	CMFVariants  []BOMItemCMFVariant `json:"cmf_variants,omitempty" gorm:"foreignKey:BOMItemID"`
	LangVariants []BOMItemLangVariant `json:"lang_variants,omitempty" gorm:"foreignKey:BOMItemID"`
	ProcessStep  *ProcessStep        `json:"process_step,omitempty" gorm:"foreignKey:ProcessStepID"`
}

func (ProjectBOMItem) TableName() string {
	return "project_bom_items"
}

// CategoryAttrTemplate 品类属性模板（定义每个sub_category的扩展字段）
type CategoryAttrTemplate struct {
	ID           string    `json:"id" gorm:"primaryKey;size:32"`
	Category     string    `json:"category" gorm:"size:32;not null;index:idx_cat_sub"`
	SubCategory  string    `json:"sub_category" gorm:"size:32;not null;index:idx_cat_sub"`
	BOMType      string    `json:"bom_type" gorm:"size:16;not null;default:EBOM"` // EBOM/PBOM — which BOM type this category belongs to
	FieldKey     string    `json:"field_key" gorm:"size:64;not null"`
	FieldName    string    `json:"field_name" gorm:"size:64;not null"`
	FieldType    string    `json:"field_type" gorm:"size:16;not null"` // text/number/select/boolean/file/thumbnail
	Unit         string    `json:"unit,omitempty" gorm:"size:16"`
	Required     bool      `json:"required" gorm:"default:false"`
	Options      JSONB     `json:"options,omitempty" gorm:"type:jsonb"`    // select options {"values":["A","B"]}
	Validation   JSONB     `json:"validation,omitempty" gorm:"type:jsonb"` // {min,max,pattern}
	DefaultValue string    `json:"default_value,omitempty" gorm:"size:128"`
	SortOrder    int       `json:"sort_order" gorm:"default:0"`
	ShowInTable  bool      `json:"show_in_table" gorm:"default:true"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (CategoryAttrTemplate) TableName() string {
	return "category_attr_templates"
}

// ProcessRoute 工艺路线
type ProcessRoute struct {
	ID          string    `json:"id" gorm:"primaryKey;size:32"`
	ProjectID   string    `json:"project_id" gorm:"size:32;not null"`
	BOMID       string    `json:"bom_id" gorm:"size:32"`
	Name        string    `json:"name" gorm:"size:128;not null"`
	Version     string    `json:"version" gorm:"size:16;default:v1.0"`
	Status      string    `json:"status" gorm:"size:16;default:draft"` // draft/active/obsolete
	Description string    `json:"description" gorm:"type:text"`
	TotalSteps  int       `json:"total_steps" gorm:"default:0"`
	CreatedBy   string    `json:"created_by" gorm:"size:32;not null"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	Steps []ProcessStep `json:"steps,omitempty" gorm:"foreignKey:RouteID"`
}

func (ProcessRoute) TableName() string {
	return "process_routes"
}

// ProcessStep 工艺步骤
type ProcessStep struct {
	ID             string   `json:"id" gorm:"primaryKey;size:32"`
	RouteID        string   `json:"route_id" gorm:"size:32;not null"`
	StepNumber     int      `json:"step_number" gorm:"not null"`
	Name           string   `json:"name" gorm:"size:128;not null"`
	WorkCenter     string   `json:"work_center,omitempty" gorm:"size:64"`
	Description    string   `json:"description,omitempty" gorm:"type:text"`
	StdTimeMinutes float64  `json:"std_time_minutes" gorm:"type:numeric(10,2)"`
	SetupMinutes   float64  `json:"setup_minutes" gorm:"type:numeric(10,2)"`
	LaborCost      *float64 `json:"labor_cost,omitempty" gorm:"type:numeric(15,4)"`
	SortOrder      int      `json:"sort_order" gorm:"not null;default:0"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	Materials []ProcessStepMaterial `json:"materials,omitempty" gorm:"foreignKey:StepID"`
}

func (ProcessStep) TableName() string {
	return "process_steps"
}

// ProcessStepMaterial 工序-物料关联
type ProcessStepMaterial struct {
	ID         string    `json:"id" gorm:"primaryKey;size:32"`
	StepID     string    `json:"step_id" gorm:"size:32;not null"`
	MaterialID string    `json:"material_id" gorm:"size:32"`
	Name       string    `json:"name" gorm:"size:128"`
	Category   string    `json:"category" gorm:"size:32;not null"` // tooling/consumable/service
	Quantity   float64   `json:"quantity" gorm:"type:numeric(15,4);default:1"`
	Unit       string    `json:"unit" gorm:"size:16;default:pcs"`
	Notes      string    `json:"notes,omitempty" gorm:"type:text"`
	CreatedAt  time.Time `json:"created_at"`
}

func (ProcessStepMaterial) TableName() string {
	return "process_step_materials"
}

// BOMRelease BOM发布快照（ERP对接用）
type BOMRelease struct {
	ID           string     `json:"id" gorm:"primaryKey;size:36"`
	BOMID        string     `json:"bom_id" gorm:"size:32;not null"`
	ProjectID    string     `json:"project_id" gorm:"size:32;not null"`
	BOMType      string     `json:"bom_type" gorm:"size:16;not null"`
	Version      string     `json:"version" gorm:"size:16;not null"`
	SnapshotJSON string     `json:"snapshot_json" gorm:"type:jsonb;not null"`
	Status       string     `json:"status" gorm:"size:16;not null;default:pending"` // pending/synced/failed
	SyncedAt     *time.Time `json:"synced_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (BOMRelease) TableName() string {
	return "bom_releases"
}
