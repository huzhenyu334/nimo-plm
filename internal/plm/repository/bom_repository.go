package repository

import (
	"context"
	"github.com/bitfantasy/nimo/internal/plm/entity"

	"gorm.io/gorm"
)

type ProjectBOMRepository struct {
	db *gorm.DB
}

func NewProjectBOMRepository(db *gorm.DB) *ProjectBOMRepository {
	return &ProjectBOMRepository{db: db}
}

func (r *ProjectBOMRepository) DB() *gorm.DB {
	return r.db
}

// Create 创建BOM
func (r *ProjectBOMRepository) Create(ctx context.Context, bom *entity.ProjectBOM) error {
	return r.db.WithContext(ctx).Create(bom).Error
}

// FindByID 根据ID查找BOM
func (r *ProjectBOMRepository) FindByID(ctx context.Context, id string) (*entity.ProjectBOM, error) {
	var bom entity.ProjectBOM
	err := r.db.WithContext(ctx).
		Preload("Phase").
		Preload("Items").
		Preload("Items.Material").
		Preload("Submitter").
		Preload("Reviewer").
		Preload("Creator").
		First(&bom, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &bom, nil
}

// ListByProject 获取项目的BOM列表
func (r *ProjectBOMRepository) ListByProject(ctx context.Context, projectID string, bomType, status string) ([]entity.ProjectBOM, error) {
	var boms []entity.ProjectBOM
	query := r.db.WithContext(ctx).
		Preload("Phase").
		Preload("Creator").
		Preload("Submitter").
		Preload("Reviewer").
		Where("project_id = ?", projectID)

	if bomType != "" {
		query = query.Where("bom_type = ?", bomType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	err := query.Order("created_at DESC").Find(&boms).Error
	return boms, err
}

// Update 更新BOM
func (r *ProjectBOMRepository) Update(ctx context.Context, bom *entity.ProjectBOM) error {
	return r.db.WithContext(ctx).Save(bom).Error
}

// Delete 删除BOM
func (r *ProjectBOMRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.ProjectBOM{}, "id = ?", id).Error
}

// CreateItem 创建BOM行项
func (r *ProjectBOMRepository) CreateItem(ctx context.Context, item *entity.ProjectBOMItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// FindItemByID 根据ID查找BOM行项
func (r *ProjectBOMRepository) FindItemByID(ctx context.Context, id string) (*entity.ProjectBOMItem, error) {
	var item entity.ProjectBOMItem
	err := r.db.WithContext(ctx).Preload("Material").First(&item, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// UpdateItem 更新BOM行项
func (r *ProjectBOMRepository) UpdateItem(ctx context.Context, item *entity.ProjectBOMItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}

// UpdateItemField 更新BOM行项的单个字段
func (r *ProjectBOMRepository) UpdateItemField(ctx context.Context, itemID string, field string, value interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.ProjectBOMItem{}).Where("id = ?", itemID).Update(field, value).Error
}

// DeleteItem 删除BOM行项
func (r *ProjectBOMRepository) DeleteItem(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.ProjectBOMItem{}, "id = ?", id).Error
}

// DeleteItemsByBOM 删除BOM所有行项
func (r *ProjectBOMRepository) DeleteItemsByBOM(ctx context.Context, bomID string) error {
	return r.db.WithContext(ctx).Delete(&entity.ProjectBOMItem{}, "bom_id = ?", bomID).Error
}

// DeleteItemsByBOMAndCategories 删除BOM中指定categories的行项
func (r *ProjectBOMRepository) DeleteItemsByBOMAndCategories(ctx context.Context, bomID string, categories []string) error {
	return r.db.WithContext(ctx).Delete(&entity.ProjectBOMItem{}, "bom_id = ? AND category IN ?", bomID, categories).Error
}

// CountItems 统计BOM行项数
func (r *ProjectBOMRepository) CountItems(ctx context.Context, bomID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.ProjectBOMItem{}).Where("bom_id = ?", bomID).Count(&count).Error
	return count, err
}

// GetMaxItemNumber 获取BOM的最大item_number
func (r *ProjectBOMRepository) GetMaxItemNumber(ctx context.Context, bomID string) (int, error) {
	var maxNum *int
	err := r.db.WithContext(ctx).Model(&entity.ProjectBOMItem{}).Where("bom_id = ?", bomID).Select("MAX(item_number)").Scan(&maxNum).Error
	if err != nil {
		return 0, err
	}
	if maxNum == nil {
		return 0, nil
	}
	return *maxNum, nil
}

// FindDerivedItemByVariant 查找CMF变体对应的衍生零件
func (r *ProjectBOMRepository) FindDerivedItemByVariant(ctx context.Context, parentItemID, variantID string) (*entity.ProjectBOMItem, error) {
	var item entity.ProjectBOMItem
	err := r.db.WithContext(ctx).
		Where("parent_item_id = ? AND notes LIKE ?", parentItemID, "%cmf_variant_id:"+variantID+"%").
		First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// BatchCreateItems 批量创建BOM行项
func (r *ProjectBOMRepository) BatchCreateItems(ctx context.Context, items []entity.ProjectBOMItem) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&items).Error
}

// ListItemsByBOM 获取BOM的所有行项（按item_number排序）
func (r *ProjectBOMRepository) ListItemsByBOM(ctx context.Context, bomID string) ([]entity.ProjectBOMItem, error) {
	var items []entity.ProjectBOMItem
	err := r.db.WithContext(ctx).
		Where("bom_id = ?", bomID).
		Order("item_number ASC").
		Find(&items).Error
	return items, err
}

// MatchMaterialByNameAndPN 通过名称+制造商料号匹配物料库
func (r *ProjectBOMRepository) MatchMaterialByNameAndPN(ctx context.Context, name, manufacturerPN string) (*entity.Material, error) {
	var material entity.Material
	query := r.db.WithContext(ctx).Where("deleted_at IS NULL")
	if manufacturerPN != "" {
		query = query.Where("name ILIKE ? OR code ILIKE ?", "%"+name+"%", "%"+manufacturerPN+"%")
	} else {
		query = query.Where("name ILIKE ?", "%"+name+"%")
	}
	err := query.First(&material).Error
	if err != nil {
		return nil, err
	}
	return &material, nil
}

// SearchItems 跨项目搜索BOM行项（按name/mpn/material_code模糊匹配）
func (r *ProjectBOMRepository) SearchItems(ctx context.Context, keyword string, category string, limit int) ([]entity.ProjectBOMItem, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	var items []entity.ProjectBOMItem
	query := r.db.WithContext(ctx).Model(&entity.ProjectBOMItem{})
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where(
			"name ILIKE ? OR mpn ILIKE ? OR extended_attrs->>'manufacturer_pn' ILIKE ? OR extended_attrs->>'specification' ILIKE ?",
			like, like, like, like,
		)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}
	err := query.Order("updated_at DESC").Limit(limit).Find(&items).Error
	return items, err
}

// MaterialSearchResult wraps a BOM item with its parent BOM's project/type info
type MaterialSearchResult struct {
	entity.ProjectBOMItem
	ProjectID        string  `json:"project_id" gorm:"column:project_id"`
	BOMType          string  `json:"bom_type" gorm:"column:bom_type"`
	BOMName          string  `json:"bom_name" gorm:"column:bom_name"`
	ProjectName      string  `json:"project_name" gorm:"column:project_name"`
	SupplierName     *string `json:"supplier_name" gorm:"column:supplier_name"`
	ManufacturerName *string `json:"manufacturer_name" gorm:"column:manufacturer_name"`
}

// GlobalSearchParams 全局物料搜索参数
type GlobalSearchParams struct {
	Keyword        string
	Category       string
	SubCategory    string
	BOMID          string
	ProjectID      string
	SupplierID     string
	ManufacturerID string
	Page           int
	PageSize       int
}

// SearchItemsPaginated 跨项目搜索BOM行项（分页版，用于全局物料查询页面）
func (r *ProjectBOMRepository) SearchItemsPaginated(ctx context.Context, keyword, category, subCategory, bomID string, page, pageSize int) ([]MaterialSearchResult, int64, error) {
	return r.GlobalSearchItems(ctx, GlobalSearchParams{
		Keyword:     keyword,
		Category:    category,
		SubCategory: subCategory,
		BOMID:       bomID,
		Page:        page,
		PageSize:    pageSize,
	})
}

// GlobalSearchItems 全局物料搜索（支持project/supplier/manufacturer筛选）
func (r *ProjectBOMRepository) GlobalSearchItems(ctx context.Context, params GlobalSearchParams) ([]MaterialSearchResult, int64, error) {
	if params.PageSize <= 0 || params.PageSize > 100 {
		params.PageSize = 20
	}
	if params.Page <= 0 {
		params.Page = 1
	}
	var total int64
	query := r.db.WithContext(ctx).Model(&entity.ProjectBOMItem{}).
		Joins("LEFT JOIN project_boms ON project_boms.id = project_bom_items.bom_id").
		Joins("LEFT JOIN projects ON projects.id = project_boms.project_id")

	if params.Keyword != "" {
		like := "%" + params.Keyword + "%"
		query = query.Where(
			"project_bom_items.name ILIKE ? OR project_bom_items.mpn ILIKE ? OR project_bom_items.supplier ILIKE ? OR project_bom_items.extended_attrs->>'specification' ILIKE ?",
			like, like, like, like,
		)
	}
	if params.Category != "" {
		query = query.Where("project_bom_items.category = ?", params.Category)
	}
	if params.SubCategory != "" {
		query = query.Where("project_bom_items.sub_category = ?", params.SubCategory)
	}
	if params.BOMID != "" {
		query = query.Where("project_bom_items.bom_id = ?", params.BOMID)
	}
	if params.ProjectID != "" {
		query = query.Where("project_boms.project_id = ?", params.ProjectID)
	}
	if params.SupplierID != "" {
		query = query.Where("project_bom_items.supplier_id = ?", params.SupplierID)
	}
	if params.ManufacturerID != "" {
		query = query.Where("project_bom_items.manufacturer_id = ?", params.ManufacturerID)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Fetch with joins to get project/bom/supplier/manufacturer info
	var items []MaterialSearchResult
	err := query.
		Joins("LEFT JOIN srm_suppliers AS sup ON sup.id = project_bom_items.supplier_id").
		Joins("LEFT JOIN srm_suppliers AS mfr ON mfr.id = project_bom_items.manufacturer_id").
		Select("project_bom_items.*, project_boms.project_id, project_boms.bom_type, project_boms.name as bom_name, projects.name as project_name, sup.name as supplier_name, mfr.name as manufacturer_name").
		Order("project_bom_items.updated_at DESC").
		Offset((params.Page - 1) * params.PageSize).
		Limit(params.PageSize).
		Find(&items).Error
	return items, total, err
}

// FindItemsByMPN 按MPN精确查找已有BOM行项（用于导入去重）
func (r *ProjectBOMRepository) FindItemsByMPN(ctx context.Context, bomID string, mpns []string) (map[string]entity.ProjectBOMItem, error) {
	if len(mpns) == 0 {
		return map[string]entity.ProjectBOMItem{}, nil
	}
	var items []entity.ProjectBOMItem
	err := r.db.WithContext(ctx).
		Where("bom_id = ? AND mpn IN ?", bomID, mpns).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string]entity.ProjectBOMItem, len(items))
	for _, item := range items {
		result[item.MPN] = item
	}
	return result, err
}

// CreateRelease 创建BOM发布快照
func (r *ProjectBOMRepository) CreateRelease(ctx context.Context, release *entity.BOMRelease) error {
	return r.db.WithContext(ctx).Create(release).Error
}

// ListPendingReleases 获取待同步的发布快照
func (r *ProjectBOMRepository) ListPendingReleases(ctx context.Context) ([]entity.BOMRelease, error) {
	var releases []entity.BOMRelease
	err := r.db.WithContext(ctx).
		Where("status = ?", "pending").
		Order("created_at ASC").
		Find(&releases).Error
	return releases, err
}

// FindReleaseByID 根据ID查找发布快照
func (r *ProjectBOMRepository) FindReleaseByID(ctx context.Context, id string) (*entity.BOMRelease, error) {
	var release entity.BOMRelease
	err := r.db.WithContext(ctx).First(&release, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &release, nil
}

// UpdateRelease 更新发布快照
func (r *ProjectBOMRepository) UpdateRelease(ctx context.Context, release *entity.BOMRelease) error {
	return r.db.WithContext(ctx).Save(release).Error
}

// === CategoryAttrTemplate Methods ===

// ListTemplates 查询属性模板（按category+sub_category筛选）
func (r *ProjectBOMRepository) ListTemplates(ctx context.Context, category, subCategory string) ([]entity.CategoryAttrTemplate, error) {
	var templates []entity.CategoryAttrTemplate
	query := r.db.WithContext(ctx).Model(&entity.CategoryAttrTemplate{})
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if subCategory != "" {
		query = query.Where("sub_category = ?", subCategory)
	}
	err := query.Order("sort_order ASC").Find(&templates).Error
	return templates, err
}

// CreateTemplate 创建属性模板字段
func (r *ProjectBOMRepository) CreateTemplate(ctx context.Context, t *entity.CategoryAttrTemplate) error {
	return r.db.WithContext(ctx).Create(t).Error
}

// UpdateTemplate 更新属性模板字段
func (r *ProjectBOMRepository) UpdateTemplate(ctx context.Context, t *entity.CategoryAttrTemplate) error {
	return r.db.WithContext(ctx).Save(t).Error
}

// DeleteTemplate 删除属性模板字段
func (r *ProjectBOMRepository) DeleteTemplate(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.CategoryAttrTemplate{}, "id = ?", id).Error
}

// FindTemplateByID 根据ID查找属性模板
func (r *ProjectBOMRepository) FindTemplateByID(ctx context.Context, id string) (*entity.CategoryAttrTemplate, error) {
	var t entity.CategoryAttrTemplate
	if err := r.db.WithContext(ctx).First(&t, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// GetCategoryTree 获取分类树（大类→小类，含item数量）
func (r *ProjectBOMRepository) GetCategoryTree(ctx context.Context, bomID string) ([]map[string]interface{}, error) {
	var results []struct {
		Category    string `json:"category"`
		SubCategory string `json:"sub_category"`
		Count       int    `json:"count"`
	}
	query := r.db.WithContext(ctx).Model(&entity.ProjectBOMItem{}).
		Select("category, sub_category, COUNT(*) as count").
		Group("category, sub_category").
		Order("category, sub_category")
	if bomID != "" {
		query = query.Where("bom_id = ?", bomID)
	}
	if err := query.Find(&results).Error; err != nil {
		return nil, err
	}
	var out []map[string]interface{}
	for _, r := range results {
		out = append(out, map[string]interface{}{
			"category":     r.Category,
			"sub_category": r.SubCategory,
			"count":        r.Count,
		})
	}
	return out, nil
}

// === ProcessRoute Methods ===

// CreateRoute 创建工艺路线
func (r *ProjectBOMRepository) CreateRoute(ctx context.Context, route *entity.ProcessRoute) error {
	return r.db.WithContext(ctx).Create(route).Error
}

// FindRouteByID 根据ID查找工艺路线
func (r *ProjectBOMRepository) FindRouteByID(ctx context.Context, id string) (*entity.ProcessRoute, error) {
	var route entity.ProcessRoute
	err := r.db.WithContext(ctx).
		Preload("Steps", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order ASC") }).
		Preload("Steps.Materials").
		First(&route, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &route, nil
}

// ListRoutesByProject 获取项目的工艺路线列表
func (r *ProjectBOMRepository) ListRoutesByProject(ctx context.Context, projectID string) ([]entity.ProcessRoute, error) {
	var routes []entity.ProcessRoute
	err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("created_at DESC").
		Find(&routes).Error
	return routes, err
}

// UpdateRoute 更新工艺路线
func (r *ProjectBOMRepository) UpdateRoute(ctx context.Context, route *entity.ProcessRoute) error {
	return r.db.WithContext(ctx).Save(route).Error
}

// CreateStep 创建工序
func (r *ProjectBOMRepository) CreateStep(ctx context.Context, step *entity.ProcessStep) error {
	return r.db.WithContext(ctx).Create(step).Error
}

// FindStepByID 根据ID查找工序
func (r *ProjectBOMRepository) FindStepByID(ctx context.Context, id string) (*entity.ProcessStep, error) {
	var step entity.ProcessStep
	err := r.db.WithContext(ctx).Preload("Materials").First(&step, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &step, nil
}

// UpdateStep 更新工序
func (r *ProjectBOMRepository) UpdateStep(ctx context.Context, step *entity.ProcessStep) error {
	return r.db.WithContext(ctx).Save(step).Error
}

// DeleteStep 删除工序
func (r *ProjectBOMRepository) DeleteStep(ctx context.Context, id string) error {
	r.db.WithContext(ctx).Delete(&entity.ProcessStepMaterial{}, "step_id = ?", id)
	return r.db.WithContext(ctx).Delete(&entity.ProcessStep{}, "id = ?", id).Error
}

// ListStepsByRoute 获取工艺路线下的工序
func (r *ProjectBOMRepository) ListStepsByRoute(ctx context.Context, routeID string) ([]entity.ProcessStep, error) {
	var steps []entity.ProcessStep
	err := r.db.WithContext(ctx).
		Preload("Materials").
		Where("route_id = ?", routeID).
		Order("sort_order ASC").
		Find(&steps).Error
	return steps, err
}

// CreateStepMaterial 添加工序物料
func (r *ProjectBOMRepository) CreateStepMaterial(ctx context.Context, m *entity.ProcessStepMaterial) error {
	return r.db.WithContext(ctx).Create(m).Error
}

// DeleteStepMaterial 删除工序物料
func (r *ProjectBOMRepository) DeleteStepMaterial(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.ProcessStepMaterial{}, "id = ?", id).Error
}
