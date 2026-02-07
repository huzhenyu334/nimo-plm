package service

import (
	"fmt"
	"time"

	"github.com/bitfantasy/nimo-plm/internal/erp/entity"
	"github.com/bitfantasy/nimo-plm/internal/erp/repository"
	"github.com/google/uuid"
)

type SupplierService struct {
	repo *repository.SupplierRepository
}

func NewSupplierService(repo *repository.SupplierRepository) *SupplierService {
	return &SupplierService{repo: repo}
}

type CreateSupplierRequest struct {
	Name         string `json:"name" binding:"required"`
	Type         string `json:"type" binding:"required"`
	ContactName  string `json:"contact_name" binding:"required"`
	Phone        string `json:"phone" binding:"required"`
	Email        string `json:"email"`
	Address      string `json:"address" binding:"required"`
	PaymentTerms string `json:"payment_terms"`
	Notes        string `json:"notes"`
}

type UpdateSupplierRequest struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	ContactName  string `json:"contact_name"`
	Phone        string `json:"phone"`
	Email        string `json:"email"`
	Address      string `json:"address"`
	PaymentTerms string `json:"payment_terms"`
	Status       string `json:"status"`
	Notes        string `json:"notes"`
}

type UpdateScoreRequest struct {
	QualityScore  float64 `json:"quality_score" binding:"min=0,max=100"`
	DeliveryScore float64 `json:"delivery_score" binding:"min=0,max=100"`
	PriceScore    float64 `json:"price_score" binding:"min=0,max=100"`
	ServiceScore  float64 `json:"service_score" binding:"min=0,max=100"`
}

func (s *SupplierService) Create(req CreateSupplierRequest, userID string) (*entity.Supplier, error) {
	// 生成供应商编码
	code := fmt.Sprintf("SUP-%s", time.Now().Format("20060102")+fmt.Sprintf("%04d", time.Now().UnixNano()%10000))

	supplier := &entity.Supplier{
		ID:           uuid.New().String(),
		SupplierCode: code,
		Name:         req.Name,
		Type:         req.Type,
		ContactName:  req.ContactName,
		Phone:        req.Phone,
		Email:        req.Email,
		Address:      req.Address,
		PaymentTerms: req.PaymentTerms,
		Status:       entity.SupplierStatusActive,
		Notes:        req.Notes,
		CreatedBy:    userID,
		UpdatedBy:    userID,
	}

	if err := s.repo.Create(supplier); err != nil {
		return nil, fmt.Errorf("failed to create supplier: %w", err)
	}
	return supplier, nil
}

func (s *SupplierService) GetByID(id string) (*entity.Supplier, error) {
	return s.repo.GetByID(id)
}

func (s *SupplierService) Update(id string, req UpdateSupplierRequest, userID string) (*entity.Supplier, error) {
	supplier, err := s.repo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("supplier not found: %w", err)
	}

	if req.Name != "" {
		supplier.Name = req.Name
	}
	if req.Type != "" {
		supplier.Type = req.Type
	}
	if req.ContactName != "" {
		supplier.ContactName = req.ContactName
	}
	if req.Phone != "" {
		supplier.Phone = req.Phone
	}
	if req.Email != "" {
		supplier.Email = req.Email
	}
	if req.Address != "" {
		supplier.Address = req.Address
	}
	if req.PaymentTerms != "" {
		supplier.PaymentTerms = req.PaymentTerms
	}
	if req.Status != "" {
		supplier.Status = req.Status
	}
	if req.Notes != "" {
		supplier.Notes = req.Notes
	}
	supplier.UpdatedBy = userID

	if err := s.repo.Update(supplier); err != nil {
		return nil, fmt.Errorf("failed to update supplier: %w", err)
	}
	return supplier, nil
}

func (s *SupplierService) Delete(id string) error {
	return s.repo.Delete(id)
}

func (s *SupplierService) List(params repository.SupplierListParams) ([]entity.Supplier, int64, error) {
	return s.repo.List(params)
}

func (s *SupplierService) UpdateScore(id string, req UpdateScoreRequest) error {
	// 验证供应商存在
	if _, err := s.repo.GetByID(id); err != nil {
		return fmt.Errorf("supplier not found: %w", err)
	}
	return s.repo.UpdateScore(id, req.QualityScore, req.DeliveryScore, req.PriceScore, req.ServiceScore)
}
