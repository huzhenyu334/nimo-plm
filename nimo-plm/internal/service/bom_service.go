package service

import (
	"context"
	"fmt"
	"time"

	"github.com/bitfantasy/nimo-plm/internal/model/entity"
	"github.com/bitfantasy/nimo-plm/internal/repository"
	"github.com/google/uuid"
)

// AddBOMItemRequest 添加BOM行项请求
type AddBOMItemRequest struct {
	ParentItemID string  `json:"parent_item_id"`
	MaterialID   string  `json:"material_id" binding:"required"`
	Quantity     float64 `json:"quantity" binding:"required,min=0.001"`
	Unit         string  `json:"unit"`
	Position     string  `json:"position"`
	Notes        string  `json:"notes"`
}

// UpdateBOMItemRequest 更新BOM行项请求
type UpdateBOMItemRequest struct {
	Quantity float64 `json:"quantity"`
	Position string  `json:"position"`
	Notes    string  `json:"notes"`
}

// ReleaseBOMRequest 发布BOM请求
type ReleaseBOMRequest struct {
	Version      string `json:"version" binding:"required"`
	ReleaseNotes string `json:"release_notes"`
}

// BOMCompareResult BOM对比结果
type BOMCompareResult struct {
	VersionA string            `json:"version_a"`
	VersionB string            `json:"version_b"`
	Added    []entity.BOMItem  `json:"added"`
	Removed  []entity.BOMItem  `json:"removed"`
	Modified []entity.BOMItem  `json:"modified"`
}

// GetBOM 获取产品BOM
func (s *BOMService) GetBOM(ctx context.Context, productID, version string, expandLevel int, includeCost bool) (*entity.BOMHeader, error) {
	bom, err := s.bomRepo.GetByProductID(ctx, productID, version)
	if err != nil {
		if err == repository.ErrNotFound {
			// 如果没有BOM，创建一个草稿
			return nil, fmt.Errorf("BOM not found")
		}
		return nil, fmt.Errorf("get BOM: %w", err)
	}

	// 构建层级结构
	bom.Items = s.buildBOMTree(bom.Items, expandLevel)

	// 如果需要成本，计算成本
	if includeCost {
		s.calculateBOMCost(bom)
	}

	return bom, nil
}

// buildBOMTree 构建BOM树结构
func (s *BOMService) buildBOMTree(items []entity.BOMItem, maxLevel int) []entity.BOMItem {
	// 建立父子关系映射
	itemMap := make(map[string]*entity.BOMItem)
	for i := range items {
		itemMap[items[i].ID] = &items[i]
	}

	// 构建树
	var roots []entity.BOMItem
	for i := range items {
		item := &items[i]
		if item.ParentItemID == "" {
			// 顶级项
			if maxLevel < 0 || item.Level <= maxLevel {
				s.attachChildren(item, itemMap, maxLevel)
				roots = append(roots, *item)
			}
		}
	}

	return roots
}

// attachChildren 递归附加子项
func (s *BOMService) attachChildren(item *entity.BOMItem, itemMap map[string]*entity.BOMItem, maxLevel int) {
	if maxLevel >= 0 && item.Level >= maxLevel {
		return
	}

	for _, child := range itemMap {
		if child.ParentItemID == item.ID {
			s.attachChildren(child, itemMap, maxLevel)
			item.Children = append(item.Children, *child)
		}
	}
}

// calculateBOMCost 计算BOM成本
func (s *BOMService) calculateBOMCost(bom *entity.BOMHeader) {
	var totalCost float64
	for i := range bom.Items {
		item := &bom.Items[i]
		if item.Material != nil {
			item.UnitCost = item.Material.StandardCost
			item.ExtendedCost = item.UnitCost * item.Quantity
			if item.Level == 0 {
				totalCost += item.ExtendedCost
			}
		}
		// 递归计算子项
		s.calculateItemCost(item)
	}
	bom.TotalCost = totalCost
}

// calculateItemCost 递归计算行项成本
func (s *BOMService) calculateItemCost(item *entity.BOMItem) {
	for i := range item.Children {
		child := &item.Children[i]
		if child.Material != nil {
			child.UnitCost = child.Material.StandardCost
			child.ExtendedCost = child.UnitCost * child.Quantity
		}
		s.calculateItemCost(child)
	}
}

// ListBOMVersions 获取BOM版本列表
func (s *BOMService) ListVersions(ctx context.Context, productID string) ([]entity.BOMHeader, error) {
	return s.bomRepo.ListVersions(ctx, productID)
}

// AddBOMItem 添加BOM行项
func (s *BOMService) AddItem(ctx context.Context, productID, userID string, req *AddBOMItemRequest) (*entity.BOMItem, error) {
	// 获取或创建草稿BOM
	bom, err := s.bomRepo.GetDraftBOM(ctx, productID)
	if err != nil {
		if err == repository.ErrNotFound {
			bom, err = s.bomRepo.CreateDraftBOM(ctx, productID, userID)
			if err != nil {
				return nil, fmt.Errorf("create draft BOM: %w", err)
			}
		} else {
			return nil, fmt.Errorf("get draft BOM: %w", err)
		}
	}

	// 检查BOM状态
	if bom.Status != entity.BOMStatusDraft {
		return nil, fmt.Errorf("can only modify draft BOM")
	}

	// 获取物料信息
	material, err := s.materialRepo.FindByID(ctx, req.MaterialID)
	if err != nil {
		return nil, fmt.Errorf("material not found: %w", err)
	}

	// 确定层级
	level := 0
	if req.ParentItemID != "" {
		parentItem, err := s.bomRepo.GetItem(ctx, req.ParentItemID)
		if err != nil {
			return nil, fmt.Errorf("parent item not found: %w", err)
		}
		level = parentItem.Level + 1
	}

	// 获取序号
	sequence, err := s.bomRepo.GetMaxSequence(ctx, bom.ID, req.ParentItemID)
	if err != nil {
		return nil, fmt.Errorf("get max sequence: %w", err)
	}

	// 创建BOM行项
	unit := req.Unit
	if unit == "" {
		unit = material.Unit
	}

	now := time.Now()
	item := &entity.BOMItem{
		ID:           uuid.New().String()[:32],
		BOMHeaderID:  bom.ID,
		ParentItemID: req.ParentItemID,
		MaterialID:   req.MaterialID,
		Level:        level,
		Sequence:     sequence + 1,
		Quantity:     req.Quantity,
		Unit:         unit,
		Position:     req.Position,
		Notes:        req.Notes,
		UnitCost:     material.StandardCost,
		ExtendedCost: material.StandardCost * req.Quantity,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.bomRepo.CreateItem(ctx, item); err != nil {
		return nil, fmt.Errorf("create BOM item: %w", err)
	}

	// 更新BOM统计
	if err := s.bomRepo.UpdateStats(ctx, bom.ID); err != nil {
		// 不阻断流程
	}

	// 加载物料信息
	item.Material = material

	return item, nil
}

// UpdateBOMItem 更新BOM行项
func (s *BOMService) UpdateItem(ctx context.Context, productID, itemID, userID string, req *UpdateBOMItemRequest) (*entity.BOMItem, error) {
	// 获取行项
	item, err := s.bomRepo.GetItem(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("item not found: %w", err)
	}

	// 获取BOM头
	bom, err := s.bomRepo.GetByID(ctx, item.BOMHeaderID)
	if err != nil {
		return nil, fmt.Errorf("BOM not found: %w", err)
	}

	// 检查状态
	if bom.Status != entity.BOMStatusDraft {
		return nil, fmt.Errorf("can only modify draft BOM")
	}

	// 更新字段
	if req.Quantity > 0 {
		item.Quantity = req.Quantity
		item.ExtendedCost = item.UnitCost * req.Quantity
	}
	if req.Position != "" {
		item.Position = req.Position
	}
	if req.Notes != "" {
		item.Notes = req.Notes
	}
	item.UpdatedAt = time.Now()

	if err := s.bomRepo.UpdateItem(ctx, item); err != nil {
		return nil, fmt.Errorf("update item: %w", err)
	}

	// 更新BOM统计
	s.bomRepo.UpdateStats(ctx, bom.ID)

	return item, nil
}

// DeleteBOMItem 删除BOM行项
func (s *BOMService) DeleteItem(ctx context.Context, productID, itemID, userID string) error {
	// 获取行项
	item, err := s.bomRepo.GetItem(ctx, itemID)
	if err != nil {
		return fmt.Errorf("item not found: %w", err)
	}

	// 获取BOM头
	bom, err := s.bomRepo.GetByID(ctx, item.BOMHeaderID)
	if err != nil {
		return fmt.Errorf("BOM not found: %w", err)
	}

	// 检查状态
	if bom.Status != entity.BOMStatusDraft {
		return fmt.Errorf("can only modify draft BOM")
	}

	// 删除行项（包括子项）
	if err := s.bomRepo.DeleteItem(ctx, itemID); err != nil {
		return fmt.Errorf("delete item: %w", err)
	}

	// 更新BOM统计
	s.bomRepo.UpdateStats(ctx, bom.ID)

	return nil
}

// ReleaseBOM 发布BOM
func (s *BOMService) Release(ctx context.Context, productID, userID string, req *ReleaseBOMRequest) (*entity.BOMHeader, error) {
	// 获取草稿BOM
	bom, err := s.bomRepo.GetDraftBOM(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("no draft BOM found: %w", err)
	}

	// 检查是否有物料
	if bom.TotalItems == 0 {
		return nil, fmt.Errorf("BOM has no items")
	}

	// 更新状态
	now := time.Now()
	bom.Status = entity.BOMStatusReleased
	bom.Version = req.Version
	bom.ReleasedBy = userID
	bom.ReleasedAt = &now
	bom.ReleaseNotes = req.ReleaseNotes
	bom.UpdatedAt = now

	if err := s.bomRepo.Update(ctx, bom); err != nil {
		return nil, fmt.Errorf("update BOM: %w", err)
	}

	// 更新产品的当前BOM版本
	// TODO: 调用ProductRepository更新

	return bom, nil
}

// CompareBOM 对比BOM版本
func (s *BOMService) Compare(ctx context.Context, productID, versionA, versionB string) (*BOMCompareResult, error) {
	added, removed, modified, err := s.bomRepo.Compare(ctx, productID, versionA, versionB)
	if err != nil {
		return nil, fmt.Errorf("compare BOM: %w", err)
	}

	return &BOMCompareResult{
		VersionA: versionA,
		VersionB: versionB,
		Added:    added,
		Removed:  removed,
		Modified: modified,
	}, nil
}
