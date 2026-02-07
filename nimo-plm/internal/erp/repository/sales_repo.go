package repository

import (
	"github.com/bitfantasy/nimo-plm/internal/erp/entity"
	"gorm.io/gorm"
)

type SalesRepository struct {
	db *gorm.DB
}

func NewSalesRepository(db *gorm.DB) *SalesRepository {
	return &SalesRepository{db: db}
}

// --- Customer ---

func (r *SalesRepository) CreateCustomer(c *entity.Customer) error {
	return r.db.Create(c).Error
}

func (r *SalesRepository) GetCustomerByID(id string) (*entity.Customer, error) {
	var c entity.Customer
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&c).Error
	return &c, err
}

func (r *SalesRepository) UpdateCustomer(c *entity.Customer) error {
	return r.db.Save(c).Error
}

func (r *SalesRepository) DeleteCustomer(id string) error {
	return r.db.Where("id = ?", id).Delete(&entity.Customer{}).Error
}

type CustomerListParams struct {
	Status  string
	Type    string
	Keyword string
	Page    int
	Size    int
}

func (r *SalesRepository) ListCustomers(params CustomerListParams) ([]entity.Customer, int64, error) {
	query := r.db.Model(&entity.Customer{}).Where("deleted_at IS NULL")
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.Type != "" {
		query = query.Where("type = ?", params.Type)
	}
	if params.Keyword != "" {
		kw := "%" + params.Keyword + "%"
		query = query.Where("name ILIKE ? OR customer_code ILIKE ?", kw, kw)
	}
	var total int64
	query.Count(&total)
	if params.Page <= 0 { params.Page = 1 }
	if params.Size <= 0 { params.Size = 20 }
	var customers []entity.Customer
	err := query.Order("created_at DESC").Offset((params.Page-1)*params.Size).Limit(params.Size).Find(&customers).Error
	return customers, total, err
}

// --- Sales Order ---

func (r *SalesRepository) CreateSO(so *entity.SalesOrder) error {
	return r.db.Create(so).Error
}

func (r *SalesRepository) GetSOByID(id string) (*entity.SalesOrder, error) {
	var so entity.SalesOrder
	err := r.db.Preload("Customer").Preload("Items").
		Where("id = ? AND deleted_at IS NULL", id).First(&so).Error
	return &so, err
}

func (r *SalesRepository) UpdateSO(so *entity.SalesOrder) error {
	return r.db.Save(so).Error
}

type SOListParams struct {
	Status     string
	CustomerID string
	Channel    string
	Keyword    string
	Page       int
	Size       int
}

func (r *SalesRepository) ListSOs(params SOListParams) ([]entity.SalesOrder, int64, error) {
	query := r.db.Model(&entity.SalesOrder{}).Where("deleted_at IS NULL")
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.CustomerID != "" {
		query = query.Where("customer_id = ?", params.CustomerID)
	}
	if params.Channel != "" {
		query = query.Where("channel = ?", params.Channel)
	}
	if params.Keyword != "" {
		kw := "%" + params.Keyword + "%"
		query = query.Where("so_code ILIKE ?", kw)
	}
	var total int64
	query.Count(&total)
	if params.Page <= 0 { params.Page = 1 }
	if params.Size <= 0 { params.Size = 20 }
	var sos []entity.SalesOrder
	err := query.Preload("Customer").Order("created_at DESC").
		Offset((params.Page-1)*params.Size).Limit(params.Size).Find(&sos).Error
	return sos, total, err
}

func (r *SalesRepository) CreateSOItem(item *entity.SOItem) error {
	return r.db.Create(item).Error
}

func (r *SalesRepository) UpdateSOItem(item *entity.SOItem) error {
	return r.db.Save(item).Error
}

// GetPendingDemand 获取待满足的销售订单需求（按产品汇总）
func (r *SalesRepository) GetPendingDemand() (map[string]float64, error) {
	type DemandRow struct {
		ProductID string
		Total     float64
	}
	var rows []DemandRow
	err := r.db.Raw(`
		SELECT i.product_id, COALESCE(SUM(i.quantity - i.shipped_qty), 0) as total
		FROM erp_so_items i
		JOIN erp_sales_orders so ON so.id = i.so_id
		WHERE so.status IN ('PENDING', 'CONFIRMED', 'PICKING')
		AND so.deleted_at IS NULL
		AND i.status != 'CLOSED'
		GROUP BY i.product_id
	`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	demand := make(map[string]float64)
	for _, row := range rows {
		demand[row.ProductID] = row.Total
	}
	return demand, nil
}

// --- Service Order ---

func (r *SalesRepository) CreateServiceOrder(so *entity.ServiceOrder) error {
	return r.db.Create(so).Error
}

func (r *SalesRepository) GetServiceOrderByID(id string) (*entity.ServiceOrder, error) {
	var so entity.ServiceOrder
	err := r.db.Preload("Customer").
		Where("id = ? AND deleted_at IS NULL", id).First(&so).Error
	return &so, err
}

func (r *SalesRepository) UpdateServiceOrder(so *entity.ServiceOrder) error {
	return r.db.Save(so).Error
}

type ServiceOrderListParams struct {
	Status      string
	ServiceType string
	CustomerID  string
	AssigneeID  string
	Keyword     string
	Page        int
	Size        int
}

func (r *SalesRepository) ListServiceOrders(params ServiceOrderListParams) ([]entity.ServiceOrder, int64, error) {
	query := r.db.Model(&entity.ServiceOrder{}).Where("deleted_at IS NULL")
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.ServiceType != "" {
		query = query.Where("service_type = ?", params.ServiceType)
	}
	if params.CustomerID != "" {
		query = query.Where("customer_id = ?", params.CustomerID)
	}
	if params.AssigneeID != "" {
		query = query.Where("assignee_id = ?", params.AssigneeID)
	}
	if params.Keyword != "" {
		kw := "%" + params.Keyword + "%"
		query = query.Where("service_code ILIKE ? OR product_sn ILIKE ?", kw, kw)
	}
	var total int64
	query.Count(&total)
	if params.Page <= 0 { params.Page = 1 }
	if params.Size <= 0 { params.Size = 20 }
	var orders []entity.ServiceOrder
	err := query.Preload("Customer").Order("created_at DESC").
		Offset((params.Page-1)*params.Size).Limit(params.Size).Find(&orders).Error
	return orders, total, err
}
