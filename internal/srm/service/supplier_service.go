package service

import (
	"context"

	"github.com/bitfantasy/nimo/internal/srm/entity"
	"github.com/bitfantasy/nimo/internal/srm/repository"
	"github.com/google/uuid"
)

// SupplierService 供应商服务
type SupplierService struct {
	repo *repository.SupplierRepository
}

func NewSupplierService(repo *repository.SupplierRepository) *SupplierService {
	return &SupplierService{repo: repo}
}

// CreateSupplierRequest 创建供应商请求
type CreateSupplierRequest struct {
	Name           string              `json:"name" binding:"required"`
	ShortName      string              `json:"short_name"`
	Category       string              `json:"category" binding:"required"`
	Level          string              `json:"level"`
	Country        string              `json:"country"`
	Province       string              `json:"province"`
	City           string              `json:"city"`
	Address        string              `json:"address"`
	Website        string              `json:"website"`
	BusinessScope  string              `json:"business_scope"`
	AnnualRevenue  *float64            `json:"annual_revenue"`
	EmployeeCount  *int                `json:"employee_count"`
	FactoryArea    *float64            `json:"factory_area"`
	Certifications *entity.JSONBArray  `json:"certifications"`
	BankName       string              `json:"bank_name"`
	BankAccount    string              `json:"bank_account"`
	TaxID          string              `json:"tax_id"`
	PaymentTerms   string              `json:"payment_terms"`
	Tags           *entity.JSONBArray  `json:"tags"`
	TechCapability string              `json:"tech_capability"`
	Cooperation    string              `json:"cooperation"`
	CapacityLimit  string              `json:"capacity_limit"`
	Notes          string              `json:"notes"`
}

// UpdateSupplierRequest 更新供应商请求
type UpdateSupplierRequest struct {
	Name           *string             `json:"name"`
	ShortName      *string             `json:"short_name"`
	Category       *string             `json:"category"`
	Level          *string             `json:"level"`
	Status         *string             `json:"status"`
	Country        *string             `json:"country"`
	Province       *string             `json:"province"`
	City           *string             `json:"city"`
	Address        *string             `json:"address"`
	Website        *string             `json:"website"`
	BusinessScope  *string             `json:"business_scope"`
	AnnualRevenue  *float64            `json:"annual_revenue"`
	EmployeeCount  *int                `json:"employee_count"`
	FactoryArea    *float64            `json:"factory_area"`
	Certifications *entity.JSONBArray  `json:"certifications"`
	BankName       *string             `json:"bank_name"`
	BankAccount    *string             `json:"bank_account"`
	TaxID          *string             `json:"tax_id"`
	PaymentTerms   *string             `json:"payment_terms"`
	Tags           *entity.JSONBArray  `json:"tags"`
	TechCapability *string             `json:"tech_capability"`
	Cooperation    *string             `json:"cooperation"`
	CapacityLimit  *string             `json:"capacity_limit"`
	Notes          *string             `json:"notes"`
}

// List 获取供应商列表
func (s *SupplierService) List(ctx context.Context, page, pageSize int, filters map[string]string) ([]entity.Supplier, int64, error) {
	return s.repo.FindAll(ctx, page, pageSize, filters)
}

// Get 获取供应商详情
func (s *SupplierService) Get(ctx context.Context, id string) (*entity.Supplier, error) {
	return s.repo.FindByID(ctx, id)
}

// Create 创建供应商
func (s *SupplierService) Create(ctx context.Context, userID string, req *CreateSupplierRequest) (*entity.Supplier, error) {
	code, err := s.repo.GenerateCode(ctx)
	if err != nil {
		return nil, err
	}

	supplier := &entity.Supplier{
		ID:             uuid.New().String()[:32],
		Code:           code,
		Name:           req.Name,
		ShortName:      req.ShortName,
		Category:       req.Category,
		Level:          req.Level,
		Status:         entity.SupplierStatusPending,
		Country:        req.Country,
		Province:       req.Province,
		City:           req.City,
		Address:        req.Address,
		Website:        req.Website,
		BusinessScope:  req.BusinessScope,
		AnnualRevenue:  req.AnnualRevenue,
		EmployeeCount:  req.EmployeeCount,
		FactoryArea:    req.FactoryArea,
		Certifications: req.Certifications,
		BankName:       req.BankName,
		BankAccount:    req.BankAccount,
		TaxID:          req.TaxID,
		PaymentTerms:   req.PaymentTerms,
		Tags:           req.Tags,
		TechCapability: req.TechCapability,
		Cooperation:    req.Cooperation,
		CapacityLimit:  req.CapacityLimit,
		Notes:          req.Notes,
		CreatedBy:      userID,
	}

	if supplier.Level == "" {
		supplier.Level = entity.SupplierLevelPotential
	}

	if err := s.repo.Create(ctx, supplier); err != nil {
		return nil, err
	}
	return supplier, nil
}

// Update 更新供应商
func (s *SupplierService) Update(ctx context.Context, id string, req *UpdateSupplierRequest) (*entity.Supplier, error) {
	supplier, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		supplier.Name = *req.Name
	}
	if req.ShortName != nil {
		supplier.ShortName = *req.ShortName
	}
	if req.Category != nil {
		supplier.Category = *req.Category
	}
	if req.Level != nil {
		supplier.Level = *req.Level
	}
	if req.Status != nil {
		supplier.Status = *req.Status
	}
	if req.Country != nil {
		supplier.Country = *req.Country
	}
	if req.Province != nil {
		supplier.Province = *req.Province
	}
	if req.City != nil {
		supplier.City = *req.City
	}
	if req.Address != nil {
		supplier.Address = *req.Address
	}
	if req.Website != nil {
		supplier.Website = *req.Website
	}
	if req.BusinessScope != nil {
		supplier.BusinessScope = *req.BusinessScope
	}
	if req.AnnualRevenue != nil {
		supplier.AnnualRevenue = req.AnnualRevenue
	}
	if req.EmployeeCount != nil {
		supplier.EmployeeCount = req.EmployeeCount
	}
	if req.FactoryArea != nil {
		supplier.FactoryArea = req.FactoryArea
	}
	if req.Certifications != nil {
		supplier.Certifications = req.Certifications
	}
	if req.BankName != nil {
		supplier.BankName = *req.BankName
	}
	if req.BankAccount != nil {
		supplier.BankAccount = *req.BankAccount
	}
	if req.TaxID != nil {
		supplier.TaxID = *req.TaxID
	}
	if req.PaymentTerms != nil {
		supplier.PaymentTerms = *req.PaymentTerms
	}
	if req.Tags != nil {
		supplier.Tags = req.Tags
	}
	if req.TechCapability != nil {
		supplier.TechCapability = *req.TechCapability
	}
	if req.Cooperation != nil {
		supplier.Cooperation = *req.Cooperation
	}
	if req.CapacityLimit != nil {
		supplier.CapacityLimit = *req.CapacityLimit
	}
	if req.Notes != nil {
		supplier.Notes = *req.Notes
	}

	if err := s.repo.Update(ctx, supplier); err != nil {
		return nil, err
	}
	return supplier, nil
}

// Delete 删除供应商
func (s *SupplierService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// ListContacts 获取联系人列表
func (s *SupplierService) ListContacts(ctx context.Context, supplierID string) ([]entity.SupplierContact, error) {
	return s.repo.FindContacts(ctx, supplierID)
}

// CreateContactRequest 创建联系人请求
type CreateContactRequest struct {
	Name      string `json:"name" binding:"required"`
	Title     string `json:"title"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	Wechat    string `json:"wechat"`
	IsPrimary bool   `json:"is_primary"`
}

// CreateContact 创建联系人
func (s *SupplierService) CreateContact(ctx context.Context, supplierID string, req *CreateContactRequest) (*entity.SupplierContact, error) {
	contact := &entity.SupplierContact{
		ID:         uuid.New().String()[:32],
		SupplierID: supplierID,
		Name:       req.Name,
		Title:      req.Title,
		Phone:      req.Phone,
		Email:      req.Email,
		Wechat:     req.Wechat,
		IsPrimary:  req.IsPrimary,
	}
	if err := s.repo.CreateContact(ctx, contact); err != nil {
		return nil, err
	}
	return contact, nil
}

// DeleteContact 删除联系人
func (s *SupplierService) DeleteContact(ctx context.Context, contactID string) error {
	return s.repo.DeleteContact(ctx, contactID)
}

// UpdateScores 更新供应商评分
func (s *SupplierService) UpdateScores(ctx context.Context, id string, quality, delivery, price float64) error {
	// 综合评分: 质量40% + 交期30% + 价格30%
	overall := quality*0.4 + delivery*0.3 + price*0.3
	return s.repo.UpdateScores(ctx, id, quality, delivery, price, overall)
}
