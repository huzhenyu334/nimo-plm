package service

import (
	"context"
	"fmt"

	"github.com/bitfantasy/nimo/internal/config"
	"github.com/bitfantasy/nimo/internal/plm/entity"
	"github.com/bitfantasy/nimo/internal/plm/repository"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
)

// Services 服务集合
type Services struct {
	Auth       *AuthService
	User       *UserService
	Product    *ProductService
	Material   *MaterialService
	BOM        *BOMService
	Project    *ProjectService
	ECN        *ECNService
	Document   *DocumentService
	Feishu     *FeishuIntegrationService
	Template   *TemplateService
	Automation *AutomationService
	// V2 新增
	ProjectBOM *ProjectBOMService
	// V14 SKU
	SKU *SKUService
}

// NewServices 创建服务集合
func NewServices(repos *repository.Repositories, rdb *redis.Client, cfg *config.Config) *Services {
	// 初始化飞书集成服务
	var feishuSvc *FeishuIntegrationService
	if cfg.Feishu.AppID != "" && cfg.Feishu.AppSecret != "" {
		feishuSvc = NewFeishuIntegrationService(cfg.Feishu.AppID, cfg.Feishu.AppSecret)
	}

	// 初始化MinIO客户端
	var minioClient *minio.Client
	if cfg.MinIO.Endpoint != "" {
		var err error
		minioClient, err = minio.New(cfg.MinIO.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.MinIO.AccessKey, cfg.MinIO.SecretKey, ""),
			Secure: cfg.MinIO.UseSSL,
		})
		if err != nil {
			// Log warning but continue without MinIO
			minioClient = nil
		}
	}

	templateSvc := NewTemplateService(repos.Template, repos.Project, repos.TaskForm)

	return &Services{
		Auth:       NewAuthService(repos.User, rdb, cfg),
		User:       NewUserService(repos.User, rdb),
		Product:    NewProductService(repos.Product, repos.ProductCategory, rdb),
		Material:   NewMaterialService(repos.Material, repos.MaterialCategory, rdb),
		BOM:        NewBOMService(repos.BOM, repos.Material, rdb),
		Project:    NewProjectService(repos.Project, repos.Task, repos.Product, feishuSvc, repos.TaskForm),
		ECN:        NewECNService(repos.ECN, repos.Product, feishuSvc),
		Document:   NewDocumentService(repos.Document, repos.DocumentCategory, minioClient, cfg.MinIO.Bucket),
		Feishu:     feishuSvc,
		Template:   templateSvc,
		Automation: nil, // Will be initialized with logger later if needed
		// V2 新增
		ProjectBOM: NewProjectBOMService(repos.ProjectBOM, repos.Project, repos.Deliverable, repos.Material),
		// V14 SKU
		SKU: NewSKUService(repos.SKU, repos.ProjectBOM),
	}
}

// UserService 用户服务
type UserService struct {
	repo *repository.UserRepository
	rdb  *redis.Client
}

// NewUserService 创建用户服务
func NewUserService(repo *repository.UserRepository, rdb *redis.Client) *UserService {
	return &UserService{repo: repo, rdb: rdb}
}

// ListAll 获取所有活跃用户
func (s *UserService) ListAll(ctx context.Context) ([]entity.User, error) {
	return s.repo.ListActive(ctx)
}

// Search 搜索用户（按名字/邮箱模糊匹配）
func (s *UserService) Search(ctx context.Context, query string) ([]entity.User, error) {
	return s.repo.Search(ctx, query)
}

// MaterialService 物料服务
type MaterialService struct {
	repo    *repository.MaterialRepository
	catRepo *repository.MaterialCategoryRepository
	rdb     *redis.Client
}

// NewMaterialService 创建物料服务
func NewMaterialService(repo *repository.MaterialRepository, catRepo *repository.MaterialCategoryRepository, rdb *redis.Client) *MaterialService {
	return &MaterialService{repo: repo, catRepo: catRepo, rdb: rdb}
}

// CreateMaterialRequest 创建物料请求
type CreateMaterialRequest struct {
	Name         string             `json:"name" binding:"required"`
	CategoryID   string             `json:"category_id"`
	Unit         string             `json:"unit"`
	Description  string             `json:"description"`
	Specs        entity.JSONB       `json:"specs"`
	LeadTimeDays int                `json:"lead_time_days"`
	MinOrderQty  float64            `json:"min_order_qty"`
	SafetyStock  float64            `json:"safety_stock"`
	StandardCost float64            `json:"standard_cost"`
	Currency     string             `json:"currency"`
}

// UpdateMaterialRequest 更新物料请求
type UpdateMaterialRequest struct {
	Name         *string            `json:"name"`
	CategoryID   *string            `json:"category_id"`
	Status       *string            `json:"status"`
	Unit         *string            `json:"unit"`
	Description  *string            `json:"description"`
	Specs        entity.JSONB       `json:"specs"`
	LeadTimeDays *int               `json:"lead_time_days"`
	MinOrderQty  *float64           `json:"min_order_qty"`
	SafetyStock  *float64           `json:"safety_stock"`
	StandardCost *float64           `json:"standard_cost"`
	LastCost     *float64           `json:"last_cost"`
	Currency     *string            `json:"currency"`
}


// List 获取物料列表
func (s *MaterialService) List(ctx context.Context, page, pageSize int, filters map[string]interface{}) (map[string]interface{}, error) {
	materials, total, err := s.repo.List(ctx, page, pageSize, filters)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"materials": materials,
		"total":     total,
	}, nil
}

// Get 获取物料详情
func (s *MaterialService) Get(ctx context.Context, id string) (*entity.Material, error) {
	return s.repo.FindByID(ctx, id)
}

// Create 创建物料
func (s *MaterialService) Create(ctx context.Context, userID string, req *CreateMaterialRequest) (*entity.Material, error) {
	// 根据 categoryID 查找分类，使用其 code 生成编码
	categoryCode := "OT-OTH"
	if req.CategoryID != "" {
		cat, err := s.repo.FindCategoryByID(ctx, req.CategoryID)
		if err != nil {
			return nil, fmt.Errorf("获取物料类别失败: %w", err)
		}
		categoryCode = cat.Code
		// 如果选的是一级分类，用 {code}-OTH 格式
		if cat.Level == 1 {
			categoryCode = cat.Code + "-OTH"
		}
	}

	code, err := s.repo.GenerateCode(ctx, categoryCode)
	if err != nil {
		return nil, fmt.Errorf("生成物料编码失败: %w", err)
	}

	unit := req.Unit
	if unit == "" {
		unit = "pcs"
	}
	currency := req.Currency
	if currency == "" {
		currency = "CNY"
	}

	material := &entity.Material{
		ID:           uuid.New().String()[:32],
		Code:         code,
		Name:         req.Name,
		CategoryID:   req.CategoryID,
		Status:       entity.MaterialStatusActive,
		Unit:         unit,
		Description:  req.Description,
		Specs:        req.Specs,
		LeadTimeDays: req.LeadTimeDays,
		MinOrderQty:  req.MinOrderQty,
		SafetyStock:  req.SafetyStock,
		StandardCost: req.StandardCost,
		Currency:     currency,
		CreatedBy:    userID,
	}

	if err := s.repo.Create(ctx, material); err != nil {
		return nil, fmt.Errorf("创建物料失败: %w", err)
	}

	return material, nil
}

// Update 更新物料
func (s *MaterialService) Update(ctx context.Context, id string, req *UpdateMaterialRequest) (*entity.Material, error) {
	material, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		material.Name = *req.Name
	}
	if req.CategoryID != nil {
		material.CategoryID = *req.CategoryID
	}
	if req.Status != nil {
		material.Status = *req.Status
	}
	if req.Unit != nil {
		material.Unit = *req.Unit
	}
	if req.Description != nil {
		material.Description = *req.Description
	}
	if req.Specs != nil {
		material.Specs = req.Specs
	}
	if req.LeadTimeDays != nil {
		material.LeadTimeDays = *req.LeadTimeDays
	}
	if req.MinOrderQty != nil {
		material.MinOrderQty = *req.MinOrderQty
	}
	if req.SafetyStock != nil {
		material.SafetyStock = *req.SafetyStock
	}
	if req.StandardCost != nil {
		material.StandardCost = *req.StandardCost
	}
	if req.LastCost != nil {
		material.LastCost = *req.LastCost
	}
	if req.Currency != nil {
		material.Currency = *req.Currency
	}

	if err := s.repo.Update(ctx, material); err != nil {
		return nil, fmt.Errorf("更新物料失败: %w", err)
	}

	return material, nil
}

// ListCategories 获取物料类别列表
func (s *MaterialService) ListCategories(ctx context.Context) ([]entity.MaterialCategory, error) {
	return s.repo.GetCategories(ctx)
}

// BOMService BOM服务
type BOMService struct {
	bomRepo      *repository.BOMRepository
	materialRepo *repository.MaterialRepository
	rdb          *redis.Client
}

// NewBOMService 创建BOM服务
func NewBOMService(bomRepo *repository.BOMRepository, materialRepo *repository.MaterialRepository, rdb *redis.Client) *BOMService {
	return &BOMService{
		bomRepo:      bomRepo,
		materialRepo: materialRepo,
		rdb:          rdb,
	}
}
