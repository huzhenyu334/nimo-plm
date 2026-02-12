package entity

import "time"

// ProjectBOM 项目BOM（研发BOM，关联项目+阶段）
type ProjectBOM struct {
	ID            string     `json:"id" gorm:"primaryKey;size:32"`
	ProjectID     string     `json:"project_id" gorm:"size:32;not null"`
	PhaseID       *string    `json:"phase_id" gorm:"size:32"`
	TaskID        *string    `json:"task_id" gorm:"size:32;index"`
	BOMType       string     `json:"bom_type" gorm:"size:16;not null;default:EBOM"` // EBOM/SBOM/OBOM/FWBOM
	Version       string     `json:"version" gorm:"size:16;not null;default:v1.0"`
	Name          string     `json:"name" gorm:"size:128;not null"`
	Status        string     `json:"status" gorm:"size:16;not null;default:draft"` // draft/pending_review/published/frozen/rejected
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
	ParentBOMID   *string    `json:"parent_bom_id,omitempty" gorm:"size:32"`
	TotalItems    int        `json:"total_items" gorm:"default:0"`
	EstimatedCost *float64   `json:"estimated_cost,omitempty" gorm:"type:numeric(15,4)"`
	CreatedBy     string     `json:"created_by" gorm:"size:32;not null"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	// Relations
	Project   *Project       `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	Phase     *ProjectPhase  `json:"phase,omitempty" gorm:"foreignKey:PhaseID"`
	Items     []ProjectBOMItem `json:"items,omitempty" gorm:"foreignKey:BOMID"`
	Submitter *User          `json:"submitter,omitempty" gorm:"foreignKey:SubmittedBy"`
	Reviewer  *User          `json:"reviewer,omitempty" gorm:"foreignKey:ReviewedBy"`
	Creator   *User          `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
}

func (ProjectBOM) TableName() string {
	return "project_boms"
}

// ProjectBOMItem BOM行项
type ProjectBOMItem struct {
	ID              string   `json:"id" gorm:"primaryKey;size:32"`
	BOMID           string   `json:"bom_id" gorm:"size:32;not null"`
	ItemNumber      int      `json:"item_number" gorm:"default:0"`
	ParentItemID    *string  `json:"parent_item_id,omitempty" gorm:"size:32"`
	Level           int      `json:"level" gorm:"not null;default:0"`
	MaterialID      *string  `json:"material_id,omitempty" gorm:"size:32"`
	Category        string   `json:"category,omitempty" gorm:"size:32"`
	Name            string   `json:"name" gorm:"size:128;not null"`
	Specification   string   `json:"specification,omitempty"`
	Quantity        float64  `json:"quantity" gorm:"type:numeric(15,4);not null;default:1"`
	Unit            string   `json:"unit" gorm:"size:16;not null;default:pcs"`
	Reference       string   `json:"reference,omitempty" gorm:"size:256"`
	Manufacturer    string   `json:"manufacturer,omitempty" gorm:"size:128"`
	ManufacturerPN  string   `json:"manufacturer_pn,omitempty" gorm:"size:64"`
	Supplier        string   `json:"supplier,omitempty" gorm:"size:128"`
	SupplierPN      string   `json:"supplier_pn,omitempty" gorm:"size:64"`
	UnitPrice       *float64 `json:"unit_price,omitempty" gorm:"type:numeric(15,4)"`
	ExtendedCost    *float64 `json:"extended_cost,omitempty" gorm:"type:numeric(15,4)"`
	LeadTimeDays    *int     `json:"lead_time_days,omitempty"`
	ProcurementType string   `json:"procurement_type" gorm:"size:16;not null;default:buy"`
	MOQ             *int     `json:"moq,omitempty"`
	ApprovedVendors *string  `json:"approved_vendors,omitempty" gorm:"type:jsonb"`
	LifecycleStatus string   `json:"lifecycle_status,omitempty" gorm:"size:16;default:active"`
	IsCritical      bool     `json:"is_critical" gorm:"default:false"`
	IsAlternative   bool     `json:"is_alternative" gorm:"default:false"`
	AlternativeFor  *string  `json:"alternative_for,omitempty" gorm:"size:32"`
	Notes           string   `json:"notes,omitempty"`

	// 结构BOM专属字段
	MaterialType     string   `json:"material_type,omitempty" gorm:"size:64"`          // 材质：PC, ABS, PA66+GF30, 铝合金6061, 不锈钢304, 硅胶
	Color            string   `json:"color,omitempty" gorm:"size:64"`                  // 颜色/外观：磨砂黑, Pantone Black 6C
	SurfaceTreatment string   `json:"surface_treatment,omitempty" gorm:"size:128"`     // 表面处理：阳极氧化, 喷涂, 电镀, 丝印, UV转印, PVD
	ProcessType      string   `json:"process_type,omitempty" gorm:"size:32"`           // 工艺类型：注塑, CNC, 冲压, 模切, 3D打印, 激光切割
	DrawingNo        string   `json:"drawing_no,omitempty" gorm:"size:64"`             // 图纸编号
	// Deprecated: use PartDrawing table instead
	Drawing2DFileID  *string  `json:"drawing_2d_file_id,omitempty" gorm:"column:drawing2d_file_id;size:32"`     // 2D工程图文件ID
	// Deprecated: use PartDrawing table instead
	Drawing2DFileName string  `json:"drawing_2d_file_name,omitempty" gorm:"column:drawing2d_file_name;size:256"`  // 2D文件名
	// Deprecated: use PartDrawing table instead
	Drawing3DFileID  *string  `json:"drawing_3d_file_id,omitempty" gorm:"column:drawing3d_file_id;size:32"`     // 3D模型文件ID
	// Deprecated: use PartDrawing table instead
	Drawing3DFileName string  `json:"drawing_3d_file_name,omitempty" gorm:"column:drawing3d_file_name;size:256"`  // 3D文件名
	WeightGrams      *float64 `json:"weight_grams,omitempty" gorm:"type:numeric(10,2)"` // 重量(克)
	TargetPrice      *float64 `json:"target_price,omitempty" gorm:"type:numeric(15,4)"` // 目标单价
	ToolingEstimate  *float64 `json:"tooling_estimate,omitempty" gorm:"type:numeric(15,2)"` // 模具费预估
	CostNotes        string   `json:"cost_notes,omitempty" gorm:"type:text"`           // 成本备注
	IsAppearancePart bool     `json:"is_appearance_part" gorm:"default:false"`          // 是否外观件
	AssemblyMethod   string   `json:"assembly_method,omitempty" gorm:"size:32"`        // 装配方式：卡扣, 螺丝, 胶合, 超声波焊接, 热熔
	ToleranceGrade   string   `json:"tolerance_grade,omitempty" gorm:"size:32"`        // 公差等级：普通/精密/超精密
	IsVariant        bool     `json:"is_variant" gorm:"default:false"`                // 标记该件在SKU间可能不同

	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Relations
	Material   *Material        `json:"material,omitempty" gorm:"foreignKey:MaterialID"`
	ParentItem *ProjectBOMItem  `json:"parent_item,omitempty" gorm:"foreignKey:ParentItemID"`
	Children   []ProjectBOMItem `json:"children,omitempty" gorm:"foreignKey:ParentItemID"`
	Drawings   []PartDrawing    `json:"drawings,omitempty" gorm:"foreignKey:BOMItemID"`
}

func (ProjectBOMItem) TableName() string {
	return "project_bom_items"
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
