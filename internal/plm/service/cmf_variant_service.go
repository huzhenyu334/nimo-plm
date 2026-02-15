package service

import (
	"context"
	"fmt"
	"time"

	"github.com/bitfantasy/nimo/internal/plm/entity"
	"github.com/bitfantasy/nimo/internal/plm/repository"
	"github.com/google/uuid"
)

type CMFVariantService struct {
	variantRepo *repository.CMFVariantRepository
	bomRepo     *repository.ProjectBOMRepository
}

func NewCMFVariantService(variantRepo *repository.CMFVariantRepository, bomRepo *repository.ProjectBOMRepository) *CMFVariantService {
	return &CMFVariantService{
		variantRepo: variantRepo,
		bomRepo:     bomRepo,
	}
}

// CreateVariantInput 创建CMF变体请求
type CreateVariantInput struct {
	ColorHex             string `json:"color_hex"`
	Finish               string `json:"finish"`
	Texture              string `json:"texture"`
	Coating              string `json:"coating"`
	PantoneCode          string `json:"pantone_code"`
	ReferenceImageFileID string `json:"reference_image_file_id"`
	ReferenceImageURL    string `json:"reference_image_url"`
	ProcessDrawingType   string `json:"process_drawing_type"`
	ProcessDrawings      string `json:"process_drawings"`
	Notes                string `json:"notes"`
}

// UpdateVariantInput 更新CMF变体请求
type UpdateVariantInput struct {
	ColorHex             *string `json:"color_hex"`
	Finish               *string `json:"finish"`
	Texture              *string `json:"texture"`
	Coating              *string `json:"coating"`
	PantoneCode          *string `json:"pantone_code"`
	GlossLevel           *string `json:"gloss_level"`
	ReferenceImageFileID *string `json:"reference_image_file_id"`
	ReferenceImageURL    *string `json:"reference_image_url"`
	ProcessDrawingType   *string `json:"process_drawing_type"`
	ProcessDrawings      *string `json:"process_drawings"`
	Notes                *string `json:"notes"`
}

// ListByBOMItem 获取零件的所有CMF变体
func (s *CMFVariantService) ListByBOMItem(ctx context.Context, bomItemID string) ([]entity.BOMItemCMFVariant, error) {
	return s.variantRepo.ListByBOMItem(ctx, bomItemID)
}

// Create 创建CMF变体
func (s *CMFVariantService) Create(ctx context.Context, bomItemID string, input *CreateVariantInput) (*entity.BOMItemCMFVariant, error) {
	item, err := s.bomRepo.FindItemByID(ctx, bomItemID)
	if err != nil {
		return nil, fmt.Errorf("零件不存在: %w", err)
	}
	if !getExtAttrBool(item.ExtendedAttrs, "is_appearance_part") {
		return nil, fmt.Errorf("只有外观件才能添加CMF变体")
	}

	nextIndex, err := s.variantRepo.GetNextVariantIndex(ctx, bomItemID)
	if err != nil {
		return nil, fmt.Errorf("获取变体序号失败: %w", err)
	}

	materialCode := s.generateMaterialCode(item, nextIndex)

	// 材质从BOM主零件继承
	material := getExtAttr(item.ExtendedAttrs, "material_type")

	variant := &entity.BOMItemCMFVariant{
		ID:                   uuid.New().String()[:32],
		BOMItemID:            bomItemID,
		VariantIndex:         nextIndex,
		MaterialCode:         materialCode,
		ColorHex:             input.ColorHex,
		Material:             material,
		Finish:               input.Finish,
		Texture:              input.Texture,
		Coating:              input.Coating,
		PantoneCode:          input.PantoneCode,
		ReferenceImageFileID: input.ReferenceImageFileID,
		ReferenceImageURL:    input.ReferenceImageURL,
		ProcessDrawingType:   input.ProcessDrawingType,
		ProcessDrawings:      input.ProcessDrawings,
		Notes:                input.Notes,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	if err := s.variantRepo.Create(ctx, variant); err != nil {
		return nil, fmt.Errorf("创建CMF变体失败: %w", err)
	}

	return variant, nil
}

// Update 更新CMF变体 (material不可修改，从BOM继承)
func (s *CMFVariantService) Update(ctx context.Context, variantID string, input *UpdateVariantInput) (*entity.BOMItemCMFVariant, error) {
	variant, err := s.variantRepo.FindByID(ctx, variantID)
	if err != nil {
		return nil, fmt.Errorf("CMF变体不存在: %w", err)
	}

	if input.ColorHex != nil {
		variant.ColorHex = *input.ColorHex
	}
	if input.Finish != nil {
		variant.Finish = *input.Finish
	}
	if input.Texture != nil {
		variant.Texture = *input.Texture
	}
	if input.Coating != nil {
		variant.Coating = *input.Coating
	}
	if input.PantoneCode != nil {
		variant.PantoneCode = *input.PantoneCode
	}
	if input.GlossLevel != nil {
		variant.GlossLevel = *input.GlossLevel
	}
	if input.ReferenceImageFileID != nil {
		variant.ReferenceImageFileID = *input.ReferenceImageFileID
	}
	if input.ReferenceImageURL != nil {
		variant.ReferenceImageURL = *input.ReferenceImageURL
	}
	if input.ProcessDrawingType != nil {
		variant.ProcessDrawingType = *input.ProcessDrawingType
	}
	if input.ProcessDrawings != nil {
		variant.ProcessDrawings = *input.ProcessDrawings
	}
	if input.Notes != nil {
		variant.Notes = *input.Notes
	}
	variant.UpdatedAt = time.Now()

	if err := s.variantRepo.Update(ctx, variant); err != nil {
		return nil, fmt.Errorf("更新CMF变体失败: %w", err)
	}

	return variant, nil
}

// Delete 删除CMF变体
func (s *CMFVariantService) Delete(ctx context.Context, variantID string) error {
	if _, err := s.variantRepo.FindByID(ctx, variantID); err != nil {
		return fmt.Errorf("CMF变体不存在: %w", err)
	}

	if err := s.variantRepo.Delete(ctx, variantID); err != nil {
		return fmt.Errorf("删除CMF变体失败: %w", err)
	}

	return nil
}

// GetAppearanceParts 获取项目所有外观件及其CMF变体
// 从EBOM中查找 category=structural 且 is_appearance_part=true 的结构件
// 如果外观件没有CMF变体，自动创建一条默认的draft变体
func (s *CMFVariantService) GetAppearanceParts(ctx context.Context, projectID string) ([]map[string]interface{}, error) {
	boms, err := s.bomRepo.ListByProject(ctx, projectID, "EBOM", "")
	if err != nil {
		return nil, fmt.Errorf("获取BOM列表失败: %w", err)
	}

	var result []map[string]interface{}

	for _, bom := range boms {
		bomDetail, err := s.bomRepo.FindByID(ctx, bom.ID)
		if err != nil {
			continue
		}
		for _, item := range bomDetail.Items {
			// Only structural parts with is_appearance_part=true
			if item.Category != "structural" {
				continue
			}
			if !getExtAttrBool(item.ExtendedAttrs, "is_appearance_part") || getExtAttrBool(item.ExtendedAttrs, "is_variant") {
				continue
			}
			variants, _ := s.variantRepo.ListByBOMItem(ctx, item.ID)

			// 自动创建默认CMF变体(#7: 默认1条)
			if len(variants) == 0 {
				materialCode := s.generateMaterialCode(&item, 1)
				material := getExtAttr(item.ExtendedAttrs, "material_type")
				defaultVariant := &entity.BOMItemCMFVariant{
					ID:           uuid.New().String()[:32],
					BOMItemID:    item.ID,
					VariantIndex: 1,
					MaterialCode: materialCode,
					Material:     material,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}
				if err := s.variantRepo.Create(ctx, defaultVariant); err == nil {
					variants = []entity.BOMItemCMFVariant{*defaultVariant}
				}
			}

			result = append(result, map[string]interface{}{
				"bom_item":     item,
				"cmf_variants": variants,
				"bom_id":       bom.ID,
				"bom_name":     bom.Name,
			})
		}
	}

	return result, nil
}

// GetSRMItems 获取可采购项（含CMF变体展开）
// 从所有BOM类型中获取物料（EBOM + PBOM）
func (s *CMFVariantService) GetSRMItems(ctx context.Context, projectID string) ([]map[string]interface{}, error) {
	var result []map[string]interface{}

	// Query all BOM types
	for _, bomType := range []string{"EBOM", "PBOM"} {
		boms, err := s.bomRepo.ListByProject(ctx, projectID, bomType, "")
		if err != nil {
			continue
		}

		for _, bom := range boms {
			bomDetail, err := s.bomRepo.FindByID(ctx, bom.ID)
			if err != nil {
				continue
			}
			for _, item := range bomDetail.Items {
				if getExtAttrBool(item.ExtendedAttrs, "is_variant") {
					continue
				}

				if !getExtAttrBool(item.ExtendedAttrs, "is_appearance_part") {
					materialCode := ""
					if item.Material != nil {
						materialCode = item.Material.Code
					}
					result = append(result, map[string]interface{}{
						"type":          "standard",
						"bom_item_id":   item.ID,
						"material_code": materialCode,
						"name":          item.Name,
						"quantity":      item.Quantity,
						"unit":          item.Unit,
						"category":      item.Category,
						"sub_category":  item.SubCategory,
						"bom_type":      bomType,
						"bom_item":      item,
					})
				} else {
					variants, _ := s.variantRepo.ListByBOMItem(ctx, item.ID)
					for _, v := range variants {
						result = append(result, map[string]interface{}{
							"type":           "cmf_variant",
							"bom_item_id":    item.ID,
							"cmf_variant_id": v.ID,
							"material_code":  v.MaterialCode,
							"name":           item.Name,
							"quantity":       item.Quantity,
							"unit":           item.Unit,
							"category":       item.Category,
							"sub_category":   item.SubCategory,
							"bom_type":       bomType,
							"cmf": map[string]interface{}{
								"material":     v.Material,
								"finish":       v.Finish,
								"color_hex":    v.ColorHex,
								"texture":      v.Texture,
								"coating":      v.Coating,
								"pantone_code": v.PantoneCode,
							},
							"bom_item":    item,
							"cmf_variant": v,
						})
					}
				}
			}
		}
	}

	return result, nil
}

// generateMaterialCode 生成CMF变体料号 (序号制，不依赖颜色名)
func (s *CMFVariantService) generateMaterialCode(item *entity.ProjectBOMItem, index int) string {
	baseCode := ""
	if item.Material != nil {
		baseCode = item.Material.Code
	}
	if baseCode == "" {
		baseCode = fmt.Sprintf("AP-%03d", item.ItemNumber)
	}
	return fmt.Sprintf("%s-V%02d", baseCode, index)
}

// getExtAttr extracts a string value from JSONB extended_attrs
func getExtAttr(attrs entity.JSONB, key string) string {
	if attrs == nil {
		return ""
	}
	if v, ok := attrs[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	return ""
}

// getExtAttrBool extracts a boolean value from JSONB extended_attrs
func getExtAttrBool(attrs entity.JSONB, key string) bool {
	if attrs == nil {
		return false
	}
	v, ok := attrs[key]
	if !ok {
		return false
	}
	switch b := v.(type) {
	case bool:
		return b
	case float64:
		return b != 0
	case string:
		return b == "true" || b == "1" || b == "Y" || b == "是"
	default:
		return false
	}
}

// setExtAttr sets a value in JSONB extended_attrs, initializing if nil
func setExtAttr(attrs *entity.JSONB, key string, value interface{}) {
	if *attrs == nil {
		*attrs = entity.JSONB{}
	}
	(*attrs)[key] = value
}
