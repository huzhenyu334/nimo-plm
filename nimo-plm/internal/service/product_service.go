package service

import (
	"context"
	"fmt"
	"time"

	"github.com/bitfantasy/nimo-plm/internal/model/entity"
	"github.com/bitfantasy/nimo-plm/internal/repository"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ProductService 产品服务
type ProductService struct {
	repo    *repository.ProductRepository
	catRepo *repository.ProductCategoryRepository
	rdb     *redis.Client
}

// NewProductService 创建产品服务
func NewProductService(repo *repository.ProductRepository, catRepo *repository.ProductCategoryRepository, rdb *redis.Client) *ProductService {
	return &ProductService{repo: repo, catRepo: catRepo, rdb: rdb}
}

// CreateProductRequest 创建产品请求
type CreateProductRequest struct {
	Name          string                 `json:"name" binding:"required"`
	CategoryID    string                 `json:"category_id" binding:"required"`
	Description   string                 `json:"description"`
	Specs         map[string]interface{} `json:"specs"`
	BaseProductID string                 `json:"base_product_id"`
}

// UpdateProductRequest 更新产品请求
type UpdateProductRequest struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Specs        map[string]interface{} `json:"specs"`
	ThumbnailURL string                 `json:"thumbnail_url"`
}

// ProductListResult 产品列表结果
type ProductListResult struct {
	Items      []entity.Product `json:"items"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

// List 获取产品列表
func (s *ProductService) List(ctx context.Context, page, pageSize int, filters map[string]interface{}) (*ProductListResult, error) {
	products, total, err := s.repo.List(ctx, page, pageSize, filters)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &ProductListResult{
		Items:      products,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// Get 获取产品详情
func (s *ProductService) Get(ctx context.Context, id string) (*entity.Product, error) {
	product, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find product: %w", err)
	}
	return product, nil
}

// Create 创建产品
func (s *ProductService) Create(ctx context.Context, userID string, req *CreateProductRequest) (*entity.Product, error) {
	// 生成产品编码
	code, err := s.repo.GenerateCode(ctx, "PRD-NIMO")
	if err != nil {
		return nil, fmt.Errorf("generate code: %w", err)
	}

	now := time.Now()
	product := &entity.Product{
		ID:          uuid.New().String()[:32],
		Code:        code,
		Name:        req.Name,
		CategoryID:  req.CategoryID,
		Status:      entity.ProductStatusDraft,
		Description: req.Description,
		Specs:       entity.JSONB(req.Specs),
		CreatedBy:   userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, product); err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}

	// 如果基于已有产品创建，复制BOM
	if req.BaseProductID != "" {
		// TODO: 复制BOM逻辑
	}

	// 清除缓存
	s.clearCache(ctx)

	return product, nil
}

// Update 更新产品
func (s *ProductService) Update(ctx context.Context, id string, userID string, req *UpdateProductRequest) (*entity.Product, error) {
	product, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find product: %w", err)
	}

	// 检查状态，已发布的产品不能直接修改
	if product.Status == entity.ProductStatusActive {
		return nil, fmt.Errorf("cannot modify active product, please create ECN")
	}

	// 更新字段
	if req.Name != "" {
		product.Name = req.Name
	}
	if req.Description != "" {
		product.Description = req.Description
	}
	if req.Specs != nil {
		product.Specs = entity.JSONB(req.Specs)
	}
	if req.ThumbnailURL != "" {
		product.ThumbnailURL = req.ThumbnailURL
	}
	product.UpdatedBy = userID
	product.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, product); err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}

	// 清除缓存
	s.clearCache(ctx)

	return product, nil
}

// Delete 删除产品
func (s *ProductService) Delete(ctx context.Context, id string) error {
	product, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("find product: %w", err)
	}

	// 检查状态，已发布的产品不能删除
	if product.Status == entity.ProductStatusActive {
		return fmt.Errorf("cannot delete active product")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete product: %w", err)
	}

	// 清除缓存
	s.clearCache(ctx)

	return nil
}

// Release 发布产品
func (s *ProductService) Release(ctx context.Context, id string, userID string) (*entity.Product, error) {
	product, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find product: %w", err)
	}

	// 检查是否有已发布的BOM
	if product.CurrentBOMVersion == "" {
		return nil, fmt.Errorf("product must have a released BOM before release")
	}

	// 更新状态
	now := time.Now()
	product.Status = entity.ProductStatusActive
	product.ReleasedAt = &now
	product.UpdatedBy = userID
	product.UpdatedAt = now

	if err := s.repo.Update(ctx, product); err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}

	// 清除缓存
	s.clearCache(ctx)

	return product, nil
}

// GetCategories 获取产品类别列表
func (s *ProductService) GetCategories(ctx context.Context) ([]entity.ProductCategory, error) {
	return s.repo.GetCategories(ctx)
}

// clearCache 清除产品缓存
func (s *ProductService) clearCache(ctx context.Context) {
	s.rdb.Del(ctx, "products:list:*")
}
