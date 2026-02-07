package service

import (
	"fmt"
	"time"

	"github.com/bitfantasy/nimo-plm/internal/erp/entity"
	"github.com/bitfantasy/nimo-plm/internal/erp/repository"
	"github.com/google/uuid"
)

type SalesService struct {
	repo          *repository.SalesRepository
	inventoryRepo *repository.InventoryRepository
}

func NewSalesService(repo *repository.SalesRepository, invRepo *repository.InventoryRepository) *SalesService {
	return &SalesService{repo: repo, inventoryRepo: invRepo}
}

// --- Customer ---

type CreateCustomerRequest struct {
	Name         string  `json:"name" binding:"required"`
	Type         string  `json:"type"`
	ContactName  string  `json:"contact_name"`
	Phone        string  `json:"phone"`
	Email        string  `json:"email"`
	Address      string  `json:"address"`
	Channel      string  `json:"channel"`
	CreditLimit  float64 `json:"credit_limit"`
	PaymentTerms string  `json:"payment_terms"`
	Notes        string  `json:"notes"`
}

func (s *SalesService) CreateCustomer(req CreateCustomerRequest, userID string) (*entity.Customer, error) {
	code := fmt.Sprintf("CUS-%s%04d", time.Now().Format("20060102"), time.Now().UnixNano()%10000)
	custType := req.Type
	if custType == "" {
		custType = entity.CustomerTypeRetail
	}

	customer := &entity.Customer{
		ID:           uuid.New().String(),
		CustomerCode: code,
		Name:         req.Name,
		Type:         custType,
		ContactName:  req.ContactName,
		Phone:        req.Phone,
		Email:        req.Email,
		Address:      req.Address,
		Channel:      req.Channel,
		CreditLimit:  req.CreditLimit,
		PaymentTerms: req.PaymentTerms,
		Status:       entity.CustomerStatusActive,
		Notes:        req.Notes,
		CreatedBy:    userID,
	}

	if err := s.repo.CreateCustomer(customer); err != nil {
		return nil, fmt.Errorf("创建客户失败: %w", err)
	}
	return customer, nil
}

func (s *SalesService) GetCustomerByID(id string) (*entity.Customer, error) {
	return s.repo.GetCustomerByID(id)
}

func (s *SalesService) ListCustomers(params repository.CustomerListParams) ([]entity.Customer, int64, error) {
	return s.repo.ListCustomers(params)
}

func (s *SalesService) DeleteCustomer(id string) error {
	return s.repo.DeleteCustomer(id)
}

// --- Sales Order ---

type CreateSORequest struct {
	CustomerID      string         `json:"customer_id" binding:"required"`
	Channel         string         `json:"channel"`
	ShippingAddress string         `json:"shipping_address"`
	Currency        string         `json:"currency"`
	Notes           string         `json:"notes"`
	Items           []CreateSOItem `json:"items" binding:"required,min=1"`
}

type CreateSOItem struct {
	ProductID   string  `json:"product_id" binding:"required"`
	ProductCode string  `json:"product_code"`
	ProductName string  `json:"product_name"`
	Quantity    float64 `json:"quantity" binding:"required,gt=0"`
	UnitPrice   float64 `json:"unit_price" binding:"required,gt=0"`
}

func (s *SalesService) CreateSO(req CreateSORequest, userID string) (*entity.SalesOrder, error) {
	// 验证客户
	if _, err := s.repo.GetCustomerByID(req.CustomerID); err != nil {
		return nil, fmt.Errorf("客户不存在: %w", err)
	}

	code := fmt.Sprintf("SO-%s%04d", time.Now().Format("20060102"), time.Now().UnixNano()%10000)
	now := time.Now()
	channel := req.Channel
	if channel == "" {
		channel = entity.ChannelDirect
	}
	currency := req.Currency
	if currency == "" {
		currency = "CNY"
	}

	so := &entity.SalesOrder{
		ID:              uuid.New().String(),
		SOCode:          code,
		CustomerID:      req.CustomerID,
		Channel:         channel,
		Status:          entity.SOStatusPending,
		Currency:        currency,
		OrderDate:       &now,
		ShippingAddress: req.ShippingAddress,
		Notes:           req.Notes,
		CreatedBy:       userID,
	}

	var totalAmount float64
	var items []entity.SOItem
	for _, item := range req.Items {
		amount := item.Quantity * item.UnitPrice
		totalAmount += amount
		items = append(items, entity.SOItem{
			ID:          uuid.New().String(),
			SOID:        so.ID,
			ProductID:   item.ProductID,
			ProductCode: item.ProductCode,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			Amount:      amount,
			Status:      entity.SOItemStatusOpen,
		})
	}
	so.TotalAmount = totalAmount
	so.Items = items

	if err := s.repo.CreateSO(so); err != nil {
		return nil, fmt.Errorf("创建销售订单失败: %w", err)
	}
	return so, nil
}

func (s *SalesService) GetSOByID(id string) (*entity.SalesOrder, error) {
	return s.repo.GetSOByID(id)
}

func (s *SalesService) ListSOs(params repository.SOListParams) ([]entity.SalesOrder, int64, error) {
	return s.repo.ListSOs(params)
}

func (s *SalesService) ConfirmSO(id string) error {
	so, err := s.repo.GetSOByID(id)
	if err != nil {
		return fmt.Errorf("销售订单不存在: %w", err)
	}
	if so.Status != entity.SOStatusPending {
		return fmt.Errorf("订单状态不允许确认: %s", so.Status)
	}
	so.Status = entity.SOStatusConfirmed
	return s.repo.UpdateSO(so)
}

func (s *SalesService) ShipSO(id, trackingNo, userID string) error {
	so, err := s.repo.GetSOByID(id)
	if err != nil {
		return fmt.Errorf("销售订单不存在: %w", err)
	}
	if so.Status != entity.SOStatusConfirmed && so.Status != entity.SOStatusPicking {
		return fmt.Errorf("订单状态不允许发货: %s", so.Status)
	}
	now := time.Now()
	so.Status = entity.SOStatusShipped
	so.ShippingDate = &now
	so.TrackingNo = trackingNo
	return s.repo.UpdateSO(so)
}

func (s *SalesService) CancelSO(id string) error {
	so, err := s.repo.GetSOByID(id)
	if err != nil {
		return fmt.Errorf("销售订单不存在: %w", err)
	}
	if so.Status == entity.SOStatusShipped || so.Status == entity.SOStatusDelivered || so.Status == entity.SOStatusCompleted {
		return fmt.Errorf("已发货/签收/完成的订单不能取消")
	}
	so.Status = entity.SOStatusCancelled
	return s.repo.UpdateSO(so)
}

// --- Service Order ---

type CreateServiceOrderRequest struct {
	CustomerID  string `json:"customer_id" binding:"required"`
	ProductSN   string `json:"product_sn" binding:"required"`
	ServiceType string `json:"service_type" binding:"required"`
	Priority    int    `json:"priority"`
	Description string `json:"description" binding:"required"`
	Notes       string `json:"notes"`
}

func (s *SalesService) CreateServiceOrder(req CreateServiceOrderRequest, userID string) (*entity.ServiceOrder, error) {
	code := fmt.Sprintf("SVC-%s%04d", time.Now().Format("20060102"), time.Now().UnixNano()%10000)

	svcOrder := &entity.ServiceOrder{
		ID:          uuid.New().String(),
		ServiceCode: code,
		CustomerID:  req.CustomerID,
		ProductSN:   req.ProductSN,
		ServiceType: req.ServiceType,
		Status:      entity.SvcStatusCreated,
		Priority:    req.Priority,
		Description: req.Description,
		Notes:       req.Notes,
		CreatedBy:   userID,
	}

	if err := s.repo.CreateServiceOrder(svcOrder); err != nil {
		return nil, fmt.Errorf("创建服务工单失败: %w", err)
	}
	return svcOrder, nil
}

func (s *SalesService) GetServiceOrderByID(id string) (*entity.ServiceOrder, error) {
	return s.repo.GetServiceOrderByID(id)
}

func (s *SalesService) ListServiceOrders(params repository.ServiceOrderListParams) ([]entity.ServiceOrder, int64, error) {
	return s.repo.ListServiceOrders(params)
}

func (s *SalesService) AssignServiceOrder(id, assigneeID, assigneeName string) error {
	so, err := s.repo.GetServiceOrderByID(id)
	if err != nil {
		return fmt.Errorf("服务工单不存在: %w", err)
	}
	so.AssigneeID = assigneeID
	so.AssigneeName = assigneeName
	so.Status = entity.SvcStatusAssigned
	return s.repo.UpdateServiceOrder(so)
}

func (s *SalesService) CompleteServiceOrder(id, solution string) error {
	so, err := s.repo.GetServiceOrderByID(id)
	if err != nil {
		return fmt.Errorf("服务工单不存在: %w", err)
	}
	now := time.Now()
	so.Status = entity.SvcStatusCompleted
	so.Solution = solution
	so.CompletedAt = &now
	return s.repo.UpdateServiceOrder(so)
}
