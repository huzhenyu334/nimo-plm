package service

import (
	"context"
	"fmt"
	"time"

	"github.com/bitfantasy/nimo/internal/plm/entity"
	"github.com/bitfantasy/nimo/internal/plm/repository"
	"github.com/google/uuid"
)

type LangVariantService struct {
	variantRepo *repository.LangVariantRepository
	bomRepo     *repository.ProjectBOMRepository
}

func NewLangVariantService(variantRepo *repository.LangVariantRepository, bomRepo *repository.ProjectBOMRepository) *LangVariantService {
	return &LangVariantService{
		variantRepo: variantRepo,
		bomRepo:     bomRepo,
	}
}

type CreateLangVariantInput struct {
	LanguageCode   string `json:"language_code"`
	LanguageName   string `json:"language_name"`
	DesignFileID   string `json:"design_file_id"`
	DesignFileName string `json:"design_file_name"`
	DesignFileURL  string `json:"design_file_url"`
	Notes          string `json:"notes"`
}

type UpdateLangVariantInput struct {
	LanguageCode   *string `json:"language_code"`
	LanguageName   *string `json:"language_name"`
	DesignFileID   *string `json:"design_file_id"`
	DesignFileName *string `json:"design_file_name"`
	DesignFileURL  *string `json:"design_file_url"`
	Notes          *string `json:"notes"`
}

func (s *LangVariantService) ListByBOMItem(ctx context.Context, bomItemID string) ([]entity.BOMItemLangVariant, error) {
	return s.variantRepo.ListByBOMItem(ctx, bomItemID)
}

func (s *LangVariantService) Create(ctx context.Context, bomItemID string, input *CreateLangVariantInput) (*entity.BOMItemLangVariant, error) {
	nextIdx, err := s.variantRepo.GetNextVariantIndex(ctx, bomItemID)
	if err != nil {
		return nil, fmt.Errorf("获取变体序号失败: %w", err)
	}

	v := &entity.BOMItemLangVariant{
		ID:             uuid.New().String()[:32],
		BOMItemID:      bomItemID,
		VariantIndex:   nextIdx,
		LanguageCode:   input.LanguageCode,
		LanguageName:   input.LanguageName,
		DesignFileID:   input.DesignFileID,
		DesignFileName: input.DesignFileName,
		DesignFileURL:  input.DesignFileURL,
		Notes:          input.Notes,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.variantRepo.Create(ctx, v); err != nil {
		return nil, fmt.Errorf("创建语言变体失败: %w", err)
	}
	return v, nil
}

func (s *LangVariantService) Update(ctx context.Context, variantID string, input *UpdateLangVariantInput) (*entity.BOMItemLangVariant, error) {
	v, err := s.variantRepo.FindByID(ctx, variantID)
	if err != nil {
		return nil, fmt.Errorf("语言变体不存在: %w", err)
	}

	if input.LanguageCode != nil {
		v.LanguageCode = *input.LanguageCode
	}
	if input.LanguageName != nil {
		v.LanguageName = *input.LanguageName
	}
	if input.DesignFileID != nil {
		v.DesignFileID = *input.DesignFileID
	}
	if input.DesignFileName != nil {
		v.DesignFileName = *input.DesignFileName
	}
	if input.DesignFileURL != nil {
		v.DesignFileURL = *input.DesignFileURL
	}
	if input.Notes != nil {
		v.Notes = *input.Notes
	}
	v.UpdatedAt = time.Now()

	if err := s.variantRepo.Update(ctx, v); err != nil {
		return nil, fmt.Errorf("更新语言变体失败: %w", err)
	}
	return v, nil
}

func (s *LangVariantService) Delete(ctx context.Context, variantID string) error {
	_, err := s.variantRepo.FindByID(ctx, variantID)
	if err != nil {
		return fmt.Errorf("语言变体不存在: %w", err)
	}
	return s.variantRepo.Delete(ctx, variantID)
}

// GetMultilangParts 获取项目的所有多语言件及其语言变体
func (s *LangVariantService) GetMultilangParts(ctx context.Context, projectID string) ([]map[string]interface{}, error) {
	boms, err := s.bomRepo.ListByProject(ctx, projectID, "PBOM", "")
	if err != nil {
		return nil, fmt.Errorf("获取PBOM列表失败: %w", err)
	}

	var result []map[string]interface{}

	for _, bom := range boms {
		bomDetail, err := s.bomRepo.FindByID(ctx, bom.ID)
		if err != nil {
			continue
		}
		for _, item := range bomDetail.Items {
			if getExtAttr(item.ExtendedAttrs, "is_multilang") != "true" {
				continue
			}
			variants, _ := s.variantRepo.ListByBOMItem(ctx, item.ID)

			// 自动创建默认语言变体
			if len(variants) == 0 {
				defaultVariant := &entity.BOMItemLangVariant{
					ID:           uuid.New().String()[:32],
					BOMItemID:    item.ID,
					VariantIndex: 1,
					LanguageCode: "zh-CN",
					LanguageName: "简体中文",
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}
				if err := s.variantRepo.Create(ctx, defaultVariant); err == nil {
					variants = []entity.BOMItemLangVariant{*defaultVariant}
				}
			}

			result = append(result, map[string]interface{}{
				"bom_item":      item,
				"lang_variants": variants,
				"bom_id":        bom.ID,
				"bom_name":      bom.Name,
			})
		}
	}

	return result, nil
}
