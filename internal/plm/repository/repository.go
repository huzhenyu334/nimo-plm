package repository

import (
	"errors"

	"gorm.io/gorm"
)

// 错误定义
var (
	ErrNotFound = errors.New("record not found")
)

// Repositories 仓库集合
type Repositories struct {
	User            *UserRepository
	Product         *ProductRepository
	ProductCategory *ProductCategoryRepository
	Material        *MaterialRepository
	MaterialCategory *MaterialCategoryRepository
	BOM             *BOMRepository
	Project         *ProjectRepository
	Task            *TaskRepository
	ECN             *ECNRepository
	Document        *DocumentRepository
	DocumentCategory *DocumentCategoryRepository
	TaskAttachment  *TaskAttachmentRepository
	OperationLog    *OperationLogRepository
	SystemConfig    *SystemConfigRepository
	Template        *TemplateRepository
	// V2 新增
	ProjectBOM      *ProjectBOMRepository
	Deliverable     *DeliverableRepository
	Codename        *CodenameRepository
	// V6 任务表单
	TaskForm        *TaskFormRepository
	// V13 CMF
	CMF             *CMFRepository
	// V14 SKU
	SKU             *SKURepository
	// V15 图纸版本管理
	PartDrawing     *PartDrawingRepository
}

// NewRepositories 创建仓库集合
func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		User:            NewUserRepository(db),
		Product:         NewProductRepository(db),
		ProductCategory: NewProductCategoryRepository(db),
		Material:        NewMaterialRepository(db),
		MaterialCategory: NewMaterialCategoryRepository(db),
		BOM:             NewBOMRepository(db),
		Project:         NewProjectRepository(db),
		Task:            NewTaskRepository(db),
		ECN:             NewECNRepository(db),
		Document:        NewDocumentRepository(db),
		DocumentCategory: NewDocumentCategoryRepository(db),
		TaskAttachment:  NewTaskAttachmentRepository(db),
		OperationLog:    NewOperationLogRepository(db),
		SystemConfig:    NewSystemConfigRepository(db),
		Template:        NewTemplateRepository(db),
		// V2 新增
		ProjectBOM:      NewProjectBOMRepository(db),
		Deliverable:     NewDeliverableRepository(db),
		Codename:        NewCodenameRepository(db),
		// V6 任务表单
		TaskForm:        NewTaskFormRepository(db),
		// V13 CMF
		CMF:             NewCMFRepository(db),
		// V14 SKU
		SKU:             NewSKURepository(db),
		// V15 图纸版本管理
		PartDrawing:     NewPartDrawingRepository(db),
	}
}
