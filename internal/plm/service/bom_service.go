package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
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
}

func NewProjectBOMService(bomRepo *repository.ProjectBOMRepository, projectRepo *repository.ProjectRepository, deliverableRepo *repository.DeliverableRepository, materialRepo *repository.MaterialRepository) *ProjectBOMService {
	return &ProjectBOMService{
		bomRepo:         bomRepo,
		projectRepo:     projectRepo,
		deliverableRepo: deliverableRepo,
		materialRepo:    materialRepo,
	}
}

// CreateBOM 创建BOM（草稿状态）
func (s *ProjectBOMService) CreateBOM(ctx context.Context, projectID string, input *CreateBOMInput, createdBy string) (*entity.ProjectBOM, error) {
	bom := &entity.ProjectBOM{
		ID:          uuid.New().String()[:32],
		ProjectID:   projectID,
		PhaseID:     input.PhaseID,
		BOMType:     input.BOMType,
		Version:     input.Version,
		Name:        input.Name,
		Status:      "draft",
		Description: input.Description,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if bom.Version == "" {
		bom.Version = "v1.0"
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

	// 检查是否有行项
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

// FreezeBOM 冻结BOM（阶段门评审通过后）
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

	// Phase 4: 冻结时自动生成ERP发布快照
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
		ID:              uuid.New().String()[:32],
		BOMID:           bomID,
		ItemNumber:      input.ItemNumber,
		ParentItemID:    input.ParentItemID,
		Level:           input.Level,
		MaterialID:      input.MaterialID,
		Category:        input.Category,
		Name:            input.Name,
		Specification:   input.Specification,
		Quantity:        input.Quantity,
		Unit:            input.Unit,
		Reference:       input.Reference,
		Manufacturer:    input.Manufacturer,
		ManufacturerPN:  input.ManufacturerPN,
		Supplier:        input.Supplier,
		SupplierPN:      input.SupplierPN,
		UnitPrice:       input.UnitPrice,
		LeadTimeDays:    input.LeadTimeDays,
		ProcurementType: input.ProcurementType,
		MOQ:             input.MOQ,
		LifecycleStatus: input.LifecycleStatus,
		IsCritical:      input.IsCritical,
		IsAlternative:   input.IsAlternative,
		Notes:           input.Notes,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if item.Unit == "" {
		item.Unit = "pcs"
	}
	if item.ProcurementType == "" {
		item.ProcurementType = "buy"
	}
	if item.LifecycleStatus == "" {
		item.LifecycleStatus = "active"
	}

	// 计算小计
	if input.UnitPrice != nil {
		extCost := input.Quantity * *input.UnitPrice
		item.ExtendedCost = &extCost
	}

	if err := s.bomRepo.CreateItem(ctx, item); err != nil {
		return nil, fmt.Errorf("create bom item: %w", err)
	}

	// 更新BOM统计
	s.updateBOMCost(ctx, bomID)

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
			ID:             uuid.New().String()[:32],
			BOMID:          bomID,
			ItemNumber:     i + 1,
			Category:       input.Category,
			Name:           input.Name,
			Specification:  input.Specification,
			Quantity:        input.Quantity,
			Unit:           input.Unit,
			Reference:      input.Reference,
			Manufacturer:   input.Manufacturer,
			ManufacturerPN: input.ManufacturerPN,
			Supplier:       input.Supplier,
			UnitPrice:      input.UnitPrice,
			LeadTimeDays:   input.LeadTimeDays,
			IsCritical:     input.IsCritical,
			IsAlternative:  input.IsAlternative,
			Notes:          input.Notes,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		if item.Unit == "" {
			item.Unit = "pcs"
		}
		if input.MaterialID != nil {
			item.MaterialID = input.MaterialID
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

// UpdateItem 更新单个BOM行项
func (s *ProjectBOMService) UpdateItem(ctx context.Context, bomID, itemID string, input *BOMItemInput) (*entity.ProjectBOMItem, error) {
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

	if input.Name != "" {
		item.Name = input.Name
	}
	if input.Category != "" {
		item.Category = input.Category
	}
	if input.Specification != "" {
		item.Specification = input.Specification
	}
	if input.MaterialID != nil {
		item.MaterialID = input.MaterialID
	}
	item.Quantity = input.Quantity
	if input.Unit != "" {
		item.Unit = input.Unit
	}
	item.Reference = input.Reference
	item.Manufacturer = input.Manufacturer
	item.ManufacturerPN = input.ManufacturerPN
	item.Supplier = input.Supplier
	item.SupplierPN = input.SupplierPN
	item.UnitPrice = input.UnitPrice
	item.LeadTimeDays = input.LeadTimeDays
	item.IsCritical = input.IsCritical
	item.IsAlternative = input.IsAlternative
	item.Notes = input.Notes
	item.ProcurementType = input.ProcurementType
	item.MOQ = input.MOQ
	item.LifecycleStatus = input.LifecycleStatus
	item.ParentItemID = input.ParentItemID
	item.Level = input.Level

	// 计算小计
	if input.UnitPrice != nil {
		extCost := input.Quantity * *input.UnitPrice
		item.ExtendedCost = &extCost
	}

	item.UpdatedAt = time.Now()

	if err := s.bomRepo.UpdateItem(ctx, item); err != nil {
		return nil, fmt.Errorf("update item: %w", err)
	}

	// 更新BOM总成本
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

// ==================== Phase 2: Excel 导入/导出 ====================

var bomExportHeaders = []string{
	"序号", "分类", "名称", "规格", "数量", "单位", "位号",
	"制造商", "制造商料号", "供应商", "供应商料号", "单价", "小计",
	"是否关键", "备注",
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

	f := excelize.NewFile()
	sheet := "BOM"
	f.SetSheetName("Sheet1", sheet)

	// 表头样式: 加粗
	boldStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#D9E1F2"}},
		Border: []excelize.Border{
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})

	// 写入表头
	for i, h := range bomExportHeaders {
		col, _ := excelize.ColumnNumberToName(i + 1)
		cell := col + "1"
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, boldStyle)
	}

	// 写入数据行
	var totalCost float64
	for rowIdx, item := range items {
		row := rowIdx + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), item.ItemNumber)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), item.Category)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), item.Name)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), item.Specification)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), item.Quantity)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), item.Unit)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), item.Reference)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), item.Manufacturer)
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), item.ManufacturerPN)
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), item.Supplier)
		f.SetCellValue(sheet, fmt.Sprintf("K%d", row), item.SupplierPN)
		if item.UnitPrice != nil {
			f.SetCellValue(sheet, fmt.Sprintf("L%d", row), *item.UnitPrice)
		}
		if item.ExtendedCost != nil {
			f.SetCellValue(sheet, fmt.Sprintf("M%d", row), *item.ExtendedCost)
			totalCost += *item.ExtendedCost
		}
		critical := "否"
		if item.IsCritical {
			critical = "是"
		}
		f.SetCellValue(sheet, fmt.Sprintf("N%d", row), critical)
		f.SetCellValue(sheet, fmt.Sprintf("O%d", row), item.Notes)
	}

	// 底部汇总行
	summaryRow := len(items) + 2
	summaryStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
	})
	f.SetCellValue(sheet, fmt.Sprintf("A%d", summaryRow), "汇总")
	f.SetCellValue(sheet, fmt.Sprintf("C%d", summaryRow), fmt.Sprintf("总物料数: %d", len(items)))
	f.SetCellValue(sheet, fmt.Sprintf("M%d", summaryRow), totalCost)
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", summaryRow), fmt.Sprintf("O%d", summaryRow), summaryStyle)

	// 列宽自适应
	colWidths := []float64{6, 10, 20, 20, 8, 6, 14, 16, 16, 16, 16, 10, 10, 8, 20}
	for i, w := range colWidths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, w)
	}

	filename := fmt.Sprintf("BOM_%s_%s.xlsx", bom.Name, bom.Version)
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

	// 获取当前最大item_number
	existingCount, _ := s.bomRepo.CountItems(ctx, bomID)
	itemNum := int(existingCount)

	var entities []entity.ProjectBOMItem
	for i, row := range rows[1:] { // 跳过表头
		if len(row) < 3 || row[2] == "" { // 至少需要名称
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

		// 解析各列 (按模板顺序)
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
		if len(row) > 9 {
			item.Supplier = row[9]
		}
		if len(row) > 10 {
			item.SupplierPN = row[10]
		}
		if len(row) > 11 {
			if p, err := strconv.ParseFloat(row[11], 64); err == nil {
				item.UnitPrice = &p
				extCost := item.Quantity * p
				item.ExtendedCost = &extCost
			}
		}
		// 列12(小计)跳过，自动计算
		if len(row) > 13 && (row[13] == "是" || row[13] == "Y" || row[13] == "1") {
			item.IsCritical = true
		}
		if len(row) > 14 {
			item.Notes = row[14]
		}

		// 尝试匹配物料库
		mat, matchErr := s.bomRepo.MatchMaterialByNameAndPN(ctx, item.Name, item.ManufacturerPN)
		if matchErr == nil && mat != nil {
			item.MaterialID = &mat.ID
			result.Matched++
		} else {
			// 自动创建物料
			newMat, createErr := s.autoCreateMaterial(ctx, item.Name, item.Specification, item.Category, item.Manufacturer, item.ManufacturerPN)
			if createErr != nil {
				fmt.Printf("[WARN] auto-create material failed for %q: %v\n", item.Name, createErr)
			} else if newMat != nil {
				item.MaterialID = &newMat.ID
				result.AutoCreated++
			}
		}

		entities = append(entities, item)
		result.Success++

		_ = i // suppress unused
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

	// GBK → UTF-8
	utf8Reader := transform.NewReader(reader, simplifiedchinese.GBK.NewDecoder())

	result := &ImportResult{}

	// 获取当前最大item_number
	existingCount, _ := s.bomRepo.CountItems(ctx, bomID)
	itemNum := int(existingCount)

	var entities []entity.ProjectBOMItem
	scanner := bufio.NewScanner(utf8Reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimRight(scanner.Text(), "\r\n")
		if line == "" {
			continue
		}
		// 第一行是表头，跳过
		if lineNo == 1 {
			continue
		}

		fields := strings.Split(line, "\t")
		// 去除每个字段的双引号
		for i := range fields {
			fields[i] = strings.Trim(fields[i], "\"")
		}

		// 至少需要4列（序号、数量、参考编号、元件名称）
		if len(fields) < 4 || fields[3] == "" {
			result.Failed++
			continue
		}

		// 备注列（第8列，index 7）为 NC 的跳过
		if len(fields) > 7 && strings.EqualFold(strings.TrimSpace(fields[7]), "NC") {
			continue
		}

		itemNum++

		// 解析数量
		qty := 1.0
		if q, parseErr := strconv.ParseFloat(fields[1], 64); parseErr == nil {
			qty = q
		}

		// 参考编号
		reference := ""
		if len(fields) > 2 {
			reference = fields[2]
		}

		// 元件名称：逗号前的部分作为Name，完整作为Specification
		componentName := fields[3]
		name := componentName
		if idx := strings.Index(componentName, ","); idx > 0 {
			name = componentName[:idx]
		}

		// 制造商
		manufacturer := ""
		if len(fields) > 4 {
			manufacturer = fields[4]
		}

		// 说明 + 备注合并为Notes
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

		// Part Number
		manufacturerPN := ""
		if len(fields) > 6 {
			manufacturerPN = fields[6]
		}

		// 从参考编号前缀自动推断分类
		categoryName, categoryID := inferCategoryFromReference(reference)

		item := entity.ProjectBOMItem{
			ID:             uuid.New().String()[:32],
			BOMID:          bomID,
			ItemNumber:     itemNum,
			Category:       categoryName,
			Name:           name,
			Specification:  componentName,
			Quantity:       qty,
			Unit:           "pcs",
			Reference:      reference,
			Manufacturer:   manufacturer,
			ManufacturerPN: manufacturerPN,
			Notes:          notes,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// 尝试匹配物料库
		mat, matchErr := s.bomRepo.MatchMaterialByNameAndPN(ctx, item.Name, item.ManufacturerPN)
		if matchErr == nil && mat != nil {
			item.MaterialID = &mat.ID
			result.Matched++
		} else {
			// 自动创建物料
			newMat, createErr := s.autoCreateMaterial(ctx, item.Name, componentName, categoryID, manufacturer, manufacturerPN)
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

	if scanErr := scanner.Err(); scanErr != nil {
		return nil, fmt.Errorf("read rep file: %w", scanErr)
	}

	if len(entities) > 0 {
		if err := s.bomRepo.BatchCreateItems(ctx, entities); err != nil {
			return nil, fmt.Errorf("batch create: %w", err)
		}
		s.updateBOMCost(ctx, bomID)
	}

	return result, nil
}

// inferCategoryFromReference 从参考编号前缀推断物料分类
// 返回 (显示名称, 二级分类ID)
func inferCategoryFromReference(reference string) (string, string) {
	if reference == "" {
		return "", "mcat_el_oth"
	}
	// 取第一个位号（空格分隔）
	first := strings.Fields(reference)[0]
	// 去除尾部数字和'-'，提取字母前缀
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
func (s *ProjectBOMService) GenerateTemplate() (*excelize.File, error) {
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

	// 列宽
	colWidths := []float64{6, 10, 20, 20, 8, 6, 14, 16, 16, 16, 16, 10, 10, 8, 20}
	for i, w := range colWidths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, w)
	}

	// 数据验证说明sheet
	helpSheet := "填写说明"
	f.NewSheet(helpSheet)
	helpData := [][]string{
		{"列名", "说明", "是否必填"},
		{"序号", "自动编号，可留空", "否"},
		{"分类", "物料分类，如: 电阻/电容/IC/结构件", "否"},
		{"名称", "物料名称", "是"},
		{"规格", "规格型号描述", "否"},
		{"数量", "用量数字，默认1", "是"},
		{"单位", "pcs/kg/m/set，默认pcs", "否"},
		{"位号", "PCB位号，如R1,R2,R3", "否"},
		{"制造商", "制造商名称", "否"},
		{"制造商料号", "制造商Part Number，用于自动匹配物料库", "否"},
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

	// 示例数据行
	sampleData := []string{"1", "电阻", "100K电阻 0402", "100KΩ ±1% 0402", "10", "pcs", "R1-R10", "Yageo", "RC0402FR-07100KL", "DigiKey", "311-100KLRCT-ND", "0.05", "", "否", ""}
	for j, val := range sampleData {
		col, _ := excelize.ColumnNumberToName(j + 1)
		f.SetCellValue(sheet, fmt.Sprintf("%s2", col), val)
	}

	return f, nil
}

// ==================== Parse-only (preview, no save) ====================

// ParsePADSBOM 解析PADS BOM (.rep) 返回预览数据，不保存
func (s *ProjectBOMService) ParsePADSBOM(ctx context.Context, reader io.Reader) ([]ParsedBOMItem, error) {
	// GBK → UTF-8
	utf8Reader := transform.NewReader(reader, simplifiedchinese.GBK.NewDecoder())

	var items []ParsedBOMItem
	scanner := bufio.NewScanner(utf8Reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	itemNum := 0
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
			continue
		}

		// 备注列为 NC 的跳过
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

// ParseExcelBOM 解析Excel BOM返回预览数据，不保存
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
		item := ParsedBOMItem{
			ItemNumber: itemNum,
			Unit:       "pcs",
		}

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

// ==================== Phase 3: EBOM→MBOM转换 + 版本对比 ====================

// ConvertToMBOM 将EBOM一键转换为MBOM
func (s *ProjectBOMService) ConvertToMBOM(ctx context.Context, bomID, createdBy string) (*entity.ProjectBOM, error) {
	srcBOM, err := s.bomRepo.FindByID(ctx, bomID)
	if err != nil {
		return nil, fmt.Errorf("source bom not found: %w", err)
	}

	// 创建新MBOM
	newBOM := &entity.ProjectBOM{
		ID:          uuid.New().String()[:32],
		ProjectID:   srcBOM.ProjectID,
		PhaseID:     srcBOM.PhaseID,
		BOMType:     "MBOM",
		Version:     "v1.0",
		Name:        srcBOM.Name + " (MBOM)",
		Status:      "draft",
		Description: fmt.Sprintf("从 %s %s 转换而来", srcBOM.Name, srcBOM.Version),
		ParentBOMID: &srcBOM.ID,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.bomRepo.Create(ctx, newBOM); err != nil {
		return nil, fmt.Errorf("create mbom: %w", err)
	}

	// 深拷贝所有行项
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
			newItem.ParentItemID = nil // 清除父项引用，需要用户重新组织
			newItem.CreatedAt = time.Now()
			newItem.UpdatedAt = time.Now()
			newItems = append(newItems, newItem)
		}

		if err := s.bomRepo.BatchCreateItems(ctx, newItems); err != nil {
			return nil, fmt.Errorf("copy items: %w", err)
		}

		newBOM.TotalItems = len(newItems)
		newBOM.EstimatedCost = srcBOM.EstimatedCost
		s.bomRepo.Update(ctx, newBOM)
	}

	// 重新加载完整数据返回
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

	items1, err := s.bomRepo.ListItemsByBOM(ctx, bom1ID)
	if err != nil {
		return nil, fmt.Errorf("list bom1 items: %w", err)
	}
	items2, err := s.bomRepo.ListItemsByBOM(ctx, bom2ID)
	if err != nil {
		return nil, fmt.Errorf("list bom2 items: %w", err)
	}

	// 用 名称+制造商料号 作为匹配key
	makeKey := func(item entity.ProjectBOMItem) string {
		return item.Name + "|" + item.ManufacturerPN
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

	// 遍历bom1的项
	for key, item1 := range map1 {
		if item2, exists := map2[key]; exists {
			// 两边都有，检查变更
			changes := compareItemFields(item1, item2)
			if len(changes) > 0 {
				result.Changed = append(result.Changed, BOMItemDiff{
					Key:     key,
					Item1:   item1,
					Item2:   item2,
					Changes: changes,
				})
			} else {
				result.Unchanged = append(result.Unchanged, item1)
			}
		} else {
			// bom1有bom2没有 → removed
			result.Removed = append(result.Removed, item1)
		}
	}

	// 遍历bom2，找出bom1没有的 → added
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
	if a.SupplierPN != b.SupplierPN {
		changes = append(changes, FieldChange{Field: "supplier_pn", Old: a.SupplierPN, New: b.SupplierPN})
	}
	if a.Specification != b.Specification {
		changes = append(changes, FieldChange{Field: "specification", Old: a.Specification, New: b.Specification})
	}
	if a.Unit != b.Unit {
		changes = append(changes, FieldChange{Field: "unit", Old: a.Unit, New: b.Unit})
	}
	if a.Reference != b.Reference {
		changes = append(changes, FieldChange{Field: "reference", Old: a.Reference, New: b.Reference})
	}
	if a.IsCritical != b.IsCritical {
		changes = append(changes, FieldChange{Field: "is_critical", Old: fmt.Sprintf("%v", a.IsCritical), New: fmt.Sprintf("%v", b.IsCritical)})
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

// ==================== Phase 4: ERP对接桥梁 ====================

// CreateBOMRelease 创建BOM发布快照（冻结时自动调用）
func (s *ProjectBOMService) CreateBOMRelease(ctx context.Context, bom *entity.ProjectBOM) (*entity.BOMRelease, error) {
	items, err := s.bomRepo.ListItemsByBOM(ctx, bom.ID)
	if err != nil {
		return nil, fmt.Errorf("list items for snapshot: %w", err)
	}

	snapshot := map[string]interface{}{
		"bom":   bom,
		"items": items,
	}
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

// ListPendingReleases 获取待同步的BOM发布快照
func (s *ProjectBOMService) ListPendingReleases(ctx context.Context) ([]entity.BOMRelease, error) {
	return s.bomRepo.ListPendingReleases(ctx)
}

// AckRelease ERP确认接收
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

// ==================== 自动建料 ====================

// autoCreateMaterial 根据BOM行信息自动创建物料
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

// mapCategoryToIDAndCode 将分类标识映射到二级分类 categoryID 和编码code
// 支持传入二级分类ID（mcat_el_res格式）或中文分类名
func mapCategoryToIDAndCode(category string) (string, string) {
	// 如果传入的已经是二级分类ID（mcat_xx_xxx格式），直接映射
	idToCode := map[string]string{
		"mcat_el_res": "EL-RES",
		"mcat_el_cap": "EL-CAP",
		"mcat_el_ind": "EL-IND",
		"mcat_el_ic":  "EL-IC",
		"mcat_el_con": "EL-CON",
		"mcat_el_dio": "EL-DIO",
		"mcat_el_trn": "EL-TRN",
		"mcat_el_osc": "EL-OSC",
		"mcat_el_led": "EL-LED",
		"mcat_el_sen": "EL-SEN",
		"mcat_el_ant": "EL-ANT",
		"mcat_el_mod": "EL-MOD",
		"mcat_el_bat": "EL-BAT",
		"mcat_el_pcb": "EL-PCB",
		"mcat_el_oth": "EL-OTH",
	}
	if code, ok := idToCode[category]; ok {
		return category, code
	}

	// 兼容中文分类名（Excel导入场景）
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
	default:
		return "mcat_el_oth", "EL-OTH"
	}
}

// ---- ParsedBOMItem for parse-only preview ----

type ParsedBOMItem struct {
	ItemNumber     int     `json:"item_number"`
	Reference      string  `json:"reference"`
	Name           string  `json:"name"`
	Specification  string  `json:"specification"`
	Quantity       float64 `json:"quantity"`
	Unit           string  `json:"unit"`
	Category       string  `json:"category"`
	Manufacturer   string  `json:"manufacturer"`
	ManufacturerPN string  `json:"manufacturer_pn"`
}

// CreateBOMFromParsedItems 根据已解析的BOM条目创建项目BOM（含自动建料）
// 用于任务表单中 bom_upload 字段的自动BOM创建
// 返回创建的BOM ID和错误
func (s *ProjectBOMService) CreateBOMFromParsedItems(ctx context.Context, projectID, userID string, bomName string, items []ParsedBOMItem) (string, error) {
	// 1. 创建 ProjectBOM 记录
	bom := &entity.ProjectBOM{
		ID:        uuid.New().String()[:32],
		ProjectID: projectID,
		BOMType:   "EBOM",
		Version:   "v1.0",
		Name:      bomName,
		Status:    "draft",
		CreatedBy: userID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.bomRepo.Create(ctx, bom); err != nil {
		return "", fmt.Errorf("create project bom: %w", err)
	}

	// 2. 为每个条目创建 BOM Item + 自动建料
	var entities []entity.ProjectBOMItem
	for i, pi := range items {
		itemNum := i + 1
		if pi.ItemNumber > 0 {
			itemNum = pi.ItemNumber
		}

		// 从参考编号推断分类（如果前端未提供）
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
			ID:             uuid.New().String()[:32],
			BOMID:          bom.ID,
			ItemNumber:     itemNum,
			Category:       categoryName,
			Name:           pi.Name,
			Specification:  pi.Specification,
			Quantity:       pi.Quantity,
			Unit:           unit,
			Reference:      pi.Reference,
			Manufacturer:   pi.Manufacturer,
			ManufacturerPN: pi.ManufacturerPN,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// 尝试匹配已有物料，未找到则自动创建
		mat, matchErr := s.bomRepo.MatchMaterialByNameAndPN(ctx, pi.Name, pi.ManufacturerPN)
		if matchErr == nil && mat != nil {
			item.MaterialID = &mat.ID
		} else if pi.Name != "" {
			specification := pi.Specification
			if specification == "" {
				specification = pi.Name
			}
			newMat, createErr := s.autoCreateMaterial(ctx, pi.Name, specification, categoryID, pi.Manufacturer, pi.ManufacturerPN)
			if createErr != nil {
				fmt.Printf("[WARN] CreateBOMFromParsedItems: auto-create material failed for %q: %v\n", pi.Name, createErr)
			} else if newMat != nil {
				item.MaterialID = &newMat.ID
			}
		}

		entities = append(entities, item)
	}

	// 3. 批量插入BOM行项
	if len(entities) > 0 {
		if err := s.bomRepo.BatchCreateItems(ctx, entities); err != nil {
			return "", fmt.Errorf("batch create bom items: %w", err)
		}
		// 更新BOM总数和成本
		s.updateBOMCost(ctx, bom.ID)
	}

	return bom.ID, nil
}

// ---- Phase 2/3/4 DTOs ----

type ImportResult struct {
	Success     int `json:"created"`
	Failed      int `json:"errors"`
	Matched     int `json:"matched"`
	AutoCreated int `json:"auto_created"`
}

type BOMCompareResult struct {
	BOM1      BOMSummary             `json:"bom1"`
	BOM2      BOMSummary             `json:"bom2"`
	Added     []entity.ProjectBOMItem `json:"added"`
	Removed   []entity.ProjectBOMItem `json:"removed"`
	Changed   []BOMItemDiff          `json:"changed"`
	Unchanged []entity.ProjectBOMItem `json:"unchanged"`
}

type BOMSummary struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	BOMType string `json:"bom_type"`
}

type BOMItemDiff struct {
	Key     string                 `json:"key"`
	Item1   entity.ProjectBOMItem  `json:"item1"`
	Item2   entity.ProjectBOMItem  `json:"item2"`
	Changes []FieldChange          `json:"changes"`
}

type FieldChange struct {
	Field string `json:"field"`
	Old   string `json:"old"`
	New   string `json:"new"`
}

// ---- Input DTOs ----

type CreateBOMInput struct {
	PhaseID     *string `json:"phase_id"`
	BOMType     string  `json:"bom_type" binding:"required"`
	Version     string  `json:"version"`
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
}

type UpdateBOMInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

type BOMItemInput struct {
	MaterialID      *string  `json:"material_id"`
	ParentItemID    *string  `json:"parent_item_id"`
	Level           int      `json:"level"`
	Category        string   `json:"category"`
	Name            string   `json:"name" binding:"required"`
	Specification   string   `json:"specification"`
	Quantity        float64  `json:"quantity"`
	Unit            string   `json:"unit"`
	Reference       string   `json:"reference"`
	Manufacturer    string   `json:"manufacturer"`
	ManufacturerPN  string   `json:"manufacturer_pn"`
	Supplier        string   `json:"supplier"`
	SupplierPN      string   `json:"supplier_pn"`
	UnitPrice       *float64 `json:"unit_price"`
	LeadTimeDays    *int     `json:"lead_time_days"`
	ProcurementType string   `json:"procurement_type"`
	MOQ             *int     `json:"moq"`
	LifecycleStatus string   `json:"lifecycle_status"`
	IsCritical      bool     `json:"is_critical"`
	IsAlternative   bool     `json:"is_alternative"`
	Notes           string   `json:"notes"`
	ItemNumber      int      `json:"item_number"`
}

type ReorderItemsInput struct {
	ItemIDs []string `json:"item_ids" binding:"required"`
}
