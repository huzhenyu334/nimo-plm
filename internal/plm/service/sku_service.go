package service

import (
	"context"
	"fmt"
	"time"

	"github.com/bitfantasy/nimo/internal/plm/entity"
	"github.com/bitfantasy/nimo/internal/plm/repository"
	"github.com/google/uuid"
)

type SKUService struct {
	skuRepo     *repository.SKURepository
	bomRepo     *repository.ProjectBOMRepository
	variantRepo *repository.CMFVariantRepository
}

func NewSKUService(skuRepo *repository.SKURepository, bomRepo *repository.ProjectBOMRepository, variantRepo *repository.CMFVariantRepository) *SKUService {
	return &SKUService{skuRepo: skuRepo, bomRepo: bomRepo, variantRepo: variantRepo}
}

// ========== SKU CRUD ==========

func (s *SKUService) ListSKUs(ctx context.Context, projectID string) ([]entity.ProductSKU, error) {
	return s.skuRepo.ListByProject(ctx, projectID)
}

func (s *SKUService) GetSKU(ctx context.Context, id string) (*entity.ProductSKU, error) {
	return s.skuRepo.FindByID(ctx, id)
}

func (s *SKUService) CreateSKU(ctx context.Context, projectID string, input *CreateSKUInput, createdBy string) (*entity.ProductSKU, error) {
	sku := &entity.ProductSKU{
		ID:          uuid.New().String()[:32],
		ProjectID:   projectID,
		Name:        input.Name,
		Code:        input.Code,
		Description: input.Description,
		Status:      "active",
		SortOrder:   input.SortOrder,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.skuRepo.Create(ctx, sku); err != nil {
		return nil, fmt.Errorf("创建SKU失败: %w", err)
	}

	// 如果传入了BOM items，一步保存
	if len(input.BOMItems) > 0 {
		s.BatchSaveBOMItems(ctx, sku.ID, input.BOMItems)
	}

	return sku, nil
}

func (s *SKUService) UpdateSKU(ctx context.Context, id string, input *UpdateSKUInput) (*entity.ProductSKU, error) {
	sku, err := s.skuRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("SKU不存在: %w", err)
	}
	if input.Name != nil {
		sku.Name = *input.Name
	}
	if input.Code != nil {
		sku.Code = *input.Code
	}
	if input.Description != nil {
		sku.Description = *input.Description
	}
	if input.Status != nil {
		sku.Status = *input.Status
	}
	if input.SortOrder != nil {
		sku.SortOrder = *input.SortOrder
	}
	sku.UpdatedAt = time.Now()
	if err := s.skuRepo.Update(ctx, sku); err != nil {
		return nil, fmt.Errorf("更新SKU失败: %w", err)
	}
	return sku, nil
}

func (s *SKUService) DeleteSKU(ctx context.Context, id string) error {
	return s.skuRepo.Delete(ctx, id)
}

// ========== CMF Config ==========

func (s *SKUService) GetCMFConfigs(ctx context.Context, skuID string) ([]entity.SKUCMFConfig, error) {
	return s.skuRepo.ListCMFConfigs(ctx, skuID)
}

func (s *SKUService) BatchSaveCMFConfigs(ctx context.Context, skuID string, inputs []CMFConfigInput) ([]entity.SKUCMFConfig, error) {
	configs := make([]entity.SKUCMFConfig, len(inputs))
	now := time.Now()
	for i, inp := range inputs {
		configs[i] = entity.SKUCMFConfig{
			ID:               uuid.New().String()[:32],
			SKUID:            skuID,
			BOMItemID:        inp.BOMItemID,
			Color:            inp.Color,
			ColorCode:        inp.ColorCode,
			SurfaceTreatment: inp.SurfaceTreatment,
			ProcessParams:    inp.ProcessParams,
			Notes:            inp.Notes,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
	}
	if err := s.skuRepo.BatchSaveCMFConfigs(ctx, skuID, configs); err != nil {
		return nil, fmt.Errorf("保存CMF配置失败: %w", err)
	}
	return s.skuRepo.ListCMFConfigs(ctx, skuID)
}

// ========== SKU BOM Items（从SBOM勾选零件） ==========

func (s *SKUService) GetBOMItems(ctx context.Context, skuID string) ([]entity.SKUBOMItem, error) {
	return s.skuRepo.ListBOMItems(ctx, skuID)
}

// BatchSaveBOMItems 批量保存SKU勾选的BOM零件（传入bom_item_id列表）
func (s *SKUService) BatchSaveBOMItems(ctx context.Context, skuID string, inputs []SKUBOMItemInput) ([]entity.SKUBOMItem, error) {
	items := make([]entity.SKUBOMItem, len(inputs))
	now := time.Now()
	for i, inp := range inputs {
		item := entity.SKUBOMItem{
			ID:        uuid.New().String()[:32],
			SKUID:     skuID,
			BOMItemID: inp.BOMItemID,
			Quantity:  inp.Quantity,
			Notes:     inp.Notes,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if inp.CMFVariantID != "" {
			vid := inp.CMFVariantID
			item.CMFVariantID = &vid
		}
		items[i] = item
	}
	if err := s.skuRepo.BatchSaveBOMItems(ctx, skuID, items); err != nil {
		return nil, fmt.Errorf("保存SKU BOM勾选失败: %w", err)
	}
	return s.skuRepo.ListBOMItems(ctx, skuID)
}

// ========== Full BOM (SBOM中该SKU勾选的零件 + CMF) ==========

func (s *SKUService) GetFullBOM(ctx context.Context, skuID string, projectID string) ([]map[string]interface{}, error) {
	sku, err := s.skuRepo.FindByID(ctx, skuID)
	if err != nil {
		return nil, fmt.Errorf("SKU不存在: %w", err)
	}

	// Build selected BOM item IDs
	selectedIDs := make(map[string]entity.SKUBOMItem)
	for _, item := range sku.BOMItems {
		selectedIDs[item.BOMItemID] = item
	}

	// Get the PBOM
	boms, err := s.bomRepo.ListByProject(ctx, projectID, "PBOM", "")
	if err != nil {
		return nil, fmt.Errorf("获取PBOM失败: %w", err)
	}
	if len(boms) == 0 {
		return []map[string]interface{}{}, nil
	}

	bom, err := s.bomRepo.FindByID(ctx, boms[0].ID)
	if err != nil {
		return nil, fmt.Errorf("获取SBOM详情失败: %w", err)
	}

	// Build CMF map
	cmfMap := make(map[string]entity.SKUCMFConfig)
	for _, c := range sku.CMFConfigs {
		cmfMap[c.BOMItemID] = c
	}

	// Filter: only items selected by this SKU
	var result []map[string]interface{}
	for _, item := range bom.Items {
		skuItem, selected := selectedIDs[item.ID]
		if !selected {
			continue
		}

		qty := item.Quantity
		if skuItem.Quantity > 0 {
			qty = skuItem.Quantity
		}

		entry := map[string]interface{}{
			"id":                item.ID,
			"item_number":       item.ItemNumber,
			"name":              item.Name,
			"specification":     getExtAttr(item.ExtendedAttrs, "specification"),
			"quantity":          qty,
			"unit":              item.Unit,
			"category":          item.Category,
			"material_type":     getExtAttr(item.ExtendedAttrs, "material_type"),
			"process_type":      getExtAttr(item.ExtendedAttrs, "process_type"),
			"is_appearance_part": getExtAttrBool(item.ExtendedAttrs, "is_appearance_part"),
		}

		// 如果关联了CMF变体，查询完整CMF数据
		if skuItem.CMFVariantID != nil && *skuItem.CMFVariantID != "" {
			entry["cmf_variant_id"] = *skuItem.CMFVariantID
			if variant, err := s.variantRepo.FindByID(ctx, *skuItem.CMFVariantID); err == nil {
				entry["cmf_variant"] = variant
				entry["color_hex"] = variant.ColorHex
				entry["material_code"] = variant.MaterialCode
				entry["finish"] = variant.Finish
				entry["texture"] = variant.Texture
				entry["coating"] = variant.Coating
			}
		} else if cmf, ok := cmfMap[item.ID]; ok {
			entry["color"] = cmf.Color
			entry["color_code"] = cmf.ColorCode
			entry["surface_treatment"] = cmf.SurfaceTreatment
		}

		result = append(result, entry)
	}

	return result, nil
}

// ========== Input DTOs ==========

type CreateSKUInput struct {
	Name        string            `json:"name" binding:"required"`
	Code        string            `json:"code"`
	Description string            `json:"description"`
	SortOrder   int               `json:"sort_order"`
	BOMItems    []SKUBOMItemInput `json:"bom_items"`
}

type UpdateSKUInput struct {
	Name        *string `json:"name"`
	Code        *string `json:"code"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	SortOrder   *int    `json:"sort_order"`
}

type CMFConfigInput struct {
	BOMItemID        string `json:"bom_item_id" binding:"required"`
	Color            string `json:"color"`
	ColorCode        string `json:"color_code"`
	SurfaceTreatment string `json:"surface_treatment"`
	ProcessParams    string `json:"process_params"`
	Notes            string `json:"notes"`
}

type SKUBOMItemInput struct {
	BOMItemID    string  `json:"bom_item_id" binding:"required"`
	CMFVariantID string  `json:"cmf_variant_id"`
	Quantity     float64 `json:"quantity"` // 0=使用SBOM默认数量
	Notes        string  `json:"notes"`
}
