package service

import (
	"context"
	"fmt"
	"time"

	"github.com/bitfantasy/nimo/internal/plm/entity"
	"github.com/bitfantasy/nimo/internal/plm/repository"
	"github.com/google/uuid"
)

// CMFService CMF服务
type CMFService struct {
	cmfRepo  *repository.CMFRepository
	bomRepo  *repository.ProjectBOMRepository
	taskRepo *repository.TaskRepository
}

func NewCMFService(cmfRepo *repository.CMFRepository, bomRepo *repository.ProjectBOMRepository, taskRepo *repository.TaskRepository) *CMFService {
	return &CMFService{cmfRepo: cmfRepo, bomRepo: bomRepo, taskRepo: taskRepo}
}

// GetAppearanceParts 获取外观件列表
// 查找项目的PBOM，过滤 is_appearance_part=true 的行项
func (s *CMFService) GetAppearanceParts(ctx context.Context, projectID, taskID string) ([]entity.ProjectBOMItem, error) {
	// 查找项目的PBOM（可能有多个，取最新的published或draft）
	boms, err := s.bomRepo.ListByProject(ctx, projectID, "PBOM", "")
	if err != nil {
		return nil, fmt.Errorf("查询SBOM失败: %w", err)
	}
	if len(boms) == 0 {
		return []entity.ProjectBOMItem{}, nil
	}

	// 优先选published的PBOM，否则用第一个
	var targetBOM *entity.ProjectBOM
	for i := range boms {
		if boms[i].Status == "published" {
			targetBOM = &boms[i]
			break
		}
	}
	if targetBOM == nil {
		targetBOM = &boms[0]
	}

	// 获取所有行项
	items, err := s.bomRepo.ListItemsByBOM(ctx, targetBOM.ID)
	if err != nil {
		return nil, fmt.Errorf("查询BOM行项失败: %w", err)
	}

	// 过滤外观件（排除CMF衍生零件）
	var parts []entity.ProjectBOMItem
	for _, item := range items {
		if getExtAttrBool(item.ExtendedAttrs, "is_appearance_part") && !getExtAttrBool(item.ExtendedAttrs, "is_variant") {
			parts = append(parts, item)
		}
	}
	if parts == nil {
		parts = []entity.ProjectBOMItem{}
	}
	return parts, nil
}

// ListDesigns 列出任务的所有CMF方案
func (s *CMFService) ListDesigns(ctx context.Context, projectID, taskID string) ([]entity.CMFDesign, error) {
	return s.cmfRepo.ListDesignsByTask(ctx, projectID, taskID)
}

// ListDesignsByProject 列出项目的所有CMF方案
func (s *CMFService) ListDesignsByProject(ctx context.Context, projectID string) ([]entity.CMFDesign, error) {
	return s.cmfRepo.ListDesignsByProject(ctx, projectID)
}

// CreateDesignInput 创建CMF方案输入
type CreateDesignInput struct {
	BOMItemID        string  `json:"bom_item_id" binding:"required"`
	SchemeName       string  `json:"scheme_name"`
	Color            string  `json:"color"`
	ColorCode        string  `json:"color_code"`
	GlossLevel       string  `json:"gloss_level"`
	SurfaceTreatment string  `json:"surface_treatment"`
	TexturePattern   string  `json:"texture_pattern"`
	CoatingType      string  `json:"coating_type"`
	RenderImageFileID   *string `json:"render_image_file_id"`
	RenderImageFileName string  `json:"render_image_file_name"`
	Notes            string  `json:"notes"`
	SortOrder        int     `json:"sort_order"`
}

// CreateDesign 创建CMF方案
func (s *CMFService) CreateDesign(ctx context.Context, projectID, taskID string, input *CreateDesignInput) (*entity.CMFDesign, error) {
	design := &entity.CMFDesign{
		ID:                  uuid.New().String()[:32],
		ProjectID:           projectID,
		TaskID:              taskID,
		BOMItemID:           input.BOMItemID,
		SchemeName:          input.SchemeName,
		Color:               input.Color,
		ColorCode:           input.ColorCode,
		GlossLevel:          input.GlossLevel,
		SurfaceTreatment:    input.SurfaceTreatment,
		TexturePattern:      input.TexturePattern,
		CoatingType:         input.CoatingType,
		RenderImageFileID:   input.RenderImageFileID,
		RenderImageFileName: input.RenderImageFileName,
		Notes:               input.Notes,
		SortOrder:           input.SortOrder,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
	if err := s.cmfRepo.CreateDesign(ctx, design); err != nil {
		return nil, fmt.Errorf("创建CMF方案失败: %w", err)
	}
	return s.cmfRepo.FindDesignByID(ctx, design.ID)
}

// UpdateDesignInput 更新CMF方案输入
type UpdateDesignInput struct {
	SchemeName       *string `json:"scheme_name"`
	Color            *string `json:"color"`
	ColorCode        *string `json:"color_code"`
	GlossLevel       *string `json:"gloss_level"`
	SurfaceTreatment *string `json:"surface_treatment"`
	TexturePattern   *string `json:"texture_pattern"`
	CoatingType      *string `json:"coating_type"`
	RenderImageFileID   *string `json:"render_image_file_id"`
	RenderImageFileName *string `json:"render_image_file_name"`
	Notes            *string `json:"notes"`
	SortOrder        *int    `json:"sort_order"`
}

// UpdateDesign 更新CMF方案
func (s *CMFService) UpdateDesign(ctx context.Context, designID string, input *UpdateDesignInput) (*entity.CMFDesign, error) {
	design, err := s.cmfRepo.FindDesignByID(ctx, designID)
	if err != nil {
		return nil, fmt.Errorf("CMF方案不存在: %w", err)
	}

	if input.SchemeName != nil {
		design.SchemeName = *input.SchemeName
	}
	if input.Color != nil {
		design.Color = *input.Color
	}
	if input.ColorCode != nil {
		design.ColorCode = *input.ColorCode
	}
	if input.GlossLevel != nil {
		design.GlossLevel = *input.GlossLevel
	}
	if input.SurfaceTreatment != nil {
		design.SurfaceTreatment = *input.SurfaceTreatment
	}
	if input.TexturePattern != nil {
		design.TexturePattern = *input.TexturePattern
	}
	if input.CoatingType != nil {
		design.CoatingType = *input.CoatingType
	}
	if input.RenderImageFileID != nil {
		design.RenderImageFileID = input.RenderImageFileID
	}
	if input.RenderImageFileName != nil {
		design.RenderImageFileName = *input.RenderImageFileName
	}
	if input.Notes != nil {
		design.Notes = *input.Notes
	}
	if input.SortOrder != nil {
		design.SortOrder = *input.SortOrder
	}
	design.UpdatedAt = time.Now()

	if err := s.cmfRepo.UpdateDesign(ctx, design); err != nil {
		return nil, fmt.Errorf("更新CMF方案失败: %w", err)
	}
	return s.cmfRepo.FindDesignByID(ctx, design.ID)
}

// DeleteDesign 删除CMF方案
func (s *CMFService) DeleteDesign(ctx context.Context, designID string) error {
	return s.cmfRepo.DeleteDesign(ctx, designID)
}

// AddDrawingInput 添加图纸输入
type AddDrawingInput struct {
	DrawingType string `json:"drawing_type"`
	FileID      string `json:"file_id" binding:"required"`
	FileName    string `json:"file_name" binding:"required"`
	Notes       string `json:"notes"`
}

// AddDrawing 添加图纸到CMF方案
func (s *CMFService) AddDrawing(ctx context.Context, designID string, input *AddDrawingInput) (*entity.CMFDrawing, error) {
	// 验证design存在
	if _, err := s.cmfRepo.FindDesignByID(ctx, designID); err != nil {
		return nil, fmt.Errorf("CMF方案不存在: %w", err)
	}

	drawing := &entity.CMFDrawing{
		ID:          uuid.New().String()[:32],
		CMFDesignID: designID,
		DrawingType: input.DrawingType,
		FileID:      input.FileID,
		FileName:    input.FileName,
		Notes:       input.Notes,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.cmfRepo.AddDrawing(ctx, drawing); err != nil {
		return nil, fmt.Errorf("添加图纸失败: %w", err)
	}
	return drawing, nil
}

// RemoveDrawing 删除图纸
func (s *CMFService) RemoveDrawing(ctx context.Context, drawingID string) error {
	return s.cmfRepo.RemoveDrawing(ctx, drawingID)
}
