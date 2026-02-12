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
	skuRepo *repository.SKURepository
	bomRepo *repository.ProjectBOMRepository
}

func NewSKUService(skuRepo *repository.SKURepository, bomRepo *repository.ProjectBOMRepository) *SKUService {
	return &SKUService{skuRepo: skuRepo, bomRepo: bomRepo}
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

// ========== BOM Override ==========

func (s *SKUService) GetBOMOverrides(ctx context.Context, skuID string) ([]entity.SKUBOMOverride, error) {
	return s.skuRepo.ListBOMOverrides(ctx, skuID)
}

func (s *SKUService) CreateBOMOverride(ctx context.Context, skuID string, input *BOMOverrideInput) (*entity.SKUBOMOverride, error) {
	override := &entity.SKUBOMOverride{
		ID:                    uuid.New().String()[:32],
		SKUID:                 skuID,
		Action:                input.Action,
		BaseItemID:            input.BaseItemID,
		OverrideName:          input.OverrideName,
		OverrideSpecification: input.OverrideSpecification,
		OverrideQuantity:      input.OverrideQuantity,
		OverrideUnit:          input.OverrideUnit,
		OverrideMaterialType:  input.OverrideMaterialType,
		OverrideProcessType:   input.OverrideProcessType,
		Notes:                 input.Notes,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
	if err := s.skuRepo.CreateBOMOverride(ctx, override); err != nil {
		return nil, fmt.Errorf("创建BOM差异失败: %w", err)
	}
	return override, nil
}

func (s *SKUService) UpdateBOMOverride(ctx context.Context, id string, input *BOMOverrideInput) (*entity.SKUBOMOverride, error) {
	override, err := s.skuRepo.FindBOMOverrideByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("BOM差异不存在: %w", err)
	}
	override.Action = input.Action
	override.BaseItemID = input.BaseItemID
	override.OverrideName = input.OverrideName
	override.OverrideSpecification = input.OverrideSpecification
	override.OverrideQuantity = input.OverrideQuantity
	override.OverrideUnit = input.OverrideUnit
	override.OverrideMaterialType = input.OverrideMaterialType
	override.OverrideProcessType = input.OverrideProcessType
	override.Notes = input.Notes
	override.UpdatedAt = time.Now()
	if err := s.skuRepo.UpdateBOMOverride(ctx, override); err != nil {
		return nil, fmt.Errorf("更新BOM差异失败: %w", err)
	}
	return override, nil
}

func (s *SKUService) DeleteBOMOverride(ctx context.Context, id string) error {
	return s.skuRepo.DeleteBOMOverride(ctx, id)
}

// ========== Full BOM (base + overrides merged) ==========

func (s *SKUService) GetFullBOM(ctx context.Context, skuID string, projectID string) ([]map[string]interface{}, error) {
	// Get the SKU
	sku, err := s.skuRepo.FindByID(ctx, skuID)
	if err != nil {
		return nil, fmt.Errorf("SKU不存在: %w", err)
	}

	// Get the SBOM for the project
	boms, err := s.bomRepo.ListByProject(ctx, projectID, "SBOM", "")
	if err != nil {
		return nil, fmt.Errorf("获取SBOM失败: %w", err)
	}
	if len(boms) == 0 {
		return []map[string]interface{}{}, nil
	}

	// Get the latest SBOM
	bom, err := s.bomRepo.FindByID(ctx, boms[0].ID)
	if err != nil {
		return nil, fmt.Errorf("获取SBOM详情失败: %w", err)
	}

	// Build base item map
	baseItems := make(map[string]entity.ProjectBOMItem)
	for _, item := range bom.Items {
		baseItems[item.ID] = item
	}

	// Apply overrides
	removedIDs := make(map[string]bool)
	replacedIDs := make(map[string]entity.SKUBOMOverride)
	var addedItems []entity.SKUBOMOverride

	for _, o := range sku.BOMOverrides {
		switch o.Action {
		case "remove":
			if o.BaseItemID != nil {
				removedIDs[*o.BaseItemID] = true
			}
		case "replace":
			if o.BaseItemID != nil {
				replacedIDs[*o.BaseItemID] = o
			}
		case "add":
			addedItems = append(addedItems, o)
		}
	}

	// Build CMF map
	cmfMap := make(map[string]entity.SKUCMFConfig)
	for _, c := range sku.CMFConfigs {
		cmfMap[c.BOMItemID] = c
	}

	var result []map[string]interface{}
	for _, item := range bom.Items {
		if removedIDs[item.ID] {
			continue
		}
		entry := map[string]interface{}{
			"id":            item.ID,
			"item_number":   item.ItemNumber,
			"name":          item.Name,
			"specification": item.Specification,
			"quantity":      item.Quantity,
			"unit":          item.Unit,
			"category":      item.Category,
			"material_type": item.MaterialType,
			"process_type":  item.ProcessType,
			"source":        "base",
		}

		// Apply CMF
		if cmf, ok := cmfMap[item.ID]; ok {
			entry["color"] = cmf.Color
			entry["color_code"] = cmf.ColorCode
			entry["surface_treatment"] = cmf.SurfaceTreatment
		}

		// Apply replacement
		if repl, ok := replacedIDs[item.ID]; ok {
			entry["name"] = repl.OverrideName
			entry["specification"] = repl.OverrideSpecification
			entry["quantity"] = repl.OverrideQuantity
			entry["unit"] = repl.OverrideUnit
			entry["material_type"] = repl.OverrideMaterialType
			entry["process_type"] = repl.OverrideProcessType
			entry["source"] = "replaced"
		}

		result = append(result, entry)
	}

	// Add new items
	for _, added := range addedItems {
		result = append(result, map[string]interface{}{
			"id":            added.ID,
			"name":          added.OverrideName,
			"specification": added.OverrideSpecification,
			"quantity":      added.OverrideQuantity,
			"unit":          added.OverrideUnit,
			"material_type": added.OverrideMaterialType,
			"process_type":  added.OverrideProcessType,
			"source":        "added",
		})
	}

	return result, nil
}

// ========== Input DTOs ==========

type CreateSKUInput struct {
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
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

type BOMOverrideInput struct {
	Action                string  `json:"action" binding:"required"`
	BaseItemID            *string `json:"base_item_id"`
	OverrideName          string  `json:"override_name"`
	OverrideSpecification string  `json:"override_specification"`
	OverrideQuantity      float64 `json:"override_quantity"`
	OverrideUnit          string  `json:"override_unit"`
	OverrideMaterialType  string  `json:"override_material_type"`
	OverrideProcessType   string  `json:"override_process_type"`
	Notes                 string  `json:"notes"`
}
