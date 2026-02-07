package repository

import (
	"github.com/bitfantasy/nimo-plm/internal/erp/entity"
	"gorm.io/gorm"
)

type WorkOrderRepository struct {
	db *gorm.DB
}

func NewWorkOrderRepository(db *gorm.DB) *WorkOrderRepository {
	return &WorkOrderRepository{db: db}
}

func (r *WorkOrderRepository) Create(wo *entity.WorkOrder) error {
	return r.db.Create(wo).Error
}

func (r *WorkOrderRepository) GetByID(id string) (*entity.WorkOrder, error) {
	var wo entity.WorkOrder
	err := r.db.Preload("Materials").Preload("Reports").
		Where("id = ? AND deleted_at IS NULL", id).First(&wo).Error
	return &wo, err
}

func (r *WorkOrderRepository) Update(wo *entity.WorkOrder) error {
	return r.db.Save(wo).Error
}

type WOListParams struct {
	Status    string
	ProductID string
	Keyword   string
	Page      int
	Size      int
}

func (r *WorkOrderRepository) List(params WOListParams) ([]entity.WorkOrder, int64, error) {
	query := r.db.Model(&entity.WorkOrder{}).Where("deleted_at IS NULL")
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.ProductID != "" {
		query = query.Where("product_id = ?", params.ProductID)
	}
	if params.Keyword != "" {
		kw := "%" + params.Keyword + "%"
		query = query.Where("wo_code ILIKE ? OR product_name ILIKE ?", kw, kw)
	}
	var total int64
	query.Count(&total)
	if params.Page <= 0 { params.Page = 1 }
	if params.Size <= 0 { params.Size = 20 }
	var wos []entity.WorkOrder
	err := query.Order("created_at DESC").Offset((params.Page-1)*params.Size).Limit(params.Size).Find(&wos).Error
	return wos, total, err
}

// CreateMaterial 创建工单物料需求
func (r *WorkOrderRepository) CreateMaterial(m *entity.WorkOrderMaterial) error {
	return r.db.Create(m).Error
}

func (r *WorkOrderRepository) BatchCreateMaterials(materials []entity.WorkOrderMaterial) error {
	return r.db.Create(&materials).Error
}

func (r *WorkOrderRepository) UpdateMaterial(m *entity.WorkOrderMaterial) error {
	return r.db.Save(m).Error
}

func (r *WorkOrderRepository) GetMaterialsByWOID(woID string) ([]entity.WorkOrderMaterial, error) {
	var materials []entity.WorkOrderMaterial
	err := r.db.Where("work_order_id = ?", woID).Find(&materials).Error
	return materials, err
}

// CreateReport 创建报工记录
func (r *WorkOrderRepository) CreateReport(report *entity.WorkOrderReport) error {
	return r.db.Create(report).Error
}

// GetInProductionQty 获取物料在制数量（活跃工单中的计划数量 - 已完成数量）
func (r *WorkOrderRepository) GetInProductionQty(productID string) (float64, error) {
	var result struct{ Total float64 }
	err := r.db.Raw(`
		SELECT COALESCE(SUM(planned_qty - completed_qty), 0) as total 
		FROM erp_work_orders 
		WHERE product_id = ? 
		AND status IN ('CREATED', 'PLANNED', 'RELEASED', 'IN_PROGRESS')
		AND deleted_at IS NULL
	`, productID).Scan(&result).Error
	return result.Total, err
}

// DB 返回底层db用于事务
func (r *WorkOrderRepository) DB() *gorm.DB {
	return r.db
}
