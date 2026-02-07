package repository

import (
	"github.com/bitfantasy/nimo-plm/internal/erp/entity"
	"gorm.io/gorm"
)

type SupplierRepository struct {
	db *gorm.DB
}

func NewSupplierRepository(db *gorm.DB) *SupplierRepository {
	return &SupplierRepository{db: db}
}

func (r *SupplierRepository) Create(supplier *entity.Supplier) error {
	return r.db.Create(supplier).Error
}

func (r *SupplierRepository) GetByID(id string) (*entity.Supplier, error) {
	var supplier entity.Supplier
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&supplier).Error
	if err != nil {
		return nil, err
	}
	return &supplier, nil
}

func (r *SupplierRepository) GetByCode(code string) (*entity.Supplier, error) {
	var supplier entity.Supplier
	err := r.db.Where("supplier_code = ? AND deleted_at IS NULL", code).First(&supplier).Error
	if err != nil {
		return nil, err
	}
	return &supplier, nil
}

func (r *SupplierRepository) Update(supplier *entity.Supplier) error {
	return r.db.Save(supplier).Error
}

func (r *SupplierRepository) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&entity.Supplier{}).Error
}

type SupplierListParams struct {
	Status  string
	Type    string
	Rating  string
	Keyword string
	Page    int
	Size    int
}

func (r *SupplierRepository) List(params SupplierListParams) ([]entity.Supplier, int64, error) {
	query := r.db.Model(&entity.Supplier{}).Where("deleted_at IS NULL")

	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.Type != "" {
		query = query.Where("type = ?", params.Type)
	}
	if params.Rating != "" {
		query = query.Where("rating = ?", params.Rating)
	}
	if params.Keyword != "" {
		kw := "%" + params.Keyword + "%"
		query = query.Where("name ILIKE ? OR supplier_code ILIKE ? OR contact_name ILIKE ?", kw, kw, kw)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if params.Page <= 0 {
		params.Page = 1
	}
	if params.Size <= 0 {
		params.Size = 20
	}

	var suppliers []entity.Supplier
	err := query.Order("created_at DESC").
		Offset((params.Page - 1) * params.Size).
		Limit(params.Size).
		Find(&suppliers).Error

	return suppliers, total, err
}

func (r *SupplierRepository) UpdateScore(id string, quality, delivery, price, service float64) error {
	overall := quality*0.4 + delivery*0.3 + price*0.2 + service*0.1
	var rating string
	switch {
	case overall >= 90:
		rating = entity.SupplierRatingA
	case overall >= 75:
		rating = entity.SupplierRatingB
	case overall >= 60:
		rating = entity.SupplierRatingC
	default:
		rating = entity.SupplierRatingD
	}

	return r.db.Model(&entity.Supplier{}).Where("id = ?", id).Updates(map[string]interface{}{
		"quality_score":  quality,
		"delivery_score": delivery,
		"price_score":    price,
		"service_score":  service,
		"overall_score":  overall,
		"rating":         rating,
	}).Error
}
