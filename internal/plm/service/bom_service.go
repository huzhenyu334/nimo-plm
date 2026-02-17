package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bitfantasy/nimo/internal/plm/entity"
	"github.com/bitfantasy/nimo/internal/plm/repository"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type ProjectBOMService struct {
	bomRepo         *repository.ProjectBOMRepository
	projectRepo     *repository.ProjectRepository
	deliverableRepo *repository.DeliverableRepository
	materialRepo    *repository.MaterialRepository
	partDrawingRepo *repository.PartDrawingRepository
}

func NewProjectBOMService(bomRepo *repository.ProjectBOMRepository, projectRepo *repository.ProjectRepository, deliverableRepo *repository.DeliverableRepository, materialRepo *repository.MaterialRepository, partDrawingRepo *repository.PartDrawingRepository) *ProjectBOMService {
	return &ProjectBOMService{
		bomRepo:         bomRepo,
		projectRepo:     projectRepo,
		deliverableRepo: deliverableRepo,
		materialRepo:    materialRepo,
		partDrawingRepo: partDrawingRepo,
	}
}

// CreateBOM 创建BOM（草稿状态）
func (s *ProjectBOMService) CreateBOM(ctx context.Context, projectID string, input *CreateBOMInput, createdBy string) (*entity.ProjectBOM, error) {
	bom := &entity.ProjectBOM{
		ID:          uuid.New().String()[:32],
		ProjectID:   projectID,
		PhaseID:     input.PhaseID,
		TaskID:      input.TaskID,
		BOMType:     input.BOMType,
		Version:     input.Version,
		Name:        input.Name,
		Status:      "draft",
		Description: input.Description,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Draft BOM has no version yet
	if bom.Version == "" {
		bom.Version = ""
	}
	// Auto-generate name if not provided
	if bom.Name == "" {
		bom.Name = bom.BOMType
	}

	if err := s.bomRepo.Create(ctx, bom); err != nil {
		return nil, fmt.Errorf("create bom: %w", err)
	}

	return bom, nil
}

// GetBOM 获取BOM详情（含行项）
func (s *ProjectBOMService) GetBOM(ctx context.Context, id string) (*entity.ProjectBOM, error) {
	return s.bomRepo.FindByID(ctx, id)
}

// ListBOMs 获取项目BOM列表
func (s *ProjectBOMService) ListBOMs(ctx context.Context, projectID, bomType, status string) ([]entity.ProjectBOM, error) {
	return s.bomRepo.ListByProject(ctx, projectID, bomType, status)
}

// UpdateBOM 更新BOM基本信息（仅草稿状态可改）
func (s *ProjectBOMService) UpdateBOM(ctx context.Context, id string, input *UpdateBOMInput) (*entity.ProjectBOM, error) {
	bom, err := s.bomRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("bom not found: %w", err)
	}

	if bom.Status != "draft" && bom.Status != "rejected" {
		return nil, fmt.Errorf("只有草稿或被驳回的BOM才能编辑")
	}

	if input.Name != "" {
		bom.Name = input.Name
	}
	if input.Description != "" {
		bom.Description = input.Description
	}
	if input.Version != "" {
		bom.Version = input.Version
	}

	if err := s.bomRepo.Update(ctx, bom); err != nil {
		return nil, fmt.Errorf("update bom: %w", err)
	}
	return bom, nil
}

// SubmitBOM 提交BOM审批
func (s *ProjectBOMService) SubmitBOM(ctx context.Context, id, submitterID string) (*entity.ProjectBOM, error) {
	bom, err := s.bomRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("bom not found: %w", err)
	}

	if bom.Status != "draft" && bom.Status != "rejected" {
		return nil, fmt.Errorf("只有草稿或被驳回的BOM才能提交审批")
	}

	count, _ := s.bomRepo.CountItems(ctx, id)
	if count == 0 {
		return nil, fmt.Errorf("BOM没有物料行项，无法提交")
	}

	now := time.Now()
	bom.Status = "pending_review"
	bom.SubmittedBy = &submitterID
	bom.SubmittedAt = &now

	if err := s.bomRepo.Update(ctx, bom); err != nil {
		return nil, fmt.Errorf("submit bom: %w", err)
	}
	return bom, nil
}

// ApproveBOM 审批通过BOM
func (s *ProjectBOMService) ApproveBOM(ctx context.Context, id, reviewerID, comment string) (*entity.ProjectBOM, error) {
	bom, err := s.bomRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("bom not found: %w", err)
	}

	if bom.Status != "pending_review" {
		return nil, fmt.Errorf("只有待审批的BOM才能审批")
	}

	now := time.Now()
	bom.Status = "published"
	bom.ReviewedBy = &reviewerID
	bom.ReviewedAt = &now
	bom.ReviewComment = comment
	bom.ApprovedBy = &reviewerID
	bom.ApprovedAt = &now

	if err := s.bomRepo.Update(ctx, bom); err != nil {
		return nil, fmt.Errorf("approve bom: %w", err)
	}
	return bom, nil
}

// RejectBOM 驳回BOM
func (s *ProjectBOMService) RejectBOM(ctx context.Context, id, reviewerID, comment string) (*entity.ProjectBOM, error) {
	bom, err := s.bomRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("bom not found: %w", err)
	}

	if bom.Status != "pending_review" {
		return nil, fmt.Errorf("只有待审批的BOM才能驳回")
	}

	now := time.Now()
	bom.Status = "rejected"
	bom.ReviewedBy = &reviewerID
	bom.ReviewedAt = &now
	bom.ReviewComment = comment

	if err := s.bomRepo.Update(ctx, bom); err != nil {
		return nil, fmt.Errorf("reject bom: %w", err)
	}
	return bom, nil
}

// FreezeBOM 冻结BOM
func (s *ProjectBOMService) FreezeBOM(ctx context.Context, id, frozenByID string) (*entity.ProjectBOM, error) {
	bom, err := s.bomRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("bom not found: %w", err)
	}

	if bom.Status != "published" {
		return nil, fmt.Errorf("只有已发布的BOM才能冻结")
	}

	now := time.Now()
	bom.Status = "frozen"
	bom.FrozenAt = &now
	bom.FrozenBy = &frozenByID

	if err := s.bomRepo.Update(ctx, bom); err != nil {
		return nil, fmt.Errorf("freeze bom: %w", err)
	}

	s.CreateBOMRelease(ctx, bom)

	return bom, nil
}

// AddItem 添加BOM行项
func (s *ProjectBOMService) AddItem(ctx context.Context, bomID string, input *BOMItemInput) (*entity.ProjectBOMItem, error) {
	bom, err := s.bomRepo.FindByID(ctx, bomID)
	if err != nil {
		return nil, fmt.Errorf("bom not found: %w", err)
	}
	if bom.Status != "draft" && bom.Status != "rejected" {
		return nil, fmt.Errorf("只有草稿状态的BOM才能添加物料")
	}

	item := &entity.ProjectBOMItem{
		ID:            uuid.New().String()[:32],
		BOMID:         bomID,
		ItemNumber:    input.ItemNumber,
		ParentItemID:  input.ParentItemID,
		Level:         input.Level,
		MaterialID:    input.MaterialID,
		Category:      input.Category,
		SubCategory:   input.SubCategory,
		Name:          input.Name,
		Quantity:      input.Quantity,
		Unit:          input.Unit,
		Supplier:       input.Supplier,
		SupplierID:     input.SupplierID,
		ManufacturerID: input.ManufacturerID,
		MPN:            input.MPN,
		UnitPrice:      input.UnitPrice,
		IsAlternative:  input.IsAlternative,
		ThumbnailURL:  input.ThumbnailURL,
		Notes:         input.Notes,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if input.ExtendedAttrs != nil {
		item.ExtendedAttrs = entity.JSONB(input.ExtendedAttrs)
	}
	// Copy API compat fields to extended_attrs
	if input.Specification != "" {
		setExtAttr(&item.ExtendedAttrs, "specification", input.Specification)
	}
	if input.Reference != "" {
		setExtAttr(&item.ExtendedAttrs, "reference", input.Reference)
	}
	if input.Manufacturer != "" {
		setExtAttr(&item.ExtendedAttrs, "manufacturer", input.Manufacturer)
	}
	if input.ManufacturerPN != "" {
		setExtAttr(&item.ExtendedAttrs, "manufacturer_pn", input.ManufacturerPN)
	}
	if input.SupplierPN != "" {
		setExtAttr(&item.ExtendedAttrs, "supplier_pn", input.SupplierPN)
	}
	if input.DrawingNo != "" {
		setExtAttr(&item.ExtendedAttrs, "drawing_no", input.DrawingNo)
	}
	if input.IsCritical {
		setExtAttr(&item.ExtendedAttrs, "is_critical", true)
	}
	if input.LeadTimeDays != nil {
		setExtAttr(&item.ExtendedAttrs, "lead_time_days", *input.LeadTimeDays)
	}
	if input.IsAppearancePart {
		setExtAttr(&item.ExtendedAttrs, "is_appearance_part", true)
	}

	if item.Unit == "" {
		item.Unit = "pcs"
	}

	if input.UnitPrice != nil {
		extCost := input.Quantity * *input.UnitPrice
		item.ExtendedCost = &extCost
	}

	if item.MaterialID == nil && item.Name != "" {
		category := item.Category
		if category == "" {
			category = defaultCategoryForBOMType(bom.BOMType)
		}
		newMat, createErr := s.autoCreateMaterial(ctx, item.Name, input.Specification, category, input.Manufacturer, input.ManufacturerPN)
		if createErr != nil {
			fmt.Printf("[WARN] auto-create material failed for %q: %v\n", item.Name, createErr)
		} else if newMat != nil {
			item.MaterialID = &newMat.ID
		}
	}

	if err := s.bomRepo.CreateItem(ctx, item); err != nil {
		return nil, fmt.Errorf("create bom item: %w", err)
	}

	s.updateBOMCost(ctx, bomID)

	created, _ := s.bomRepo.FindItemByID(ctx, item.ID)
	if created != nil {
		return created, nil
	}
	return item, nil
}

// BatchAddItems 批量添加BOM行项
func (s *ProjectBOMService) BatchAddItems(ctx context.Context, bomID string, items []BOMItemInput) (int, error) {
	bom, err := s.bomRepo.FindByID(ctx, bomID)
	if err != nil {
		return 0, fmt.Errorf("bom not found: %w", err)
	}
	if bom.Status != "draft" && bom.Status != "rejected" {
		return 0, fmt.Errorf("只有草稿状态的BOM才能添加物料")
	}

	var entities []entity.ProjectBOMItem
	for i, input := range items {
		item := entity.ProjectBOMItem{
			ID:            uuid.New().String()[:32],
			BOMID:         bomID,
			ItemNumber:    i + 1,
			Category:      input.Category,
			SubCategory:   input.SubCategory,
			Name:          input.Name,
			Quantity:      input.Quantity,
			Unit:          input.Unit,
			Supplier:      input.Supplier,
			UnitPrice:     input.UnitPrice,
			IsAlternative: input.IsAlternative,
			Notes:         input.Notes,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		if input.ExtendedAttrs != nil {
			item.ExtendedAttrs = entity.JSONB(input.ExtendedAttrs)
		}
		// Copy API compat fields to extended_attrs
		if input.Specification != "" {
			setExtAttr(&item.ExtendedAttrs, "specification", input.Specification)
		}
		if input.Reference != "" {
			setExtAttr(&item.ExtendedAttrs, "reference", input.Reference)
		}
		if input.Manufacturer != "" {
			setExtAttr(&item.ExtendedAttrs, "manufacturer", input.Manufacturer)
		}
		if input.ManufacturerPN != "" {
			setExtAttr(&item.ExtendedAttrs, "manufacturer_pn", input.ManufacturerPN)
		}
		if input.SupplierPN != "" {
			setExtAttr(&item.ExtendedAttrs, "supplier_pn", input.SupplierPN)
		}
		if input.DrawingNo != "" {
			setExtAttr(&item.ExtendedAttrs, "drawing_no", input.DrawingNo)
		}
		if input.IsCritical {
			setExtAttr(&item.ExtendedAttrs, "is_critical", true)
		}
		if input.LeadTimeDays != nil {
			setExtAttr(&item.ExtendedAttrs, "lead_time_days", *input.LeadTimeDays)
		}
		if input.IsAppearancePart {
			setExtAttr(&item.ExtendedAttrs, "is_appearance_part", true)
		}
		if item.Unit == "" {
			item.Unit = "pcs"
		}
		if input.MaterialID != nil {
			item.MaterialID = input.MaterialID
		} else if item.Name != "" {
			category := item.Category
			if category == "" {
				category = defaultCategoryForBOMType(bom.BOMType)
			}
			newMat, createErr := s.autoCreateMaterial(ctx, item.Name, input.Specification, category, input.Manufacturer, input.ManufacturerPN)
			if createErr == nil && newMat != nil {
				item.MaterialID = &newMat.ID
			}
		}
		entities = append(entities, item)
	}

	if err := s.bomRepo.BatchCreateItems(ctx, entities); err != nil {
		return 0, fmt.Errorf("batch create items: %w", err)
	}

	count, _ := s.bomRepo.CountItems(ctx, bomID)
	s.bomRepo.DB().Model(&entity.ProjectBOM{}).Where("id = ?", bomID).Update("total_items", count)

	return len(entities), nil
}

// DeleteBOM 删除整个BOM及其所有行项
func (s *ProjectBOMService) DeleteBOM(ctx context.Context, bomID string) error {
	if err := s.bomRepo.DeleteItemsByBOM(ctx, bomID); err != nil {
		return fmt.Errorf("delete bom items: %w", err)
	}
	if err := s.bomRepo.Delete(ctx, bomID); err != nil {
		return fmt.Errorf("delete bom: %w", err)
	}
	return nil
}

// SearchItems 跨项目搜索BOM行项
func (s *ProjectBOMService) SearchItems(ctx context.Context, keyword, category string, limit int) ([]entity.ProjectBOMItem, error) {
	return s.bomRepo.SearchItems(ctx, keyword, category, limit)
}

// SearchItemsPaginated 跨项目搜索BOM行项（分页版）
func (s *ProjectBOMService) SearchItemsPaginated(ctx context.Context, keyword, category, subCategory, bomID string, page, pageSize int) ([]repository.MaterialSearchResult, int64, error) {
	return s.bomRepo.SearchItemsPaginated(ctx, keyword, category, subCategory, bomID, page, pageSize)
}

// GlobalSearchItems 全局物料搜索（支持project/supplier/manufacturer筛选）
func (s *ProjectBOMService) GlobalSearchItems(ctx context.Context, keyword, category, subCategory, bomID, projectID, supplierID, manufacturerID string, page, pageSize int) ([]repository.MaterialSearchResult, int64, error) {
	return s.bomRepo.GlobalSearchItems(ctx, repository.GlobalSearchParams{
		Keyword:        keyword,
		Category:       category,
		SubCategory:    subCategory,
		BOMID:          bomID,
		ProjectID:      projectID,
		SupplierID:     supplierID,
		ManufacturerID: manufacturerID,
		Page:           page,
		PageSize:       pageSize,
	})
}

// DeleteItem 删除BOM行项
func (s *ProjectBOMService) DeleteItem(ctx context.Context, bomID, itemID string) error {
	bom, err := s.bomRepo.FindByID(ctx, bomID)
	if err != nil {
		return fmt.Errorf("bom not found: %w", err)
	}
	if bom.Status != "draft" && bom.Status != "rejected" {
		return fmt.Errorf("只有草稿状态的BOM才能删除物料")
	}

	if err := s.bomRepo.DeleteItem(ctx, itemID); err != nil {
		return fmt.Errorf("delete item: %w", err)
	}

	s.updateBOMCost(ctx, bomID)
	return nil
}

// UpdateItem 更新单个BOM行项（部分更新：只更新 presentFields 中存在的字段）
func (s *ProjectBOMService) UpdateItem(ctx context.Context, bomID, itemID string, input *BOMItemInput, presentFields map[string]bool) (*entity.ProjectBOMItem, error) {
	bom, err := s.bomRepo.FindByID(ctx, bomID)
	if err != nil {
		return nil, fmt.Errorf("bom not found: %w", err)
	}
	if bom.Status != "draft" && bom.Status != "rejected" {
		return nil, fmt.Errorf("只有草稿状态的BOM才能编辑物料")
	}

	item, err := s.bomRepo.FindItemByID(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("item not found: %w", err)
	}
	if item.BOMID != bomID {
		return nil, fmt.Errorf("item does not belong to this BOM")
	}

	// Helper: only update if the field was present in the request JSON
	has := func(key string) bool { return presentFields[key] }

	if has("name") && input.Name != "" {
		item.Name = input.Name
	}
	if has("category") && input.Category != "" {
		item.Category = input.Category
	}
	if has("sub_category") && input.SubCategory != "" {
		item.SubCategory = input.SubCategory
	}
	if has("material_id") {
		item.MaterialID = input.MaterialID
	}
	if has("quantity") {
		item.Quantity = input.Quantity
	}
	if has("unit") && input.Unit != "" {
		item.Unit = input.Unit
	}
	if has("supplier") {
		item.Supplier = input.Supplier
	}
	if has("supplier_id") {
		item.SupplierID = input.SupplierID
	}
	if has("manufacturer_id") {
		item.ManufacturerID = input.ManufacturerID
	}
	if has("mpn") {
		item.MPN = input.MPN
	}
	if has("unit_price") {
		item.UnitPrice = input.UnitPrice
	}
	if has("is_alternative") {
		item.IsAlternative = input.IsAlternative
	}
	if has("notes") {
		item.Notes = input.Notes
	}
	if has("thumbnail_url") {
		item.ThumbnailURL = input.ThumbnailURL
	}
	if has("parent_item_id") {
		item.ParentItemID = input.ParentItemID
	}
	if has("level") {
		item.Level = input.Level
	}

	// Merge extended_attrs (don't overwrite, merge keys)
	if has("extended_attrs") && input.ExtendedAttrs != nil {
		if item.ExtendedAttrs == nil {
			item.ExtendedAttrs = entity.JSONB{}
		}
		for k, v := range input.ExtendedAttrs {
			item.ExtendedAttrs[k] = v
		}
	}
	// Copy API compat fields to extended_attrs (only if sent)
	if has("specification") {
		setExtAttr(&item.ExtendedAttrs, "specification", input.Specification)
	}
	if has("reference") {
		setExtAttr(&item.ExtendedAttrs, "reference", input.Reference)
	}
	if has("manufacturer") {
		setExtAttr(&item.ExtendedAttrs, "manufacturer", input.Manufacturer)
	}
	if has("manufacturer_pn") {
		setExtAttr(&item.ExtendedAttrs, "manufacturer_pn", input.ManufacturerPN)
	}
	if has("supplier_pn") {
		setExtAttr(&item.ExtendedAttrs, "supplier_pn", input.SupplierPN)
	}
	if has("drawing_no") {
		setExtAttr(&item.ExtendedAttrs, "drawing_no", input.DrawingNo)
	}
	if has("is_critical") {
		setExtAttr(&item.ExtendedAttrs, "is_critical", input.IsCritical)
	}
	if has("lead_time_days") && input.LeadTimeDays != nil {
		setExtAttr(&item.ExtendedAttrs, "lead_time_days", *input.LeadTimeDays)
	}
	if has("is_appearance_part") {
		setExtAttr(&item.ExtendedAttrs, "is_appearance_part", input.IsAppearancePart)
	}

	// Recalculate extended cost if price or quantity changed
	if has("unit_price") || has("quantity") {
		if item.UnitPrice != nil {
			extCost := item.Quantity * *item.UnitPrice
			item.ExtendedCost = &extCost
		}
	}

	item.UpdatedAt = time.Now()

	if err := s.bomRepo.UpdateItem(ctx, item); err != nil {
		return nil, fmt.Errorf("update item: %w", err)
	}

	s.updateBOMCost(ctx, bomID)
	return item, nil
}

// ReorderItems 拖拽排序
func (s *ProjectBOMService) ReorderItems(ctx context.Context, bomID string, itemIDs []string) error {
	bom, err := s.bomRepo.FindByID(ctx, bomID)
	if err != nil {
		return fmt.Errorf("bom not found: %w", err)
	}
	if bom.Status != "draft" && bom.Status != "rejected" {
		return fmt.Errorf("只有草稿状态的BOM才能排序")
	}

	for i, id := range itemIDs {
		s.bomRepo.DB().Model(&entity.ProjectBOMItem{}).Where("id = ? AND bom_id = ?", id, bomID).Update("item_number", i+1)
	}
	return nil
}

// updateBOMCost 更新BOM总成本统计
func (s *ProjectBOMService) updateBOMCost(ctx context.Context, bomID string) {
	var totalCost float64
	s.bomRepo.DB().Model(&entity.ProjectBOMItem{}).
		Where("bom_id = ?", bomID).
		Select("COALESCE(SUM(extended_cost), 0)").
		Scan(&totalCost)
	count, _ := s.bomRepo.CountItems(ctx, bomID)
	s.bomRepo.DB().Model(&entity.ProjectBOM{}).Where("id = ?", bomID).
		Updates(map[string]interface{}{"estimated_cost": totalCost, "total_items": count})
}

// ==================== Excel 导入/导出 ====================

var bomExportHeaders = []string{
	"序号", "分类", "小类", "名称", "规格", "数量", "单位", "位号",
	"制造商", "制造商料号", "供应商", "供应商料号", "单价", "小计",
	"是否关键", "备注",
}

var pbomExportHeaders = []string{
	"序号", "层级", "零件名称", "物料编码", "规格", "数量", "单位",
	"材质", "颜色", "表面处理", "工艺类型", "图纸编号", "重量(g)",
	"目标单价", "模具费预估", "是否外观件", "装配方式", "公差等级",
	"供应商", "备注",
}

// ExportBOM 导出BOM为xlsx
func (s *ProjectBOMService) ExportBOM(ctx context.Context, bomID string) (*excelize.File, string, error) {
	bom, err := s.bomRepo.FindByID(ctx, bomID)
	if err != nil {
		return nil, "", fmt.Errorf("bom not found: %w", err)
	}

	items, err := s.bomRepo.ListItemsByBOM(ctx, bomID)
	if err != nil {
		return nil, "", fmt.Errorf("list items: %w", err)
	}

	if bom.BOMType == "PBOM" || bom.BOMType == "SBOM" {
		return s.exportStructuralBOM(bom, items)
	}
	return s.exportElectronicBOM(bom, items)
}

func (s *ProjectBOMService) exportElectronicBOM(bom *entity.ProjectBOM, items []entity.ProjectBOMItem) (*excelize.File, string, error) {
	f := excelize.NewFile()
	sheet := "BOM"
	f.SetSheetName("Sheet1", sheet)

	boldStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#D9E1F2"}},
		Border: []excelize.Border{{Type: "bottom", Color: "000000", Style: 1}},
	})

	for i, h := range bomExportHeaders {
		col, _ := excelize.ColumnNumberToName(i + 1)
		cell := col + "1"
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, boldStyle)
	}

	var totalCost float64
	for rowIdx, item := range items {
		row := rowIdx + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), item.ItemNumber)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), item.Category)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), item.SubCategory)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), item.Name)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), getExtAttr(item.ExtendedAttrs, "specification"))
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), item.Quantity)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), item.Unit)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), getExtAttr(item.ExtendedAttrs, "reference"))
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), getExtAttr(item.ExtendedAttrs, "manufacturer"))
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), getExtAttr(item.ExtendedAttrs, "manufacturer_pn"))
		f.SetCellValue(sheet, fmt.Sprintf("K%d", row), item.Supplier)
		f.SetCellValue(sheet, fmt.Sprintf("L%d", row), getExtAttr(item.ExtendedAttrs, "supplier_pn"))
		if item.UnitPrice != nil {
			f.SetCellValue(sheet, fmt.Sprintf("M%d", row), *item.UnitPrice)
		}
		if item.ExtendedCost != nil {
			f.SetCellValue(sheet, fmt.Sprintf("N%d", row), *item.ExtendedCost)
			totalCost += *item.ExtendedCost
		}
		critical := "否"
		if getExtAttrBool(item.ExtendedAttrs, "is_critical") {
			critical = "是"
		}
		f.SetCellValue(sheet, fmt.Sprintf("O%d", row), critical)
		f.SetCellValue(sheet, fmt.Sprintf("P%d", row), item.Notes)
	}

	summaryRow := len(items) + 2
	summaryStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	f.SetCellValue(sheet, fmt.Sprintf("A%d", summaryRow), "汇总")
	f.SetCellValue(sheet, fmt.Sprintf("D%d", summaryRow), fmt.Sprintf("总物料数: %d", len(items)))
	f.SetCellValue(sheet, fmt.Sprintf("N%d", summaryRow), totalCost)
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", summaryRow), fmt.Sprintf("P%d", summaryRow), summaryStyle)

	colWidths := []float64{6, 10, 10, 20, 20, 8, 6, 14, 16, 16, 16, 16, 10, 10, 8, 20}
	for i, w := range colWidths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, w)
	}

	filename := fmt.Sprintf("BOM_%s_%s.xlsx", bom.Name, bom.Version)
	return f, filename, nil
}

func (s *ProjectBOMService) exportStructuralBOM(bom *entity.ProjectBOM, items []entity.ProjectBOMItem) (*excelize.File, string, error) {
	f := excelize.NewFile()
	sheet := "PBOM"
	f.SetSheetName("Sheet1", sheet)

	boldStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#E2EFDA"}},
		Border: []excelize.Border{{Type: "bottom", Color: "000000", Style: 1}},
	})

	for i, h := range pbomExportHeaders {
		col, _ := excelize.ColumnNumberToName(i + 1)
		cell := col + "1"
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, boldStyle)
	}

	var totalTargetPrice float64
	var totalTooling float64
	for rowIdx, item := range items {
		row := rowIdx + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), item.ItemNumber)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), item.Level)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), item.Name)
		materialCode := getExtAttr(item.ExtendedAttrs, "manufacturer_pn")
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), materialCode)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), getExtAttr(item.ExtendedAttrs, "specification"))
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), item.Quantity)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), item.Unit)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), getExtAttr(item.ExtendedAttrs, "material_type"))
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), getExtAttr(item.ExtendedAttrs, "color"))
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), getExtAttr(item.ExtendedAttrs, "surface_treatment"))
		f.SetCellValue(sheet, fmt.Sprintf("K%d", row), getExtAttr(item.ExtendedAttrs, "process_type"))
		f.SetCellValue(sheet, fmt.Sprintf("L%d", row), getExtAttr(item.ExtendedAttrs, "drawing_no"))
		if wStr := getExtAttr(item.ExtendedAttrs, "weight_grams"); wStr != "" {
			if w, err := strconv.ParseFloat(wStr, 64); err == nil {
				f.SetCellValue(sheet, fmt.Sprintf("M%d", row), w)
			}
		}
		if tpStr := getExtAttr(item.ExtendedAttrs, "target_price"); tpStr != "" {
			if tp, err := strconv.ParseFloat(tpStr, 64); err == nil {
				f.SetCellValue(sheet, fmt.Sprintf("N%d", row), tp)
				totalTargetPrice += tp * item.Quantity
			}
		}
		if teStr := getExtAttr(item.ExtendedAttrs, "tooling_estimate"); teStr != "" {
			if te, err := strconv.ParseFloat(teStr, 64); err == nil {
				f.SetCellValue(sheet, fmt.Sprintf("O%d", row), te)
				totalTooling += te
			}
		}
		appearance := "N"
		if getExtAttrBool(item.ExtendedAttrs, "is_appearance_part") {
			appearance = "Y"
		}
		f.SetCellValue(sheet, fmt.Sprintf("P%d", row), appearance)
		f.SetCellValue(sheet, fmt.Sprintf("Q%d", row), getExtAttr(item.ExtendedAttrs, "assembly_method"))
		f.SetCellValue(sheet, fmt.Sprintf("R%d", row), getExtAttr(item.ExtendedAttrs, "tolerance_grade"))
		f.SetCellValue(sheet, fmt.Sprintf("S%d", row), item.Supplier)
		f.SetCellValue(sheet, fmt.Sprintf("T%d", row), item.Notes)
	}

	summaryRow := len(items) + 2
	summaryStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	f.SetCellValue(sheet, fmt.Sprintf("A%d", summaryRow), "汇总")
	f.SetCellValue(sheet, fmt.Sprintf("C%d", summaryRow), fmt.Sprintf("总零件数: %d", len(items)))
	f.SetCellValue(sheet, fmt.Sprintf("N%d", summaryRow), totalTargetPrice)
	f.SetCellValue(sheet, fmt.Sprintf("O%d", summaryRow), totalTooling)
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", summaryRow), fmt.Sprintf("T%d", summaryRow), summaryStyle)

	colWidths := []float64{6, 6, 20, 16, 20, 8, 6, 14, 12, 14, 12, 14, 10, 10, 12, 10, 12, 10, 16, 20}
	for i, w := range colWidths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, w)
	}

	filename := fmt.Sprintf("PBOM_%s_%s.xlsx", bom.Name, bom.Version)
	return f, filename, nil
}

// ImportBOM 从Excel导入BOM行项
func (s *ProjectBOMService) ImportBOM(ctx context.Context, bomID string, f *excelize.File) (*ImportResult, error) {
	bom, err := s.bomRepo.FindByID(ctx, bomID)
	if err != nil {
		return nil, fmt.Errorf("bom not found: %w", err)
	}
	if bom.Status != "draft" && bom.Status != "rejected" {
		return nil, fmt.Errorf("只有草稿或被驳回的BOM才能导入")
	}

	sheet := f.GetSheetName(0)
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("read excel: %w", err)
	}

	result := &ImportResult{}
	if len(rows) < 2 {
		return result, nil
	}

	existingCount, _ := s.bomRepo.CountItems(ctx, bomID)
	itemNum := int(existingCount)

	var entities []entity.ProjectBOMItem
	for i, row := range rows[1:] {
		if len(row) < 3 || row[2] == "" {
			result.Failed++
			continue
		}

		itemNum++
		item := entity.ProjectBOMItem{
			ID:         uuid.New().String()[:32],
			BOMID:      bomID,
			ItemNumber: itemNum,
			Unit:       "pcs",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		if len(row) > 1 {
			item.Category = row[1]
		}
		if len(row) > 2 {
			item.Name = row[2]
		}
		extAttrs := map[string]interface{}{}

		if len(row) > 3 && row[3] != "" {
			extAttrs["specification"] = row[3]
		}
		if len(row) > 4 {
			if q, err := strconv.ParseFloat(row[4], 64); err == nil {
				item.Quantity = q
			} else {
				item.Quantity = 1
			}
		}
		if len(row) > 5 && row[5] != "" {
			item.Unit = row[5]
		}
		if len(row) > 6 && row[6] != "" {
			extAttrs["reference"] = row[6]
		}
		if len(row) > 7 && row[7] != "" {
			extAttrs["manufacturer"] = row[7]
		}
		manufacturerPN := ""
		if len(row) > 8 && row[8] != "" {
			manufacturerPN = row[8]
			extAttrs["manufacturer_pn"] = row[8]
		}
		if len(row) > 9 {
			item.Supplier = row[9]
		}
		if len(row) > 10 && row[10] != "" {
			extAttrs["supplier_pn"] = row[10]
		}
		if len(row) > 11 {
			if p, err := strconv.ParseFloat(row[11], 64); err == nil {
				item.UnitPrice = &p
				extCost := item.Quantity * p
				item.ExtendedCost = &extCost
			}
		}
		if len(row) > 13 && (row[13] == "是" || row[13] == "Y" || row[13] == "1") {
			extAttrs["is_critical"] = true
		}
		if len(row) > 14 {
			item.Notes = row[14]
		}

		if len(extAttrs) > 0 {
			item.ExtendedAttrs = entity.JSONB(extAttrs)
		}

		specification := getExtAttr(item.ExtendedAttrs, "specification")
		manufacturer := getExtAttr(item.ExtendedAttrs, "manufacturer")

		mat, matchErr := s.bomRepo.MatchMaterialByNameAndPN(ctx, item.Name, manufacturerPN)
		if matchErr == nil && mat != nil {
			item.MaterialID = &mat.ID
			result.Matched++
		} else {
			newMat, createErr := s.autoCreateMaterial(ctx, item.Name, specification, item.Category, manufacturer, manufacturerPN)
			if createErr != nil {
				fmt.Printf("[WARN] auto-create material failed for %q: %v\n", item.Name, createErr)
			} else if newMat != nil {
				item.MaterialID = &newMat.ID
				result.AutoCreated++
			}
		}

		entities = append(entities, item)
		result.Success++
		_ = i
	}

	if len(entities) > 0 {
		if err := s.bomRepo.BatchCreateItems(ctx, entities); err != nil {
			return nil, fmt.Errorf("batch create: %w", err)
		}
		s.updateBOMCost(ctx, bomID)
	}

	return result, nil
}

// ImportStructuralBOM 从Excel导入结构BOM行项
func (s *ProjectBOMService) ImportStructuralBOM(ctx context.Context, bomID string, f *excelize.File) (*ImportResult, error) {
	bom, err := s.bomRepo.FindByID(ctx, bomID)
	if err != nil {
		return nil, fmt.Errorf("bom not found: %w", err)
	}
	if bom.Status != "draft" && bom.Status != "rejected" {
		return nil, fmt.Errorf("只有草稿或被驳回的BOM才能导入")
	}

	sheet := f.GetSheetName(0)
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("read excel: %w", err)
	}

	result := &ImportResult{}
	if len(rows) < 2 {
		return result, nil
	}

	existingCount, _ := s.bomRepo.CountItems(ctx, bomID)
	itemNum := int(existingCount)

	var entities []entity.ProjectBOMItem
	for _, row := range rows[1:] {
		if len(row) < 3 || row[2] == "" {
			result.Failed++
			continue
		}

		itemNum++
		item := entity.ProjectBOMItem{
			ID:          uuid.New().String()[:32],
			BOMID:       bomID,
			ItemNumber:  itemNum,
			Category:    "structural",
			SubCategory: "structural_part",
			Unit:        "pcs",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		extAttrs := map[string]interface{}{}

		if len(row) > 1 {
			if lvl, err := strconv.Atoi(strings.TrimSpace(row[1])); err == nil {
				item.Level = lvl
			}
		}
		if len(row) > 2 {
			item.Name = strings.TrimSpace(row[2])
		}
		if len(row) > 3 && strings.TrimSpace(row[3]) != "" {
			extAttrs["manufacturer_pn"] = strings.TrimSpace(row[3])
		}
		if len(row) > 4 && strings.TrimSpace(row[4]) != "" {
			extAttrs["specification"] = strings.TrimSpace(row[4])
		}
		if len(row) > 5 {
			if q, err := strconv.ParseFloat(strings.TrimSpace(row[5]), 64); err == nil {
				item.Quantity = q
			} else {
				item.Quantity = 1
			}
		}
		if len(row) > 6 && strings.TrimSpace(row[6]) != "" {
			item.Unit = strings.TrimSpace(row[6])
		}
		if len(row) > 7 && strings.TrimSpace(row[7]) != "" {
			extAttrs["material_type"] = strings.TrimSpace(row[7])
		}
		if len(row) > 8 && strings.TrimSpace(row[8]) != "" {
			extAttrs["color"] = strings.TrimSpace(row[8])
		}
		if len(row) > 9 && strings.TrimSpace(row[9]) != "" {
			extAttrs["surface_treatment"] = strings.TrimSpace(row[9])
		}
		if len(row) > 10 && strings.TrimSpace(row[10]) != "" {
			extAttrs["process_type"] = strings.TrimSpace(row[10])
		}
		if len(row) > 11 && strings.TrimSpace(row[11]) != "" {
			extAttrs["drawing_no"] = strings.TrimSpace(row[11])
		}
		if len(row) > 12 {
			if w, err := strconv.ParseFloat(strings.TrimSpace(row[12]), 64); err == nil {
				extAttrs["weight_grams"] = w
			}
		}
		if len(row) > 13 {
			if p, err := strconv.ParseFloat(strings.TrimSpace(row[13]), 64); err == nil {
				extAttrs["target_price"] = p
			}
		}
		if len(row) > 14 {
			if t, err := strconv.ParseFloat(strings.TrimSpace(row[14]), 64); err == nil {
				extAttrs["tooling_estimate"] = t
			}
		}
		if len(row) > 15 {
			val := strings.TrimSpace(row[15])
			if val == "Y" || val == "y" || val == "是" || val == "1" {
				extAttrs["is_appearance_part"] = true
			}
		}
		if len(row) > 16 && strings.TrimSpace(row[16]) != "" {
			extAttrs["assembly_method"] = strings.TrimSpace(row[16])
		}
		if len(row) > 17 && strings.TrimSpace(row[17]) != "" {
			extAttrs["tolerance_grade"] = strings.TrimSpace(row[17])
		}
		if len(row) > 18 {
			item.Supplier = strings.TrimSpace(row[18])
		}
		if len(row) > 19 {
			item.Notes = strings.TrimSpace(row[19])
		}

		if len(extAttrs) > 0 {
			item.ExtendedAttrs = entity.JSONB(extAttrs)
		}

		manufacturerPN := getExtAttr(item.ExtendedAttrs, "manufacturer_pn")
		specification := getExtAttr(item.ExtendedAttrs, "specification")

		mat, matchErr := s.bomRepo.MatchMaterialByNameAndPN(ctx, item.Name, manufacturerPN)
		if matchErr == nil && mat != nil {
			item.MaterialID = &mat.ID
			result.Matched++
		} else {
			newMat, createErr := s.autoCreateMaterial(ctx, item.Name, specification, "结构件", item.Supplier, manufacturerPN)
			if createErr != nil {
				fmt.Printf("[WARN] auto-create material failed for %q: %v\n", item.Name, createErr)
			} else if newMat != nil {
				item.MaterialID = &newMat.ID
				result.AutoCreated++
			}
		}

		entities = append(entities, item)
		result.Success++
	}

	if len(entities) > 0 {
		if err := s.bomRepo.BatchCreateItems(ctx, entities); err != nil {
			return nil, fmt.Errorf("batch create: %w", err)
		}
		s.updateBOMCost(ctx, bomID)
	}

	return result, nil
}

// ImportPADSBOM 从PADS BOM (.rep文件) 导入BOM行项
func (s *ProjectBOMService) ImportPADSBOM(ctx context.Context, bomID string, reader io.Reader) (*ImportResult, error) {
	bom, err := s.bomRepo.FindByID(ctx, bomID)
	if err != nil {
		return nil, fmt.Errorf("bom not found: %w", err)
	}
	if bom.Status != "draft" && bom.Status != "rejected" {
		return nil, fmt.Errorf("只有草稿或被驳回的BOM才能导入")
	}

	utf8Reader := transform.NewReader(reader, simplifiedchinese.GBK.NewDecoder())
	result := &ImportResult{}

	existingCount, _ := s.bomRepo.CountItems(ctx, bomID)
	itemNum := int(existingCount)

	// Phase 1: Parse all lines first to collect MPNs
	type parsedLine struct {
		qty            float64
		reference      string
		componentName  string
		name           string
		manufacturer   string
		manufacturerPN string
		notes          string
		categoryID     string
	}
	var parsed []parsedLine

	scanner := bufio.NewScanner(utf8Reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimRight(scanner.Text(), "\r\n")
		if line == "" {
			continue
		}
		if lineNo == 1 {
			continue
		}

		fields := strings.Split(line, "\t")
		for i := range fields {
			fields[i] = strings.Trim(fields[i], "\"")
		}

		if len(fields) < 4 || fields[3] == "" {
			result.Failed++
			continue
		}

		if len(fields) > 7 && strings.EqualFold(strings.TrimSpace(fields[7]), "NC") {
			continue
		}

		qty := 1.0
		if q, parseErr := strconv.ParseFloat(fields[1], 64); parseErr == nil {
			qty = q
		}

		reference := ""
		if len(fields) > 2 {
			reference = fields[2]
		}

		componentName := fields[3]
		name := componentName
		if idx := strings.Index(componentName, ","); idx > 0 {
			name = componentName[:idx]
		}

		manufacturer := ""
		if len(fields) > 4 {
			manufacturer = fields[4]
		}

		notes := ""
		if len(fields) > 5 {
			notes = fields[5]
		}
		if len(fields) > 7 && fields[7] != "" {
			if notes != "" {
				notes += "; "
			}
			notes += fields[7]
		}

		manufacturerPN := ""
		if len(fields) > 6 {
			manufacturerPN = fields[6]
		}

		_, categoryID := inferCategoryFromReference(reference)

		parsed = append(parsed, parsedLine{
			qty: qty, reference: reference, componentName: componentName,
			name: name, manufacturer: manufacturer, manufacturerPN: manufacturerPN,
			notes: notes, categoryID: categoryID,
		})
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return nil, fmt.Errorf("read rep file: %w", scanErr)
	}

	// Phase 2: Collect all MPNs and batch-check existing BOM items
	var mpns []string
	for _, p := range parsed {
		if p.manufacturerPN != "" {
			mpns = append(mpns, p.manufacturerPN)
		}
	}
	existingByMPN, _ := s.bomRepo.FindItemsByMPN(ctx, bomID, mpns)

	// Phase 3: Create items, skipping MPN duplicates
	var entities []entity.ProjectBOMItem
	for _, p := range parsed {
		detail := ImportItemDetail{
			Name:      p.name,
			MPN:       p.manufacturerPN,
			Reference: p.reference,
		}

		// Check MPN match status
		if p.manufacturerPN == "" {
			detail.Status = "missing"
			result.MPNMissing++
		} else if existing, ok := existingByMPN[p.manufacturerPN]; ok {
			// MPN already exists in this BOM — skip, don't duplicate
			detail.Status = "matched"
			detail.MatchedItemID = existing.ID
			result.MPNMatched++
			result.Items = append(result.Items, detail)
			continue // don't create a new item
		} else {
			detail.Status = "new"
			result.MPNNew++
		}
		result.Items = append(result.Items, detail)

		itemNum++

		padsExtAttrs := entity.JSONB{}
		if p.componentName != "" {
			padsExtAttrs["specification"] = p.componentName
		}
		if p.reference != "" {
			padsExtAttrs["reference"] = p.reference
		}
		if p.manufacturer != "" {
			padsExtAttrs["manufacturer"] = p.manufacturer
		}
		if p.manufacturerPN != "" {
			padsExtAttrs["manufacturer_pn"] = p.manufacturerPN
		}

		item := entity.ProjectBOMItem{
			ID:            uuid.New().String()[:32],
			BOMID:         bomID,
			ItemNumber:    itemNum,
			Category:      "electronic",
			SubCategory:   "component",
			Name:          p.name,
			Quantity:      p.qty,
			Unit:          "pcs",
			MPN:           p.manufacturerPN,
			Notes:         p.notes,
			ExtendedAttrs: padsExtAttrs,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		mat, matchErr := s.bomRepo.MatchMaterialByNameAndPN(ctx, item.Name, p.manufacturerPN)
		if matchErr == nil && mat != nil {
			item.MaterialID = &mat.ID
			result.Matched++
		} else {
			cID := p.categoryID
			if cID == "" {
				cID = "mcat_el_oth"
			}
			newMat, createErr := s.autoCreateMaterial(ctx, item.Name, p.componentName, cID, p.manufacturer, p.manufacturerPN)
			if createErr != nil {
				fmt.Printf("[WARN] auto-create material failed for %q: %v\n", item.Name, createErr)
			} else if newMat != nil {
				item.MaterialID = &newMat.ID
				result.AutoCreated++
			}
		}

		entities = append(entities, item)
		result.Success++
	}

	if len(entities) > 0 {
		if err := s.bomRepo.BatchCreateItems(ctx, entities); err != nil {
			return nil, fmt.Errorf("batch create: %w", err)
		}
		s.updateBOMCost(ctx, bomID)
	}

	return result, nil
}

func inferCategoryFromReference(reference string) (string, string) {
	if reference == "" {
		return "", "mcat_el_oth"
	}
	first := strings.Fields(reference)[0]
	prefix := strings.ToUpper(strings.TrimRight(first, "0123456789-"))

	switch prefix {
	case "R":
		return "电阻", "mcat_el_res"
	case "C":
		return "电容", "mcat_el_cap"
	case "L":
		return "电感", "mcat_el_ind"
	case "U":
		return "IC", "mcat_el_ic"
	case "J", "CN":
		return "连接器", "mcat_el_con"
	case "D":
		return "二极管", "mcat_el_dio"
	case "Q":
		return "晶体管", "mcat_el_trn"
	case "Y":
		return "晶振", "mcat_el_osc"
	case "TP":
		return "测试点", "mcat_el_oth"
	default:
		return "", "mcat_el_oth"
	}
}

// GenerateTemplate 生成BOM导入模板xlsx
func (s *ProjectBOMService) GenerateTemplate(bomType string) (*excelize.File, error) {
	if bomType == "PBOM" || bomType == "SBOM" {
		return s.generateStructuralTemplate()
	}
	return s.generateElectronicTemplate()
}

func (s *ProjectBOMService) generateElectronicTemplate() (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "BOM模板"
	f.SetSheetName("Sheet1", sheet)

	boldStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#D9E1F2"}},
	})

	for i, h := range bomExportHeaders {
		col, _ := excelize.ColumnNumberToName(i + 1)
		cell := col + "1"
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, boldStyle)
	}

	colWidths := []float64{6, 10, 10, 20, 20, 8, 6, 14, 16, 16, 16, 16, 10, 10, 8, 20}
	for i, w := range colWidths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, w)
	}

	helpSheet := "填写说明"
	f.NewSheet(helpSheet)
	helpData := [][]string{
		{"列名", "说明", "是否必填"},
		{"序号", "自动编号，可留空", "否"},
		{"分类", "物料大类，如: electronic/structural/packaging", "否"},
		{"小类", "物料小类，如: component/pcb/connector", "否"},
		{"名称", "物料名称", "是"},
		{"规格", "规格型号描述", "否"},
		{"数量", "用量数字，默认1", "是"},
		{"单位", "pcs/kg/m/set，默认pcs", "否"},
		{"位号", "PCB位号，如R1,R2,R3", "否"},
		{"制造商", "制造商名称", "否"},
		{"制造商料号", "制造商Part Number", "否"},
		{"供应商", "供应商名称", "否"},
		{"供应商料号", "供应商Part Number", "否"},
		{"单价", "物料单价(元)", "否"},
		{"小计", "自动计算=数量×单价，无需填写", "否"},
		{"是否关键", "填写 是/否", "否"},
		{"备注", "备注信息", "否"},
	}
	for i, row := range helpData {
		for j, val := range row {
			col, _ := excelize.ColumnNumberToName(j + 1)
			f.SetCellValue(helpSheet, fmt.Sprintf("%s%d", col, i+1), val)
		}
	}
	f.SetColWidth(helpSheet, "A", "A", 14)
	f.SetColWidth(helpSheet, "B", "B", 40)
	f.SetColWidth(helpSheet, "C", "C", 10)

	sampleData := []string{"1", "electronic", "component", "100K电阻 0402", "100KΩ ±1% 0402", "10", "pcs", "R1-R10", "Yageo", "RC0402FR-07100KL", "DigiKey", "311-100KLRCT-ND", "0.05", "", "否", ""}
	for j, val := range sampleData {
		col, _ := excelize.ColumnNumberToName(j + 1)
		f.SetCellValue(sheet, fmt.Sprintf("%s2", col), val)
	}

	return f, nil
}

func (s *ProjectBOMService) generateStructuralTemplate() (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "结构BOM模板"
	f.SetSheetName("Sheet1", sheet)

	boldStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#E2EFDA"}},
	})

	for i, h := range pbomExportHeaders {
		col, _ := excelize.ColumnNumberToName(i + 1)
		cell := col + "1"
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, boldStyle)
	}

	colWidths := []float64{6, 6, 20, 16, 20, 8, 6, 14, 12, 14, 12, 14, 10, 10, 12, 10, 12, 10, 16, 20}
	for i, w := range colWidths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, w)
	}

	helpSheet := "填写说明"
	f.NewSheet(helpSheet)
	helpData := [][]string{
		{"列名", "说明", "是否必填"},
		{"序号", "自动编号，可留空", "否"},
		{"层级", "BOM层级，0=顶层，1=一级子件", "否"},
		{"零件名称", "零件/组件名称", "是"},
		{"物料编码", "物料编码或制造商料号", "否"},
		{"规格", "规格型号描述", "否"},
		{"数量", "用量数字，默认1", "是"},
		{"单位", "pcs/kg/m/set，默认pcs", "否"},
		{"材质", "如: PC, ABS, PA66+GF30", "否"},
		{"颜色", "如: 磨砂黑, 透明", "否"},
		{"表面处理", "如: 阳极氧化, 喷涂, 电镀", "否"},
		{"工艺类型", "如: 注塑, CNC, 冲压", "否"},
		{"图纸编号", "工程图纸编号", "否"},
		{"重量(g)", "单个零件重量(克)", "否"},
		{"目标单价", "目标单价(元)", "否"},
		{"模具费预估", "模具费用预估(元)", "否"},
		{"是否外观件", "填写 Y/N", "否"},
		{"装配方式", "如: 卡扣, 螺丝, 胶合", "否"},
		{"公差等级", "如: 普通, 精密, 超精密", "否"},
		{"供应商", "供应商名称", "否"},
		{"备注", "备注信息", "否"},
	}
	for i, row := range helpData {
		for j, val := range row {
			col, _ := excelize.ColumnNumberToName(j + 1)
			f.SetCellValue(helpSheet, fmt.Sprintf("%s%d", col, i+1), val)
		}
	}
	f.SetColWidth(helpSheet, "A", "A", 14)
	f.SetColWidth(helpSheet, "B", "B", 50)
	f.SetColWidth(helpSheet, "C", "C", 10)

	sampleData := []string{"1", "0", "前壳", "SC-001", "前壳组件", "1", "pcs", "PC+ABS", "磨砂黑", "喷涂+丝印", "注塑", "DWG-SC-001", "35.5", "2.50", "80000", "Y", "卡扣", "精密", "东莞XX模具", ""}
	for j, val := range sampleData {
		col, _ := excelize.ColumnNumberToName(j + 1)
		f.SetCellValue(sheet, fmt.Sprintf("%s2", col), val)
	}

	return f, nil
}

// ==================== Parse-only ====================

func (s *ProjectBOMService) ParsePADSBOM(ctx context.Context, reader io.Reader) ([]ParsedBOMItem, error) {
	utf8Reader := transform.NewReader(reader, simplifiedchinese.GBK.NewDecoder())
	var items []ParsedBOMItem
	scanner := bufio.NewScanner(utf8Reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	itemNum := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimRight(scanner.Text(), "\r\n")
		if line == "" || lineNo == 1 {
			continue
		}

		fields := strings.Split(line, "\t")
		for i := range fields {
			fields[i] = strings.Trim(fields[i], "\"")
		}

		if len(fields) < 4 || fields[3] == "" {
			continue
		}
		if len(fields) > 7 && strings.EqualFold(strings.TrimSpace(fields[7]), "NC") {
			continue
		}

		itemNum++
		qty := 1.0
		if q, parseErr := strconv.ParseFloat(fields[1], 64); parseErr == nil {
			qty = q
		}
		reference := ""
		if len(fields) > 2 {
			reference = fields[2]
		}
		componentName := fields[3]
		name := componentName
		if idx := strings.Index(componentName, ","); idx > 0 {
			name = componentName[:idx]
		}
		manufacturer := ""
		if len(fields) > 4 {
			manufacturer = fields[4]
		}
		manufacturerPN := ""
		if len(fields) > 6 {
			manufacturerPN = fields[6]
		}
		categoryName, _ := inferCategoryFromReference(reference)

		items = append(items, ParsedBOMItem{
			ItemNumber:     itemNum,
			Reference:      reference,
			Name:           name,
			Specification:  componentName,
			Quantity:       qty,
			Unit:           "pcs",
			Category:       categoryName,
			Manufacturer:   manufacturer,
			ManufacturerPN: manufacturerPN,
		})
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return nil, fmt.Errorf("read rep file: %w", scanErr)
	}
	return items, nil
}

func (s *ProjectBOMService) ParseExcelBOM(ctx context.Context, f *excelize.File) ([]ParsedBOMItem, error) {
	sheet := f.GetSheetName(0)
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("read excel: %w", err)
	}

	var items []ParsedBOMItem
	if len(rows) < 2 {
		return items, nil
	}

	itemNum := 0
	for _, row := range rows[1:] {
		if len(row) < 3 || row[2] == "" {
			continue
		}
		itemNum++
		item := ParsedBOMItem{ItemNumber: itemNum, Unit: "pcs"}
		if len(row) > 1 {
			item.Category = row[1]
		}
		if len(row) > 2 {
			item.Name = row[2]
		}
		if len(row) > 3 {
			item.Specification = row[3]
		}
		if len(row) > 4 {
			if q, err := strconv.ParseFloat(row[4], 64); err == nil {
				item.Quantity = q
			} else {
				item.Quantity = 1
			}
		}
		if len(row) > 5 && row[5] != "" {
			item.Unit = row[5]
		}
		if len(row) > 6 {
			item.Reference = row[6]
		}
		if len(row) > 7 {
			item.Manufacturer = row[7]
		}
		if len(row) > 8 {
			item.ManufacturerPN = row[8]
		}
		items = append(items, item)
	}
	return items, nil
}

// ==================== BOM转换 ====================

func (s *ProjectBOMService) ConvertToPBOM(ctx context.Context, bomID, createdBy string) (*entity.ProjectBOM, error) {
	srcBOM, err := s.bomRepo.FindByID(ctx, bomID)
	if err != nil {
		return nil, fmt.Errorf("source bom not found: %w", err)
	}
	if srcBOM.BOMType != "EBOM" {
		return nil, fmt.Errorf("只能从EBOM转换为PBOM")
	}

	newBOM := &entity.ProjectBOM{
		ID:          uuid.New().String()[:32],
		ProjectID:   srcBOM.ProjectID,
		PhaseID:     srcBOM.PhaseID,
		BOMType:     "PBOM",
		SourceBOMID: &srcBOM.ID,
		Version:     "v1.0",
		Name:        srcBOM.Name + " (PBOM)",
		Status:      "draft",
		Description: fmt.Sprintf("从 %s %s 转换而来", srcBOM.Name, srcBOM.Version),
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.bomRepo.Create(ctx, newBOM); err != nil {
		return nil, fmt.Errorf("create pbom: %w", err)
	}

	items, err := s.bomRepo.ListItemsByBOM(ctx, bomID)
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}

	if len(items) > 0 {
		var newItems []entity.ProjectBOMItem
		for _, item := range items {
			newItem := item
			newItem.ID = uuid.New().String()[:32]
			newItem.BOMID = newBOM.ID
			newItem.ParentItemID = nil
			newItem.CreatedAt = time.Now()
			newItem.UpdatedAt = time.Now()
			newItem.Material = nil
			newItem.ParentItem = nil
			newItem.Children = nil
			newItem.Drawings = nil
			newItem.CMFVariants = nil
			newItem.LangVariants = nil
			newItem.ProcessStep = nil
			newItems = append(newItems, newItem)
		}
		if err := s.bomRepo.BatchCreateItems(ctx, newItems); err != nil {
			return nil, fmt.Errorf("copy items: %w", err)
		}
		newBOM.TotalItems = len(newItems)
		newBOM.EstimatedCost = srcBOM.EstimatedCost
		s.bomRepo.Update(ctx, newBOM)
	}

	route := &entity.ProcessRoute{
		ID:          uuid.New().String()[:32],
		ProjectID:   srcBOM.ProjectID,
		BOMID:       newBOM.ID,
		Name:        newBOM.Name + " 工艺路线",
		Version:     "v1.0",
		Status:      "draft",
		Description: "自动创建的默认工艺路线",
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	s.bomRepo.CreateRoute(ctx, route)

	return s.bomRepo.FindByID(ctx, newBOM.ID)
}

func (s *ProjectBOMService) ConvertToMBOM(ctx context.Context, bomID, createdBy string) (*entity.ProjectBOM, error) {
	srcBOM, err := s.bomRepo.FindByID(ctx, bomID)
	if err != nil {
		return nil, fmt.Errorf("source bom not found: %w", err)
	}

	newBOM := &entity.ProjectBOM{
		ID:          uuid.New().String()[:32],
		ProjectID:   srcBOM.ProjectID,
		PhaseID:     srcBOM.PhaseID,
		BOMType:     "MBOM",
		SourceBOMID: &srcBOM.ID,
		Version:     "v1.0",
		Name:        srcBOM.Name + " (MBOM)",
		Status:      "draft",
		Description: fmt.Sprintf("从 %s %s 转换而来", srcBOM.Name, srcBOM.Version),
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.bomRepo.Create(ctx, newBOM); err != nil {
		return nil, fmt.Errorf("create mbom: %w", err)
	}

	items, err := s.bomRepo.ListItemsByBOM(ctx, bomID)
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}

	if len(items) > 0 {
		var newItems []entity.ProjectBOMItem
		for _, item := range items {
			newItem := item
			newItem.ID = uuid.New().String()[:32]
			newItem.BOMID = newBOM.ID
			newItem.ParentItemID = nil
			newItem.CreatedAt = time.Now()
			newItem.UpdatedAt = time.Now()
			newItem.Material = nil
			newItem.ParentItem = nil
			newItem.Children = nil
			newItem.Drawings = nil
			newItem.CMFVariants = nil
			newItem.LangVariants = nil
			newItem.ProcessStep = nil
			newItems = append(newItems, newItem)
		}
		if err := s.bomRepo.BatchCreateItems(ctx, newItems); err != nil {
			return nil, fmt.Errorf("copy items: %w", err)
		}
		newBOM.TotalItems = len(newItems)
		newBOM.EstimatedCost = srcBOM.EstimatedCost
		s.bomRepo.Update(ctx, newBOM)
	}

	return s.bomRepo.FindByID(ctx, newBOM.ID)
}

// ==================== BOM版本发布 ====================

// ReleaseBOM 发布BOM（draft→released，自动生成版本号）
func (s *ProjectBOMService) ReleaseBOM(ctx context.Context, bomID, userID, releaseNote string) (*entity.ProjectBOM, error) {
	bom, err := s.bomRepo.FindByID(ctx, bomID)
	if err != nil {
		return nil, fmt.Errorf("bom not found: %w", err)
	}

	if bom.Status != "draft" {
		return nil, fmt.Errorf("只有草稿状态的BOM才能发布")
	}

	count, _ := s.bomRepo.CountItems(ctx, bomID)
	if count == 0 {
		return nil, fmt.Errorf("BOM没有物料行项，无法发布")
	}

	// Find max version for this project + bom_type
	allBoms, _ := s.bomRepo.ListByProject(ctx, bom.ProjectID, bom.BOMType, "")
	var maxMajor, maxMinor int
	for _, b := range allBoms {
		if b.ID == bomID {
			continue
		}
		if b.VersionMajor > maxMajor || (b.VersionMajor == maxMajor && b.VersionMinor > maxMinor) {
			maxMajor = b.VersionMajor
			maxMinor = b.VersionMinor
		}
	}

	// Determine new version: first release = v1.0, subsequent = minor increment
	var newMajor, newMinor int
	if maxMajor == 0 && maxMinor == 0 {
		newMajor = 1
		newMinor = 0
	} else {
		newMajor = maxMajor
		newMinor = maxMinor + 1
	}

	// Mark old released versions of same project+type as obsolete
	for i := range allBoms {
		if allBoms[i].Status == "released" && allBoms[i].ID != bomID {
			allBoms[i].Status = "obsolete"
			s.bomRepo.Update(ctx, &allBoms[i])
		}
	}

	now := time.Now()
	bom.Status = "released"
	bom.VersionMajor = newMajor
	bom.VersionMinor = newMinor
	bom.Version = fmt.Sprintf("v%d.%d", newMajor, newMinor)
	bom.ReleasedAt = &now
	bom.ReleasedBy = &userID
	bom.ReleaseNote = releaseNote
	bom.TotalItems = int(count)

	if err := s.bomRepo.Update(ctx, bom); err != nil {
		return nil, fmt.Errorf("release bom: %w", err)
	}

	return s.bomRepo.FindByID(ctx, bomID)
}

// CreateFromBOM 从已发布的上游BOM创建新的下游BOM（EBOM→PBOM 或 PBOM→MBOM）
func (s *ProjectBOMService) CreateFromBOM(ctx context.Context, projectID, sourceBomID, targetType, userID string) (*entity.ProjectBOM, error) {
	srcBOM, err := s.bomRepo.FindByID(ctx, sourceBomID)
	if err != nil {
		return nil, fmt.Errorf("source bom not found: %w", err)
	}

	if srcBOM.Status != "released" {
		return nil, fmt.Errorf("只能从已发布的BOM创建，当前状态: %s", srcBOM.Status)
	}

	// Validate transition: EBOM→PBOM, PBOM→MBOM
	validTransitions := map[string]string{"EBOM": "PBOM", "PBOM": "MBOM"}
	expected, ok := validTransitions[srcBOM.BOMType]
	if !ok || expected != targetType {
		return nil, fmt.Errorf("无效的BOM类型转换: %s → %s", srcBOM.BOMType, targetType)
	}

	newBOM := &entity.ProjectBOM{
		ID:            uuid.New().String()[:32],
		ProjectID:     projectID,
		PhaseID:       srcBOM.PhaseID,
		BOMType:       targetType,
		SourceBOMID:   &srcBOM.ID,
		SourceVersion: srcBOM.Version,
		Version:       "",
		Name:          fmt.Sprintf("%s (from %s %s)", targetType, srcBOM.BOMType, srcBOM.Version),
		Status:        "draft",
		Description:   fmt.Sprintf("从 %s %s 创建", srcBOM.BOMType, srcBOM.Version),
		CreatedBy:     userID,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := s.bomRepo.Create(ctx, newBOM); err != nil {
		return nil, fmt.Errorf("create %s: %w", targetType, err)
	}

	// Copy items from source BOM
	items, err := s.bomRepo.ListItemsByBOM(ctx, sourceBomID)
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}

	if len(items) > 0 {
		var newItems []entity.ProjectBOMItem
		for _, item := range items {
			newItem := item
			newItem.ID = uuid.New().String()[:32]
			newItem.BOMID = newBOM.ID
			newItem.ParentItemID = nil
			newItem.CreatedAt = time.Now()
			newItem.UpdatedAt = time.Now()
			newItem.Material = nil
			newItem.ParentItem = nil
			newItem.Children = nil
			newItem.Drawings = nil
			newItem.CMFVariants = nil
			newItem.LangVariants = nil
			newItem.ProcessStep = nil
			newItems = append(newItems, newItem)
		}
		if err := s.bomRepo.BatchCreateItems(ctx, newItems); err != nil {
			return nil, fmt.Errorf("copy items: %w", err)
		}
		newBOM.TotalItems = len(newItems)
		newBOM.EstimatedCost = srcBOM.EstimatedCost
		s.bomRepo.Update(ctx, newBOM)
	}

	return s.bomRepo.FindByID(ctx, newBOM.ID)
}

// CompareBOMs 对比两个BOM的行项差异
func (s *ProjectBOMService) CompareBOMs(ctx context.Context, bom1ID, bom2ID string) (*BOMCompareResult, error) {
	bom1, err := s.bomRepo.FindByID(ctx, bom1ID)
	if err != nil {
		return nil, fmt.Errorf("bom1 not found: %w", err)
	}
	bom2, err := s.bomRepo.FindByID(ctx, bom2ID)
	if err != nil {
		return nil, fmt.Errorf("bom2 not found: %w", err)
	}

	items1, _ := s.bomRepo.ListItemsByBOM(ctx, bom1ID)
	items2, _ := s.bomRepo.ListItemsByBOM(ctx, bom2ID)

	makeKey := func(item entity.ProjectBOMItem) string {
		return item.Name + "|" + getExtAttr(item.ExtendedAttrs, "manufacturer_pn")
	}

	map1 := make(map[string]entity.ProjectBOMItem)
	for _, item := range items1 {
		map1[makeKey(item)] = item
	}
	map2 := make(map[string]entity.ProjectBOMItem)
	for _, item := range items2 {
		map2[makeKey(item)] = item
	}

	result := &BOMCompareResult{
		BOM1: BOMSummary{ID: bom1.ID, Name: bom1.Name, Version: bom1.Version, BOMType: bom1.BOMType},
		BOM2: BOMSummary{ID: bom2.ID, Name: bom2.Name, Version: bom2.Version, BOMType: bom2.BOMType},
	}

	for key, item1 := range map1 {
		if item2, exists := map2[key]; exists {
			changes := compareItemFields(item1, item2)
			if len(changes) > 0 {
				result.Changed = append(result.Changed, BOMItemDiff{Key: key, Item1: item1, Item2: item2, Changes: changes})
			} else {
				result.Unchanged = append(result.Unchanged, item1)
			}
		} else {
			result.Removed = append(result.Removed, item1)
		}
	}
	for key, item2 := range map2 {
		if _, exists := map1[key]; !exists {
			result.Added = append(result.Added, item2)
		}
	}

	return result, nil
}

func compareItemFields(a, b entity.ProjectBOMItem) []FieldChange {
	var changes []FieldChange
	if a.Quantity != b.Quantity {
		changes = append(changes, FieldChange{Field: "quantity", Old: fmt.Sprintf("%.4f", a.Quantity), New: fmt.Sprintf("%.4f", b.Quantity)})
	}
	if !floatPtrEqual(a.UnitPrice, b.UnitPrice) {
		changes = append(changes, FieldChange{Field: "unit_price", Old: floatPtrStr(a.UnitPrice), New: floatPtrStr(b.UnitPrice)})
	}
	if a.Supplier != b.Supplier {
		changes = append(changes, FieldChange{Field: "supplier", Old: a.Supplier, New: b.Supplier})
	}
	aSupplierPN := getExtAttr(a.ExtendedAttrs, "supplier_pn")
	bSupplierPN := getExtAttr(b.ExtendedAttrs, "supplier_pn")
	if aSupplierPN != bSupplierPN {
		changes = append(changes, FieldChange{Field: "supplier_pn", Old: aSupplierPN, New: bSupplierPN})
	}
	aSpec := getExtAttr(a.ExtendedAttrs, "specification")
	bSpec := getExtAttr(b.ExtendedAttrs, "specification")
	if aSpec != bSpec {
		changes = append(changes, FieldChange{Field: "specification", Old: aSpec, New: bSpec})
	}
	if a.Unit != b.Unit {
		changes = append(changes, FieldChange{Field: "unit", Old: a.Unit, New: b.Unit})
	}
	aRef := getExtAttr(a.ExtendedAttrs, "reference")
	bRef := getExtAttr(b.ExtendedAttrs, "reference")
	if aRef != bRef {
		changes = append(changes, FieldChange{Field: "reference", Old: aRef, New: bRef})
	}
	aIsCritical := getExtAttrBool(a.ExtendedAttrs, "is_critical")
	bIsCritical := getExtAttrBool(b.ExtendedAttrs, "is_critical")
	if aIsCritical != bIsCritical {
		changes = append(changes, FieldChange{Field: "is_critical", Old: fmt.Sprintf("%v", aIsCritical), New: fmt.Sprintf("%v", bIsCritical)})
	}
	return changes
}

func floatPtrEqual(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return math.Abs(*a-*b) < 0.0001
}

func floatPtrStr(p *float64) string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("%.4f", *p)
}

// ==================== ERP发布 ====================

func (s *ProjectBOMService) CreateBOMRelease(ctx context.Context, bom *entity.ProjectBOM) (*entity.BOMRelease, error) {
	items, err := s.bomRepo.ListItemsByBOM(ctx, bom.ID)
	if err != nil {
		return nil, fmt.Errorf("list items for snapshot: %w", err)
	}
	snapshot := map[string]interface{}{"bom": bom, "items": items}
	snapshotBytes, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("marshal snapshot: %w", err)
	}

	release := &entity.BOMRelease{
		ID:           uuid.New().String(),
		BOMID:        bom.ID,
		ProjectID:    bom.ProjectID,
		BOMType:      bom.BOMType,
		Version:      bom.Version,
		SnapshotJSON: string(snapshotBytes),
		Status:       "pending",
		CreatedAt:    time.Now(),
	}
	if err := s.bomRepo.CreateRelease(ctx, release); err != nil {
		return nil, fmt.Errorf("create release: %w", err)
	}
	return release, nil
}

func (s *ProjectBOMService) ListPendingReleases(ctx context.Context) ([]entity.BOMRelease, error) {
	return s.bomRepo.ListPendingReleases(ctx)
}

func (s *ProjectBOMService) AckRelease(ctx context.Context, releaseID string) (*entity.BOMRelease, error) {
	release, err := s.bomRepo.FindReleaseByID(ctx, releaseID)
	if err != nil {
		return nil, fmt.Errorf("release not found: %w", err)
	}
	if release.Status != "pending" {
		return nil, fmt.Errorf("只有pending状态的发布可以确认")
	}
	now := time.Now()
	release.Status = "synced"
	release.SyncedAt = &now
	if err := s.bomRepo.UpdateRelease(ctx, release); err != nil {
		return nil, fmt.Errorf("update release: %w", err)
	}
	return release, nil
}

// ==================== 属性模板 CRUD ====================

func (s *ProjectBOMService) ListTemplates(ctx context.Context, category, subCategory string) ([]entity.CategoryAttrTemplate, error) {
	return s.bomRepo.ListTemplates(ctx, category, subCategory)
}

func (s *ProjectBOMService) CreateTemplate(ctx context.Context, input *TemplateInput) (*entity.CategoryAttrTemplate, error) {
	t := &entity.CategoryAttrTemplate{
		ID:           uuid.New().String()[:32],
		Category:     input.Category,
		SubCategory:  input.SubCategory,
		FieldKey:     input.FieldKey,
		FieldName:    input.FieldName,
		FieldType:    input.FieldType,
		Unit:         input.Unit,
		Required:     input.Required,
		DefaultValue: input.DefaultValue,
		SortOrder:    input.SortOrder,
		ShowInTable:  input.ShowInTable,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if input.Options != nil {
		t.Options = entity.JSONB(input.Options)
	}
	if input.Validation != nil {
		t.Validation = entity.JSONB(input.Validation)
	}
	if err := s.bomRepo.CreateTemplate(ctx, t); err != nil {
		return nil, fmt.Errorf("创建属性模板失败: %w", err)
	}
	return t, nil
}

func (s *ProjectBOMService) UpdateTemplate(ctx context.Context, id string, input *TemplateInput) (*entity.CategoryAttrTemplate, error) {
	t, err := s.bomRepo.FindTemplateByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("属性模板不存在: %w", err)
	}
	if input.FieldName != "" {
		t.FieldName = input.FieldName
	}
	if input.FieldType != "" {
		t.FieldType = input.FieldType
	}
	t.Unit = input.Unit
	t.Required = input.Required
	t.DefaultValue = input.DefaultValue
	t.SortOrder = input.SortOrder
	t.ShowInTable = input.ShowInTable
	if input.Options != nil {
		t.Options = entity.JSONB(input.Options)
	}
	if input.Validation != nil {
		t.Validation = entity.JSONB(input.Validation)
	}
	t.UpdatedAt = time.Now()
	if err := s.bomRepo.UpdateTemplate(ctx, t); err != nil {
		return nil, fmt.Errorf("更新属性模板失败: %w", err)
	}
	return t, nil
}

func (s *ProjectBOMService) DeleteTemplate(ctx context.Context, id string) error {
	return s.bomRepo.DeleteTemplate(ctx, id)
}

func (s *ProjectBOMService) GetCategoryTree(ctx context.Context, bomID string) ([]map[string]interface{}, error) {
	return s.bomRepo.GetCategoryTree(ctx, bomID)
}

// ==================== 工艺路线 CRUD ====================

func (s *ProjectBOMService) CreateRoute(ctx context.Context, projectID string, input *RouteInput, createdBy string) (*entity.ProcessRoute, error) {
	route := &entity.ProcessRoute{
		ID:          uuid.New().String()[:32],
		ProjectID:   projectID,
		BOMID:       input.BOMID,
		Name:        input.Name,
		Version:     input.Version,
		Status:      "draft",
		Description: input.Description,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if route.Version == "" {
		route.Version = "v1.0"
	}
	if err := s.bomRepo.CreateRoute(ctx, route); err != nil {
		return nil, fmt.Errorf("创建工艺路线失败: %w", err)
	}
	return route, nil
}

func (s *ProjectBOMService) GetRoute(ctx context.Context, id string) (*entity.ProcessRoute, error) {
	return s.bomRepo.FindRouteByID(ctx, id)
}

func (s *ProjectBOMService) ListRoutes(ctx context.Context, projectID string) ([]entity.ProcessRoute, error) {
	return s.bomRepo.ListRoutesByProject(ctx, projectID)
}

func (s *ProjectBOMService) UpdateRoute(ctx context.Context, id string, input *RouteInput) (*entity.ProcessRoute, error) {
	route, err := s.bomRepo.FindRouteByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("工艺路线不存在: %w", err)
	}
	if input.Name != "" {
		route.Name = input.Name
	}
	if input.Version != "" {
		route.Version = input.Version
	}
	if input.Description != "" {
		route.Description = input.Description
	}
	route.UpdatedAt = time.Now()
	if err := s.bomRepo.UpdateRoute(ctx, route); err != nil {
		return nil, fmt.Errorf("更新工艺路线失败: %w", err)
	}
	return route, nil
}

func (s *ProjectBOMService) CreateStep(ctx context.Context, routeID string, input *StepInput) (*entity.ProcessStep, error) {
	step := &entity.ProcessStep{
		ID:             uuid.New().String()[:32],
		RouteID:        routeID,
		StepNumber:     input.StepNumber,
		Name:           input.Name,
		WorkCenter:     input.WorkCenter,
		Description:    input.Description,
		StdTimeMinutes: input.StdTimeMinutes,
		SetupMinutes:   input.SetupMinutes,
		LaborCost:      input.LaborCost,
		SortOrder:      input.SortOrder,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := s.bomRepo.CreateStep(ctx, step); err != nil {
		return nil, fmt.Errorf("创建工序失败: %w", err)
	}
	route, _ := s.bomRepo.FindRouteByID(ctx, routeID)
	if route != nil {
		route.TotalSteps = len(route.Steps)
		s.bomRepo.UpdateRoute(ctx, route)
	}
	return step, nil
}

func (s *ProjectBOMService) UpdateStep(ctx context.Context, id string, input *StepInput) (*entity.ProcessStep, error) {
	step, err := s.bomRepo.FindStepByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("工序不存在: %w", err)
	}
	if input.Name != "" {
		step.Name = input.Name
	}
	step.StepNumber = input.StepNumber
	step.WorkCenter = input.WorkCenter
	step.Description = input.Description
	step.StdTimeMinutes = input.StdTimeMinutes
	step.SetupMinutes = input.SetupMinutes
	step.LaborCost = input.LaborCost
	step.SortOrder = input.SortOrder
	step.UpdatedAt = time.Now()
	if err := s.bomRepo.UpdateStep(ctx, step); err != nil {
		return nil, fmt.Errorf("更新工序失败: %w", err)
	}
	return step, nil
}

func (s *ProjectBOMService) DeleteStep(ctx context.Context, id string) error {
	step, err := s.bomRepo.FindStepByID(ctx, id)
	if err != nil {
		return fmt.Errorf("工序不存在: %w", err)
	}
	routeID := step.RouteID
	if err := s.bomRepo.DeleteStep(ctx, id); err != nil {
		return fmt.Errorf("删除工序失败: %w", err)
	}
	route, _ := s.bomRepo.FindRouteByID(ctx, routeID)
	if route != nil {
		route.TotalSteps = len(route.Steps)
		s.bomRepo.UpdateRoute(ctx, route)
	}
	return nil
}

func (s *ProjectBOMService) CreateStepMaterial(ctx context.Context, stepID string, input *StepMaterialInput) (*entity.ProcessStepMaterial, error) {
	m := &entity.ProcessStepMaterial{
		ID:         uuid.New().String()[:32],
		StepID:     stepID,
		MaterialID: input.MaterialID,
		Name:       input.Name,
		Category:   input.Category,
		Quantity:   input.Quantity,
		Unit:       input.Unit,
		Notes:      input.Notes,
		CreatedAt:  time.Now(),
	}
	if m.Unit == "" {
		m.Unit = "pcs"
	}
	if m.Category == "" {
		m.Category = "material"
	}
	if err := s.bomRepo.CreateStepMaterial(ctx, m); err != nil {
		return nil, fmt.Errorf("添加工序物料失败: %w", err)
	}
	return m, nil
}

func (s *ProjectBOMService) DeleteStepMaterial(ctx context.Context, id string) error {
	return s.bomRepo.DeleteStepMaterial(ctx, id)
}

// ==================== 自动建料 ====================

func (s *ProjectBOMService) BackfillMaterials(ctx context.Context) {
	var items []entity.ProjectBOMItem
	err := s.bomRepo.DB().WithContext(ctx).
		Where("material_id IS NULL AND name != ''").
		Find(&items).Error
	if err != nil || len(items) == 0 {
		return
	}

	bomTypes := map[string]string{}
	var boms []entity.ProjectBOM
	s.bomRepo.DB().WithContext(ctx).Select("id, bom_type").Find(&boms)
	for _, b := range boms {
		bomTypes[b.ID] = b.BOMType
	}

	count := 0
	for _, item := range items {
		category := item.Category
		if category == "" {
			category = defaultCategoryForBOMType(bomTypes[item.BOMID])
		}
		newMat, createErr := s.autoCreateMaterial(ctx, item.Name, getExtAttr(item.ExtendedAttrs, "specification"), category, getExtAttr(item.ExtendedAttrs, "manufacturer"), getExtAttr(item.ExtendedAttrs, "manufacturer_pn"))
		if createErr != nil {
			fmt.Printf("[BackfillMaterials] failed for %q: %v\n", item.Name, createErr)
			continue
		}
		if newMat != nil {
			s.bomRepo.DB().Model(&entity.ProjectBOMItem{}).Where("id = ?", item.ID).Update("material_id", newMat.ID)
			count++
		}
	}
	if count > 0 {
		fmt.Printf("[BackfillMaterials] created materials for %d BOM items\n", count)
	}
}

func (s *ProjectBOMService) autoCreateMaterial(ctx context.Context, name, specification, category, manufacturer, manufacturerPN string) (*entity.Material, error) {
	categoryID, categoryCode := mapCategoryToIDAndCode(category)
	code, err := s.materialRepo.GenerateCode(ctx, categoryCode)
	if err != nil {
		return nil, fmt.Errorf("generate material code: %w", err)
	}
	mat := &entity.Material{
		ID:          uuid.New().String()[:32],
		Code:        code,
		Name:        name,
		CategoryID:  categoryID,
		Status:      "active",
		Unit:        "pcs",
		Description: specification,
		CreatedBy:   "u_admin",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.materialRepo.Create(ctx, mat); err != nil {
		return nil, fmt.Errorf("create material: %w", err)
	}
	return mat, nil
}

func mapCategoryToIDAndCode(category string) (string, string) {
	idToCode := map[string]string{
		"mcat_el_res": "EL-RES", "mcat_el_cap": "EL-CAP", "mcat_el_ind": "EL-IND",
		"mcat_el_ic": "EL-IC", "mcat_el_con": "EL-CON", "mcat_el_dio": "EL-DIO",
		"mcat_el_trn": "EL-TRN", "mcat_el_osc": "EL-OSC", "mcat_el_led": "EL-LED",
		"mcat_el_sen": "EL-SEN", "mcat_el_ant": "EL-ANT", "mcat_el_mod": "EL-MOD",
		"mcat_el_bat": "EL-BAT", "mcat_el_pcb": "EL-PCB", "mcat_el_oth": "EL-OTH",
		"mcat_me_hsg": "ME-HSG", "mcat_me_lns": "ME-LNS", "mcat_me_dec": "ME-DEC",
		"mcat_me_fst": "ME-FST", "mcat_me_gsk": "ME-GSK", "mcat_me_thm": "ME-THM",
		"mcat_me_spg": "ME-SPG", "mcat_me_flx": "ME-FLX", "mcat_me_oth": "ME-OTH",
		"mcat_pk_box": "PK-BOX", "mcat_pk_ins": "PK-INS", "mcat_pk_lbl": "PK-LBL",
		"mcat_pk_try": "PK-TRY", "mcat_pk_bag": "PK-BAG",
	}
	if code, ok := idToCode[category]; ok {
		return category, code
	}
	switch category {
	case "电阻":
		return "mcat_el_res", "EL-RES"
	case "电容":
		return "mcat_el_cap", "EL-CAP"
	case "电感":
		return "mcat_el_ind", "EL-IND"
	case "IC", "集成电路":
		return "mcat_el_ic", "EL-IC"
	case "连接器":
		return "mcat_el_con", "EL-CON"
	case "二极管", "ESD器件":
		return "mcat_el_dio", "EL-DIO"
	case "晶体管":
		return "mcat_el_trn", "EL-TRN"
	case "晶振":
		return "mcat_el_osc", "EL-OSC"
	case "LED":
		return "mcat_el_led", "EL-LED"
	case "传感器":
		return "mcat_el_sen", "EL-SEN"
	case "测试点":
		return "mcat_el_oth", "EL-OTH"
	case "结构件", "外壳", "壳体":
		return "mcat_me_hsg", "ME-HSG"
	case "镜片":
		return "mcat_me_lns", "ME-LNS"
	case "装饰件":
		return "mcat_me_dec", "ME-DEC"
	case "紧固件":
		return "mcat_me_fst", "ME-FST"
	case "密封件":
		return "mcat_me_gsk", "ME-GSK"
	case "散热件":
		return "mcat_me_thm", "ME-THM"
	case "包装盒", "外包装盒":
		return "mcat_pk_box", "PK-BOX"
	case "说明书":
		return "mcat_pk_ins", "PK-INS"
	case "标签", "标签贴纸":
		return "mcat_pk_lbl", "PK-LBL"
	case "托盘", "内衬/托盘", "内衬":
		return "mcat_pk_try", "PK-TRY"
	default:
		return "mcat_el_oth", "EL-OTH"
	}
}

func defaultCategoryForBOMType(bomType string) string {
	switch bomType {
	case "PBOM", "SBOM":
		return "结构件"
	default:
		return ""
	}
}

// SeedDefaultTemplates 种子默认属性模板
func (s *ProjectBOMService) SeedDefaultTemplates(ctx context.Context) {
	existing, _ := s.bomRepo.ListTemplates(ctx, "", "")
	if len(existing) > 0 {
		return
	}

	sel := func(opts ...string) entity.JSONB {
		vals := make([]interface{}, len(opts))
		for i, o := range opts {
			vals[i] = o
		}
		return entity.JSONB{"values": vals}
	}

	// BOM type groupings: EBOM = electronic/structural/optical, PBOM = packaging/tooling/consumable
	eb, pb := "EBOM", "PBOM"

	templates := []entity.CategoryAttrTemplate{
		// ===== electronic/component 元器件 (EBOM) =====
		{Category: "electronic", SubCategory: "component", BOMType: eb, FieldKey: "designator", FieldName: "位号", FieldType: "text", SortOrder: 1, ShowInTable: true},
		{Category: "electronic", SubCategory: "component", BOMType: eb, FieldKey: "package", FieldName: "封装", FieldType: "text", SortOrder: 2, ShowInTable: true},
		{Category: "electronic", SubCategory: "component", BOMType: eb, FieldKey: "specification", FieldName: "规格参数", FieldType: "text", SortOrder: 3, ShowInTable: true},
		{Category: "electronic", SubCategory: "component", BOMType: eb, FieldKey: "manufacturer", FieldName: "制造商", FieldType: "text", SortOrder: 4, ShowInTable: true},
		{Category: "electronic", SubCategory: "component", BOMType: eb, FieldKey: "manufacturer_pn", FieldName: "制造商料号", FieldType: "text", SortOrder: 5, ShowInTable: true},
		{Category: "electronic", SubCategory: "component", BOMType: eb, FieldKey: "is_critical", FieldName: "关键器件", FieldType: "boolean", SortOrder: 6, ShowInTable: false},

		// ===== electronic/pcb PCB (EBOM) =====
		{Category: "electronic", SubCategory: "pcb", BOMType: eb, FieldKey: "layers", FieldName: "层数", FieldType: "number", SortOrder: 1, ShowInTable: true},
		{Category: "electronic", SubCategory: "pcb", BOMType: eb, FieldKey: "thickness", FieldName: "板厚", FieldType: "text", Unit: "mm", SortOrder: 2, ShowInTable: true},
		{Category: "electronic", SubCategory: "pcb", BOMType: eb, FieldKey: "material", FieldName: "板材", FieldType: "select", SortOrder: 3, ShowInTable: true, Options: sel("FR4", "Rogers", "FPC(PI)", "铝基板", "陶瓷基板")},
		{Category: "electronic", SubCategory: "pcb", BOMType: eb, FieldKey: "dimensions", FieldName: "尺寸", FieldType: "text", Unit: "mm", SortOrder: 4, ShowInTable: true},
		{Category: "electronic", SubCategory: "pcb", BOMType: eb, FieldKey: "surface_finish", FieldName: "表面工艺", FieldType: "select", SortOrder: 5, ShowInTable: true, Options: sel("沉金(ENIG)", "喷锡(HASL)", "OSP", "沉银", "沉锡")},
		{Category: "electronic", SubCategory: "pcb", BOMType: eb, FieldKey: "min_trace", FieldName: "最小线宽", FieldType: "text", Unit: "mil", SortOrder: 6, ShowInTable: false},
		{Category: "electronic", SubCategory: "pcb", BOMType: eb, FieldKey: "manufacturer", FieldName: "制造商", FieldType: "text", SortOrder: 7, ShowInTable: true},

		// ===== electronic/connector 连接器 (EBOM) =====
		{Category: "electronic", SubCategory: "connector", BOMType: eb, FieldKey: "designator", FieldName: "位号", FieldType: "text", SortOrder: 1, ShowInTable: true},
		{Category: "electronic", SubCategory: "connector", BOMType: eb, FieldKey: "package", FieldName: "封装/类型", FieldType: "text", SortOrder: 2, ShowInTable: true},
		{Category: "electronic", SubCategory: "connector", BOMType: eb, FieldKey: "specification", FieldName: "规格", FieldType: "text", SortOrder: 3, ShowInTable: true},
		{Category: "electronic", SubCategory: "connector", BOMType: eb, FieldKey: "pin_count", FieldName: "针脚数", FieldType: "number", SortOrder: 4, ShowInTable: true},
		{Category: "electronic", SubCategory: "connector", BOMType: eb, FieldKey: "manufacturer", FieldName: "制造商", FieldType: "text", SortOrder: 5, ShowInTable: true},
		{Category: "electronic", SubCategory: "connector", BOMType: eb, FieldKey: "manufacturer_pn", FieldName: "制造商料号", FieldType: "text", SortOrder: 6, ShowInTable: true},

		// ===== electronic/cable 线缆 (EBOM) =====
		{Category: "electronic", SubCategory: "cable", BOMType: eb, FieldKey: "cable_type", FieldName: "类型", FieldType: "select", SortOrder: 1, ShowInTable: true, Options: sel("FPC", "FFC", "同轴线", "排线", "USB-C线")},
		{Category: "electronic", SubCategory: "cable", BOMType: eb, FieldKey: "length", FieldName: "长度", FieldType: "number", Unit: "mm", SortOrder: 2, ShowInTable: true},
		{Category: "electronic", SubCategory: "cable", BOMType: eb, FieldKey: "pin_count", FieldName: "针脚/芯数", FieldType: "number", SortOrder: 3, ShowInTable: true},
		{Category: "electronic", SubCategory: "cable", BOMType: eb, FieldKey: "specification", FieldName: "规格", FieldType: "text", SortOrder: 4, ShowInTable: true},
		{Category: "electronic", SubCategory: "cable", BOMType: eb, FieldKey: "manufacturer", FieldName: "制造商", FieldType: "text", SortOrder: 5, ShowInTable: true},

		// ===== structural/structural_part 结构件 (EBOM) =====
		{Category: "structural", SubCategory: "structural_part", BOMType: eb, FieldKey: "is_appearance_part", FieldName: "是否外观件", FieldType: "boolean", SortOrder: 1, ShowInTable: true},
		{Category: "structural", SubCategory: "structural_part", BOMType: eb, FieldKey: "material_type", FieldName: "材质", FieldType: "text", SortOrder: 2, ShowInTable: true},
		{Category: "structural", SubCategory: "structural_part", BOMType: eb, FieldKey: "process_type", FieldName: "工艺", FieldType: "select", SortOrder: 3, ShowInTable: true, Options: sel("注塑", "CNC", "冲压", "模切", "3D打印", "激光切割", "压铸")},
		{Category: "structural", SubCategory: "structural_part", BOMType: eb, FieldKey: "surface_treatment", FieldName: "表面处理", FieldType: "text", SortOrder: 4, ShowInTable: true},
		{Category: "structural", SubCategory: "structural_part", BOMType: eb, FieldKey: "color", FieldName: "颜色", FieldType: "text", SortOrder: 5, ShowInTable: true},
		{Category: "structural", SubCategory: "structural_part", BOMType: eb, FieldKey: "tolerance_grade", FieldName: "公差等级", FieldType: "select", SortOrder: 6, ShowInTable: false, Options: sel("普通(±0.1mm)", "精密(±0.05mm)", "超精密(±0.02mm)")},
		{Category: "structural", SubCategory: "structural_part", BOMType: eb, FieldKey: "weight_grams", FieldName: "重量", FieldType: "number", Unit: "g", SortOrder: 7, ShowInTable: true},
		{Category: "structural", SubCategory: "structural_part", BOMType: eb, FieldKey: "thumbnail_url", FieldName: "缩略图", FieldType: "thumbnail", SortOrder: 8, ShowInTable: true},
		{Category: "structural", SubCategory: "structural_part", BOMType: eb, FieldKey: "drawing_2d", FieldName: "2D图纸", FieldType: "file", SortOrder: 9, ShowInTable: true},
		{Category: "structural", SubCategory: "structural_part", BOMType: eb, FieldKey: "drawing_3d", FieldName: "3D模型", FieldType: "file", SortOrder: 10, ShowInTable: true},
		{Category: "structural", SubCategory: "structural_part", BOMType: eb, FieldKey: "tooling_estimate", FieldName: "模具费", FieldType: "number", Unit: "¥", SortOrder: 11, ShowInTable: true},
		{Category: "structural", SubCategory: "structural_part", BOMType: eb, FieldKey: "target_price", FieldName: "目标单价", FieldType: "number", Unit: "¥", SortOrder: 12, ShowInTable: false},
		{Category: "structural", SubCategory: "structural_part", BOMType: eb, FieldKey: "assembly_method", FieldName: "装配方式", FieldType: "select", SortOrder: 13, ShowInTable: false, Options: sel("卡扣", "螺丝", "胶合", "超声波焊接", "热熔")},
		{Category: "structural", SubCategory: "structural_part", BOMType: eb, FieldKey: "is_variant", FieldName: "变体件", FieldType: "boolean", SortOrder: 14, ShowInTable: false},

		// ===== structural/fastener 紧固件 (EBOM) =====
		{Category: "structural", SubCategory: "fastener", BOMType: eb, FieldKey: "specification", FieldName: "规格", FieldType: "text", SortOrder: 1, ShowInTable: true},
		{Category: "structural", SubCategory: "fastener", BOMType: eb, FieldKey: "material_type", FieldName: "材质", FieldType: "select", SortOrder: 2, ShowInTable: true, Options: sel("不锈钢304", "碳钢镀锌", "铜", "尼龙")},
		{Category: "structural", SubCategory: "fastener", BOMType: eb, FieldKey: "standard", FieldName: "标准", FieldType: "text", SortOrder: 3, ShowInTable: false},

		// ===== optical/light_engine 光机 (EBOM) =====
		{Category: "optical", SubCategory: "light_engine", BOMType: eb, FieldKey: "brightness", FieldName: "亮度", FieldType: "number", Unit: "lm", SortOrder: 1, ShowInTable: true},
		{Category: "optical", SubCategory: "light_engine", BOMType: eb, FieldKey: "contrast", FieldName: "对比度", FieldType: "text", SortOrder: 2, ShowInTable: true},
		{Category: "optical", SubCategory: "light_engine", BOMType: eb, FieldKey: "resolution", FieldName: "分辨率", FieldType: "text", SortOrder: 3, ShowInTable: true},
		{Category: "optical", SubCategory: "light_engine", BOMType: eb, FieldKey: "power", FieldName: "功耗", FieldType: "number", Unit: "W", SortOrder: 4, ShowInTable: true},
		{Category: "optical", SubCategory: "light_engine", BOMType: eb, FieldKey: "interface_type", FieldName: "接口类型", FieldType: "text", SortOrder: 5, ShowInTable: true},
		{Category: "optical", SubCategory: "light_engine", BOMType: eb, FieldKey: "offset", FieldName: "Offset", FieldType: "text", SortOrder: 6, ShowInTable: false},
		{Category: "optical", SubCategory: "light_engine", BOMType: eb, FieldKey: "manufacturer", FieldName: "制造商", FieldType: "text", SortOrder: 7, ShowInTable: true},
		{Category: "optical", SubCategory: "light_engine", BOMType: eb, FieldKey: "manufacturer_pn", FieldName: "制造商料号", FieldType: "text", SortOrder: 8, ShowInTable: false},

		// ===== optical/waveguide 波导 (EBOM) =====
		{Category: "optical", SubCategory: "waveguide", BOMType: eb, FieldKey: "transmittance", FieldName: "透过率", FieldType: "number", Unit: "%", SortOrder: 1, ShowInTable: true},
		{Category: "optical", SubCategory: "waveguide", BOMType: eb, FieldKey: "refractive_index", FieldName: "折射率", FieldType: "number", SortOrder: 2, ShowInTable: true},
		{Category: "optical", SubCategory: "waveguide", BOMType: eb, FieldKey: "thickness", FieldName: "厚度", FieldType: "number", Unit: "mm", SortOrder: 3, ShowInTable: true},
		{Category: "optical", SubCategory: "waveguide", BOMType: eb, FieldKey: "fov", FieldName: "FOV视场角", FieldType: "number", Unit: "°", SortOrder: 4, ShowInTable: true},
		{Category: "optical", SubCategory: "waveguide", BOMType: eb, FieldKey: "eye_box", FieldName: "眼盒", FieldType: "text", Unit: "mm", SortOrder: 5, ShowInTable: true},
		{Category: "optical", SubCategory: "waveguide", BOMType: eb, FieldKey: "diffraction_efficiency", FieldName: "衍射效率", FieldType: "number", Unit: "%", SortOrder: 6, ShowInTable: false},
		{Category: "optical", SubCategory: "waveguide", BOMType: eb, FieldKey: "manufacturer", FieldName: "制造商", FieldType: "text", SortOrder: 7, ShowInTable: true},

		// ===== optical/lens 普通镜片 (EBOM) =====
		{Category: "optical", SubCategory: "lens", BOMType: eb, FieldKey: "transmittance", FieldName: "透光率", FieldType: "number", Unit: "%", SortOrder: 1, ShowInTable: true},
		{Category: "optical", SubCategory: "lens", BOMType: eb, FieldKey: "coating", FieldName: "镀膜类型", FieldType: "text", SortOrder: 2, ShowInTable: true},
		{Category: "optical", SubCategory: "lens", BOMType: eb, FieldKey: "material_type", FieldName: "材质", FieldType: "text", SortOrder: 3, ShowInTable: true},
		{Category: "optical", SubCategory: "lens", BOMType: eb, FieldKey: "refractive_index", FieldName: "折射率", FieldType: "number", SortOrder: 4, ShowInTable: false},
		{Category: "optical", SubCategory: "lens", BOMType: eb, FieldKey: "manufacturer", FieldName: "制造商", FieldType: "text", SortOrder: 5, ShowInTable: true},

		// ===== optical/lightguide 导光板 (EBOM) =====
		{Category: "optical", SubCategory: "lightguide", BOMType: eb, FieldKey: "material_type", FieldName: "材质", FieldType: "text", SortOrder: 1, ShowInTable: true},
		{Category: "optical", SubCategory: "lightguide", BOMType: eb, FieldKey: "dimensions", FieldName: "尺寸", FieldType: "text", Unit: "mm", SortOrder: 2, ShowInTable: true},
		{Category: "optical", SubCategory: "lightguide", BOMType: eb, FieldKey: "transmittance", FieldName: "透光率", FieldType: "number", Unit: "%", SortOrder: 3, ShowInTable: true},
		{Category: "optical", SubCategory: "lightguide", BOMType: eb, FieldKey: "manufacturer", FieldName: "制造商", FieldType: "text", SortOrder: 4, ShowInTable: true},

		// ===== packaging/box 彩盒 (PBOM) =====
		{Category: "packaging", SubCategory: "box", BOMType: pb, FieldKey: "material_type", FieldName: "材质", FieldType: "text", SortOrder: 1, ShowInTable: true},
		{Category: "packaging", SubCategory: "box", BOMType: pb, FieldKey: "dimensions", FieldName: "尺寸", FieldType: "text", Unit: "mm", SortOrder: 2, ShowInTable: true},
		{Category: "packaging", SubCategory: "box", BOMType: pb, FieldKey: "print_process", FieldName: "印刷工艺", FieldType: "select", SortOrder: 3, ShowInTable: true, Options: sel("四色", "专色", "黑白")},
		{Category: "packaging", SubCategory: "box", BOMType: pb, FieldKey: "surface_finish", FieldName: "表面处理", FieldType: "select", SortOrder: 4, ShowInTable: true, Options: sel("覆膜亮", "覆膜哑", "UV", "过油")},
		{Category: "packaging", SubCategory: "box", BOMType: pb, FieldKey: "packing_qty", FieldName: "装箱数", FieldType: "number", Unit: "台/箱", SortOrder: 5, ShowInTable: true},
		{Category: "packaging", SubCategory: "box", BOMType: pb, FieldKey: "design_file", FieldName: "设计稿", FieldType: "file", SortOrder: 6, ShowInTable: true},
		{Category: "packaging", SubCategory: "box", BOMType: pb, FieldKey: "die_cut_file", FieldName: "刀模图", FieldType: "file", SortOrder: 7, ShowInTable: true},

		// ===== packaging/document 说明书/卡片 (PBOM) =====
		{Category: "packaging", SubCategory: "document", BOMType: pb, FieldKey: "print_process", FieldName: "印刷工艺", FieldType: "select", SortOrder: 1, ShowInTable: true, Options: sel("四色", "专色", "黑白")},
		{Category: "packaging", SubCategory: "document", BOMType: pb, FieldKey: "language_code", FieldName: "语言版本", FieldType: "select", SortOrder: 2, ShowInTable: true, Options: sel("通用", "中文", "英文", "日文", "韩文")},
		{Category: "packaging", SubCategory: "document", BOMType: pb, FieldKey: "is_multilang", FieldName: "多语言件", FieldType: "boolean", SortOrder: 3, ShowInTable: true},
		{Category: "packaging", SubCategory: "document", BOMType: pb, FieldKey: "design_file", FieldName: "设计稿", FieldType: "file", SortOrder: 4, ShowInTable: true},

		// ===== packaging/cushion 内衬/缓冲 (PBOM) =====
		{Category: "packaging", SubCategory: "cushion", BOMType: pb, FieldKey: "material_type", FieldName: "材质", FieldType: "select", SortOrder: 1, ShowInTable: true, Options: sel("EVA", "EPE珍珠棉", "海绵", "吸塑PET")},
		{Category: "packaging", SubCategory: "cushion", BOMType: pb, FieldKey: "dimensions", FieldName: "尺寸", FieldType: "text", Unit: "mm", SortOrder: 2, ShowInTable: true},
		{Category: "packaging", SubCategory: "cushion", BOMType: pb, FieldKey: "die_cut_file", FieldName: "刀模图", FieldType: "file", SortOrder: 3, ShowInTable: true},

		// ===== tooling/mold 模具 (PBOM) =====
		{Category: "tooling", SubCategory: "mold", BOMType: pb, FieldKey: "mold_type", FieldName: "模具类型", FieldType: "select", SortOrder: 1, ShowInTable: true, Options: sel("注塑模", "冲压模", "压铸模", "吹塑模")},
		{Category: "tooling", SubCategory: "mold", BOMType: pb, FieldKey: "cavities", FieldName: "穴数", FieldType: "number", SortOrder: 2, ShowInTable: true},
		{Category: "tooling", SubCategory: "mold", BOMType: pb, FieldKey: "lifetime_shots", FieldName: "寿命", FieldType: "number", Unit: "模次", SortOrder: 3, ShowInTable: true},
		{Category: "tooling", SubCategory: "mold", BOMType: pb, FieldKey: "material_type", FieldName: "模具钢材", FieldType: "text", SortOrder: 4, ShowInTable: true},
		{Category: "tooling", SubCategory: "mold", BOMType: pb, FieldKey: "maintenance_cycle", FieldName: "保养周期", FieldType: "text", SortOrder: 5, ShowInTable: false},

		// ===== tooling/fixture 治具/检具 (PBOM) =====
		{Category: "tooling", SubCategory: "fixture", BOMType: pb, FieldKey: "fixture_type", FieldName: "治具类型", FieldType: "select", SortOrder: 1, ShowInTable: true, Options: sel("组装治具", "测试治具", "检具", "烧录治具", "点胶治具")},
		{Category: "tooling", SubCategory: "fixture", BOMType: pb, FieldKey: "purpose", FieldName: "用途说明", FieldType: "text", SortOrder: 2, ShowInTable: true},
		{Category: "tooling", SubCategory: "fixture", BOMType: pb, FieldKey: "drawing_no", FieldName: "图纸编号", FieldType: "text", SortOrder: 3, ShowInTable: true},

		// ===== consumable/consumable 辅料 (PBOM) =====
		{Category: "consumable", SubCategory: "consumable", BOMType: pb, FieldKey: "usage_unit", FieldName: "用量单位", FieldType: "select", SortOrder: 1, ShowInTable: true, Options: sel("g", "ml", "pcs", "cm")},
		{Category: "consumable", SubCategory: "consumable", BOMType: pb, FieldKey: "usage_per_unit", FieldName: "单台用量", FieldType: "number", SortOrder: 2, ShowInTable: true},
		{Category: "consumable", SubCategory: "consumable", BOMType: pb, FieldKey: "purchase_unit", FieldName: "采购单位", FieldType: "text", SortOrder: 3, ShowInTable: true},
		{Category: "consumable", SubCategory: "consumable", BOMType: pb, FieldKey: "conversion_rate", FieldName: "换算率", FieldType: "number", SortOrder: 4, ShowInTable: false},
		{Category: "consumable", SubCategory: "consumable", BOMType: pb, FieldKey: "shelf_life_days", FieldName: "保质期", FieldType: "number", Unit: "天", SortOrder: 5, ShowInTable: false},
		{Category: "consumable", SubCategory: "consumable", BOMType: pb, FieldKey: "manufacturer", FieldName: "制造商", FieldType: "text", SortOrder: 6, ShowInTable: true},
	}
	for i := range templates {
		templates[i].ID = uuid.New().String()[:32]
		templates[i].CreatedAt = time.Now()
		templates[i].UpdatedAt = time.Now()
		s.bomRepo.CreateTemplate(ctx, &templates[i])
	}
}

// ---- ParsedBOMItem for parse-only preview ----

type ParsedBOMItem struct {
	ItemNumber       int                    `json:"item_number"`
	Reference        string                 `json:"reference"`        // kept for API compat, stored in extended_attrs
	Name             string                 `json:"name"`
	Specification    string                 `json:"specification"`    // kept for API compat, stored in extended_attrs
	Quantity         float64                `json:"quantity"`
	Unit             string                 `json:"unit"`
	Category         string                 `json:"category"`
	SubCategory      string                 `json:"sub_category"`
	Manufacturer     string                 `json:"manufacturer"`    // kept for API compat, stored in extended_attrs
	ManufacturerPN   string                 `json:"manufacturer_pn"` // kept for API compat, stored in extended_attrs
	Supplier         string                 `json:"supplier"`
	UnitPrice        float64                `json:"unit_price"`
	LeadTimeDays     int                    `json:"lead_time_days"`  // kept for API compat, stored in extended_attrs
	IsCritical       bool                   `json:"is_critical"`     // kept for API compat, stored in extended_attrs
	IsAppearancePart bool                   `json:"is_appearance_part"` // kept for API compat, stored in extended_attrs
	DrawingNo        string                 `json:"drawing_no"`      // kept for API compat, stored in extended_attrs
	ThumbnailURL     string                 `json:"thumbnail_url"`
	Notes            string                 `json:"notes"`
	ExtendedAttrs    map[string]interface{} `json:"extended_attrs"`
}

// CreateBOMFromParsedItems 根据已解析的BOM条目创建项目BOM
func (s *ProjectBOMService) CreateBOMFromParsedItems(ctx context.Context, projectID, userID string, bomName string, items []ParsedBOMItem, bomType string) (string, error) {
	if bomType == "" {
		bomType = "EBOM"
	}

	// Find existing BOM of this type for this project (only draft ones can be merged into)
	existingBoms, _ := s.bomRepo.ListByProject(ctx, projectID, bomType, "")
	var bom *entity.ProjectBOM
	for i := range existingBoms {
		if existingBoms[i].Status == "draft" {
			bom = &existingBoms[i]
			break
		}
	}

	if bom == nil {
		// No existing draft BOM of this type — create one
		bom = &entity.ProjectBOM{
			ID:        uuid.New().String()[:32],
			ProjectID: projectID,
			BOMType:   bomType,
			Version:   "",
			Name:      bomType,
			Status:    "draft",
			CreatedBy: userID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := s.bomRepo.Create(ctx, bom); err != nil {
			return "", fmt.Errorf("create project bom: %w", err)
		}
		fmt.Printf("[CreateBOMFromParsedItems] created new %s: id=%s\n", bomType, bom.ID)
	} else {
		// Existing draft BOM found — merge items by deleting old items of submitted categories
		categorySet := make(map[string]bool)
		for _, pi := range items {
			if pi.Category != "" {
				categorySet[pi.Category] = true
			}
		}
		if len(categorySet) > 0 {
			categories := make([]string, 0, len(categorySet))
			for c := range categorySet {
				categories = append(categories, c)
			}
			if err := s.bomRepo.DeleteItemsByBOMAndCategories(ctx, bom.ID, categories); err != nil {
				fmt.Printf("[CreateBOMFromParsedItems] delete old items failed: %v\n", err)
			}
		}
		fmt.Printf("[CreateBOMFromParsedItems] merging into existing %s: id=%s\n", bomType, bom.ID)
	}

	// Get current max item_number for proper numbering
	maxNum, _ := s.bomRepo.GetMaxItemNumber(ctx, bom.ID)

	var entities []entity.ProjectBOMItem
	for i, pi := range items {
		itemNum := maxNum + i + 1
		if pi.ItemNumber > 0 {
			itemNum = pi.ItemNumber
		}

		categoryName := pi.Category
		categoryID := ""
		if categoryName == "" {
			categoryName, categoryID = inferCategoryFromReference(pi.Reference)
		} else {
			_, categoryID = inferCategoryFromReference(pi.Reference)
		}

		unit := pi.Unit
		if unit == "" {
			unit = "pcs"
		}

		item := entity.ProjectBOMItem{
			ID:           uuid.New().String()[:32],
			BOMID:        bom.ID,
			ItemNumber:   itemNum,
			Category:     categoryName,
			SubCategory:  pi.SubCategory,
			Name:         pi.Name,
			Quantity:     pi.Quantity,
			Unit:         unit,
			Supplier:     pi.Supplier,
			ThumbnailURL: pi.ThumbnailURL,
			Notes:        pi.Notes,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if pi.UnitPrice > 0 {
			up := pi.UnitPrice
			item.UnitPrice = &up
		}

		if pi.ExtendedAttrs != nil {
			item.ExtendedAttrs = entity.JSONB(pi.ExtendedAttrs)
		}
		// Copy API compat fields to extended_attrs
		if pi.Specification != "" {
			setExtAttr(&item.ExtendedAttrs, "specification", pi.Specification)
		}
		if pi.Reference != "" {
			setExtAttr(&item.ExtendedAttrs, "reference", pi.Reference)
		}
		if pi.Manufacturer != "" {
			setExtAttr(&item.ExtendedAttrs, "manufacturer", pi.Manufacturer)
		}
		if pi.ManufacturerPN != "" {
			setExtAttr(&item.ExtendedAttrs, "manufacturer_pn", pi.ManufacturerPN)
		}
		if pi.DrawingNo != "" {
			setExtAttr(&item.ExtendedAttrs, "drawing_no", pi.DrawingNo)
		}
		if pi.IsCritical {
			setExtAttr(&item.ExtendedAttrs, "is_critical", true)
		}
		if pi.LeadTimeDays > 0 {
			setExtAttr(&item.ExtendedAttrs, "lead_time_days", pi.LeadTimeDays)
		}
		if pi.IsAppearancePart {
			setExtAttr(&item.ExtendedAttrs, "is_appearance_part", true)
		}

		mat, matchErr := s.bomRepo.MatchMaterialByNameAndPN(ctx, pi.Name, pi.ManufacturerPN)
		if matchErr == nil && mat != nil {
			item.MaterialID = &mat.ID
		} else if pi.Name != "" {
			specification := pi.Specification
			if specification == "" {
				specification = pi.Name
			}
			cID := categoryID
			if cID == "" {
				cID = "mcat_el_oth"
			}
			newMat, createErr := s.autoCreateMaterial(ctx, pi.Name, specification, cID, pi.Manufacturer, pi.ManufacturerPN)
			if createErr != nil {
				fmt.Printf("[WARN] CreateBOMFromParsedItems: auto-create material failed for %q: %v\n", pi.Name, createErr)
			} else if newMat != nil {
				item.MaterialID = &newMat.ID
			}
		}

		entities = append(entities, item)
	}

	if len(entities) > 0 {
		if err := s.bomRepo.BatchCreateItems(ctx, entities); err != nil {
			return "", fmt.Errorf("batch create bom items: %w", err)
		}
		s.updateBOMCost(ctx, bom.ID)
	}

	return bom.ID, nil
}

func findUploadedFileURL(fileID, fileName string) string {
	savedName := fmt.Sprintf("%s_%s", fileID, fileName)
	matches, _ := filepath.Glob(fmt.Sprintf("./uploads/*/*/%s", savedName))
	if len(matches) > 0 {
		return strings.TrimPrefix(matches[0], ".")
	}
	now := time.Now()
	return fmt.Sprintf("/uploads/%d/%02d/%s", now.Year(), int(now.Month()), savedName)
}

func findUploadedFileSize(fileID, fileName string) int64 {
	savedName := fmt.Sprintf("%s_%s", fileID, fileName)
	matches, _ := filepath.Glob(fmt.Sprintf("./uploads/*/*/%s", savedName))
	if len(matches) > 0 {
		if fi, err := os.Stat(matches[0]); err == nil {
			return fi.Size()
		}
	}
	return 0
}

// ---- DTOs ----

type ImportResult struct {
	Success     int                `json:"created"`
	Failed      int                `json:"errors"`
	Matched     int                `json:"matched"`
	AutoCreated int                `json:"auto_created"`
	MPNMatched  int                `json:"mpn_matched"`  // MPN在已有BOM中匹配到
	MPNNew      int                `json:"mpn_new"`      // MPN不为空但未匹配
	MPNMissing  int                `json:"mpn_missing"`  // MPN为空
	Items       []ImportItemDetail `json:"items,omitempty"`
}

// ImportItemDetail 导入明细（每条物料的匹配状态）
type ImportItemDetail struct {
	Name           string `json:"name"`
	MPN            string `json:"mpn"`
	Reference      string `json:"reference"`
	Status         string `json:"status"` // matched / new / missing
	MatchedItemID  string `json:"matched_item_id,omitempty"`
}

type BOMCompareResult struct {
	BOM1      BOMSummary              `json:"bom1"`
	BOM2      BOMSummary              `json:"bom2"`
	Added     []entity.ProjectBOMItem `json:"added"`
	Removed   []entity.ProjectBOMItem `json:"removed"`
	Changed   []BOMItemDiff           `json:"changed"`
	Unchanged []entity.ProjectBOMItem `json:"unchanged"`
}

type BOMSummary struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	BOMType string `json:"bom_type"`
}

type BOMItemDiff struct {
	Key     string                `json:"key"`
	Item1   entity.ProjectBOMItem `json:"item1"`
	Item2   entity.ProjectBOMItem `json:"item2"`
	Changes []FieldChange         `json:"changes"`
}

type FieldChange struct {
	Field string `json:"field"`
	Old   string `json:"old"`
	New   string `json:"new"`
}

type CreateBOMInput struct {
	PhaseID     *string `json:"phase_id"`
	TaskID      *string `json:"task_id"`
	BOMType     string  `json:"bom_type" binding:"required"`
	Version     string  `json:"version"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
}

type UpdateBOMInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

type BOMItemInput struct {
	MaterialID       *string                `json:"material_id"`
	ParentItemID     *string                `json:"parent_item_id"`
	Level            int                    `json:"level"`
	Category         string                 `json:"category"`
	SubCategory      string                 `json:"sub_category"`
	Name             string                 `json:"name"`
	Specification    string                 `json:"specification"`     // kept for API compat, stored in extended_attrs
	Quantity         float64                `json:"quantity"`
	Unit             string                 `json:"unit"`
	Reference        string                 `json:"reference"`        // kept for API compat, stored in extended_attrs
	Manufacturer     string                 `json:"manufacturer"`     // kept for API compat, stored in extended_attrs
	ManufacturerPN   string                 `json:"manufacturer_pn"`  // kept for API compat, stored in extended_attrs
	Supplier         string                 `json:"supplier"`
	SupplierID       *string                `json:"supplier_id"`
	ManufacturerID   *string                `json:"manufacturer_id"`
	MPN              string                 `json:"mpn"`
	SupplierPN       string                 `json:"supplier_pn"`      // kept for API compat, stored in extended_attrs
	UnitPrice        *float64               `json:"unit_price"`
	LeadTimeDays     *int                   `json:"lead_time_days"`   // kept for API compat, stored in extended_attrs
	DrawingNo        string                 `json:"drawing_no"`       // kept for API compat, stored in extended_attrs
	IsCritical       bool                   `json:"is_critical"`      // kept for API compat, stored in extended_attrs
	IsAlternative    bool                   `json:"is_alternative"`
	IsAppearancePart bool                   `json:"is_appearance_part"`
	ThumbnailURL     string                 `json:"thumbnail_url"`
	Notes            string                 `json:"notes"`
	ItemNumber       int                    `json:"item_number"`
	ExtendedAttrs    map[string]interface{} `json:"extended_attrs"`
}

type ReorderItemsInput struct {
	ItemIDs []string `json:"item_ids" binding:"required"`
}

type TemplateInput struct {
	Category     string      `json:"category" binding:"required"`
	SubCategory  string      `json:"sub_category" binding:"required"`
	FieldKey     string      `json:"field_key" binding:"required"`
	FieldName    string      `json:"field_name" binding:"required"`
	FieldType    string      `json:"field_type" binding:"required"`
	Unit         string      `json:"unit"`
	Required     bool        `json:"required"`
	Options      map[string]interface{} `json:"options"`
	Validation   map[string]interface{} `json:"validation"`
	DefaultValue string      `json:"default_value"`
	SortOrder    int         `json:"sort_order"`
	ShowInTable  bool        `json:"show_in_table"`
}

type RouteInput struct {
	BOMID       string `json:"bom_id"`
	Name        string `json:"name" binding:"required"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

type StepInput struct {
	StepNumber     int      `json:"step_number"`
	Name           string   `json:"name" binding:"required"`
	WorkCenter     string   `json:"work_center"`
	Description    string   `json:"description"`
	StdTimeMinutes float64  `json:"std_time_minutes"`
	SetupMinutes   float64  `json:"setup_minutes"`
	LaborCost      *float64 `json:"labor_cost"`
	SortOrder      int      `json:"sort_order"`
}

type StepMaterialInput struct {
	MaterialID string  `json:"material_id"`
	Name       string  `json:"name" binding:"required"`
	Category   string  `json:"category"`
	Quantity   float64 `json:"quantity"`
	Unit       string  `json:"unit"`
	Notes      string  `json:"notes"`
}

// BOMPermissions 用户在项目中的BOM编辑权限
type BOMPermissions struct {
	CanEditCategories []string `json:"can_edit_categories"`
	CanViewAll        bool     `json:"can_view_all"`
	CanRelease        bool     `json:"can_release"`
}

// GetBOMPermissions 获取用户在项目中的BOM编辑权限
func (s *ProjectBOMService) GetBOMPermissions(ctx context.Context, projectID, userID string) (*BOMPermissions, error) {
	db := s.projectRepo.DB()

	// 1. Get project to check if user is creator (PM)
	var project entity.Project
	if err := db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", projectID).First(&project).Error; err != nil {
		return nil, fmt.Errorf("项目不存在")
	}

	// PM (project creator or manager) gets full permissions
	if project.CreatedBy == userID || project.ManagerID == userID {
		return &BOMPermissions{
			CanEditCategories: []string{"electronic", "structural", "optical", "packaging", "tooling", "consumable"},
			CanViewAll:        true,
			CanRelease:        true,
		}, nil
	}

	// 2. Check if user has admin role
	var adminCount int64
	db.WithContext(ctx).Table("user_roles").
		Joins("JOIN roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ? AND roles.code IN ?", userID, []string{"admin", "super_admin", "plm_admin"}).
		Count(&adminCount)
	if adminCount > 0 {
		return &BOMPermissions{
			CanEditCategories: []string{"electronic", "structural", "optical", "packaging", "tooling", "consumable"},
			CanViewAll:        true,
			CanRelease:        true,
		}, nil
	}

	// 3. Check user's assigned tasks in this project
	var tasks []entity.Task
	db.WithContext(ctx).
		Where("project_id = ? AND assignee_id = ?", projectID, userID).
		Find(&tasks)

	categorySet := map[string]bool{}
	for _, t := range tasks {
		titleLower := strings.ToLower(t.Title + " " + t.TaskType)
		if strings.Contains(titleLower, "电子") || strings.Contains(titleLower, "硬件") ||
			strings.Contains(titleLower, "electronic") || strings.Contains(titleLower, "hardware") {
			categorySet["electronic"] = true
		}
		if strings.Contains(titleLower, "结构") || strings.Contains(titleLower, "mechanical") ||
			strings.Contains(titleLower, "structural") {
			categorySet["structural"] = true
		}
		if strings.Contains(titleLower, "光学") || strings.Contains(titleLower, "optical") {
			categorySet["optical"] = true
		}
		if strings.Contains(titleLower, "包装") || strings.Contains(titleLower, "packaging") {
			categorySet["packaging"] = true
		}
		if strings.Contains(titleLower, "模具") || strings.Contains(titleLower, "tooling") {
			categorySet["tooling"] = true
		}
	}

	categories := make([]string, 0, len(categorySet))
	for cat := range categorySet {
		categories = append(categories, cat)
	}

	// If user has tasks but no category match, give view-only
	return &BOMPermissions{
		CanEditCategories: categories,
		CanViewAll:        true,
		CanRelease:        false,
	}, nil
}
