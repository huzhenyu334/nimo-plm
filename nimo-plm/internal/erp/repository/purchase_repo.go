package repository

import (
	"github.com/bitfantasy/nimo-plm/internal/erp/entity"
	"gorm.io/gorm"
)

type PurchaseRepository struct {
	db *gorm.DB
}

func NewPurchaseRepository(db *gorm.DB) *PurchaseRepository {
	return &PurchaseRepository{db: db}
}

// --- Purchase Requisition ---

func (r *PurchaseRepository) CreatePR(pr *entity.PurchaseRequisition) error {
	return r.db.Create(pr).Error
}

func (r *PurchaseRepository) GetPRByID(id string) (*entity.PurchaseRequisition, error) {
	var pr entity.PurchaseRequisition
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&pr).Error
	return &pr, err
}

func (r *PurchaseRepository) UpdatePR(pr *entity.PurchaseRequisition) error {
	return r.db.Save(pr).Error
}

func (r *PurchaseRepository) ListPRs(status string, page, size int) ([]entity.PurchaseRequisition, int64, error) {
	query := r.db.Model(&entity.PurchaseRequisition{}).Where("deleted_at IS NULL")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	query.Count(&total)
	if page <= 0 { page = 1 }
	if size <= 0 { size = 20 }
	var prs []entity.PurchaseRequisition
	err := query.Order("created_at DESC").Offset((page-1)*size).Limit(size).Find(&prs).Error
	return prs, total, err
}

func (r *PurchaseRepository) BatchCreatePRs(prs []entity.PurchaseRequisition) error {
	return r.db.Create(&prs).Error
}

// --- Purchase Order ---

func (r *PurchaseRepository) CreatePO(po *entity.PurchaseOrder) error {
	return r.db.Create(po).Error
}

func (r *PurchaseRepository) GetPOByID(id string) (*entity.PurchaseOrder, error) {
	var po entity.PurchaseOrder
	err := r.db.Preload("Supplier").Preload("Items").
		Where("id = ? AND deleted_at IS NULL", id).First(&po).Error
	return &po, err
}

func (r *PurchaseRepository) UpdatePO(po *entity.PurchaseOrder) error {
	return r.db.Save(po).Error
}

type POListParams struct {
	Status     string
	SupplierID string
	Keyword    string
	Page       int
	Size       int
}

func (r *PurchaseRepository) ListPOs(params POListParams) ([]entity.PurchaseOrder, int64, error) {
	query := r.db.Model(&entity.PurchaseOrder{}).Where("deleted_at IS NULL")
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.SupplierID != "" {
		query = query.Where("supplier_id = ?", params.SupplierID)
	}
	if params.Keyword != "" {
		kw := "%" + params.Keyword + "%"
		query = query.Where("po_code ILIKE ?", kw)
	}
	var total int64
	query.Count(&total)
	if params.Page <= 0 { params.Page = 1 }
	if params.Size <= 0 { params.Size = 20 }
	var pos []entity.PurchaseOrder
	err := query.Preload("Supplier").Order("created_at DESC").
		Offset((params.Page-1)*params.Size).Limit(params.Size).Find(&pos).Error
	return pos, total, err
}

// --- PO Items ---

func (r *PurchaseRepository) CreatePOItem(item *entity.POItem) error {
	return r.db.Create(item).Error
}

func (r *PurchaseRepository) UpdatePOItem(item *entity.POItem) error {
	return r.db.Save(item).Error
}

func (r *PurchaseRepository) GetPOItemsByPOID(poID string) ([]entity.POItem, error) {
	var items []entity.POItem
	err := r.db.Where("po_id = ?", poID).Find(&items).Error
	return items, err
}

// GetInTransitQty 获取物料在途数量（已批准的PO中未完成收货的数量）
func (r *PurchaseRepository) GetInTransitQty(materialID string) (float64, error) {
	var result struct{ Total float64 }
	err := r.db.Raw(`
		SELECT COALESCE(SUM(i.quantity - i.received_qty), 0) as total 
		FROM erp_po_items i 
		JOIN erp_purchase_orders po ON po.id = i.po_id 
		WHERE i.material_id = ? 
		AND po.status IN ('APPROVED', 'SENT', 'PARTIAL')
		AND po.deleted_at IS NULL
		AND i.status != 'CLOSED'
	`, materialID).Scan(&result).Error
	return result.Total, err
}
