package service

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	plmentity "github.com/bitfantasy/nimo/internal/plm/entity"
	"github.com/bitfantasy/nimo/internal/shared/feishu"
	"github.com/bitfantasy/nimo/internal/srm/entity"
	"github.com/bitfantasy/nimo/internal/srm/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BOMProvider PLM BOM数据接口（避免直接依赖PLM包）
type BOMProvider interface {
	GetBOMWithItems(ctx context.Context, bomID string) (projectID string, phase string, items []BOMItemInfo, err error)
}

// BOMItemInfo BOM行项信息
type BOMItemInfo struct {
	MaterialID    string
	MaterialCode  string
	MaterialName  string
	Specification string
	Category      string
	Quantity      float64
	Unit          string
}

// ProcurementService 采购服务
type ProcurementService struct {
	prRepo          *repository.PRRepository
	poRepo          *repository.PORepository
	db              *gorm.DB
	feishuClient    *feishu.FeishuClient
	activityLogRepo *repository.ActivityLogRepository
}

func NewProcurementService(prRepo *repository.PRRepository, poRepo *repository.PORepository, db *gorm.DB) *ProcurementService {
	return &ProcurementService{
		prRepo: prRepo,
		poRepo: poRepo,
		db:     db,
	}
}

// SetActivityLogRepo 注入操作日志仓库
func (s *ProcurementService) SetActivityLogRepo(repo *repository.ActivityLogRepository) {
	s.activityLogRepo = repo
}

// logActivity 记录操作日志（安全调用）
func (s *ProcurementService) logActivity(ctx context.Context, entityType, entityID, entityCode, action, fromStatus, toStatus, content, operatorID string) {
	if s.activityLogRepo != nil {
		s.activityLogRepo.LogActivity(ctx, entityType, entityID, entityCode, action, fromStatus, toStatus, content, operatorID, "")
	}
}

// SetFeishuClient 注入飞书客户端
func (s *ProcurementService) SetFeishuClient(fc *feishu.FeishuClient) {
	s.feishuClient = fc
}

// === 采购需求(PR) ===

// ListPRs 获取PR列表
func (s *ProcurementService) ListPRs(ctx context.Context, page, pageSize int, filters map[string]string) ([]entity.PurchaseRequest, int64, error) {
	return s.prRepo.FindAll(ctx, page, pageSize, filters)
}

// GetPR 获取PR详情
func (s *ProcurementService) GetPR(ctx context.Context, id string) (*entity.PurchaseRequest, error) {
	return s.prRepo.FindByID(ctx, id)
}

// CreatePRRequest 创建PR请求
type CreatePRRequest struct {
	Title        string         `json:"title" binding:"required"`
	Type         string         `json:"type"`
	Priority     string         `json:"priority"`
	ProjectID    *string        `json:"project_id"`
	Phase        string         `json:"phase"`
	RequiredDate *time.Time     `json:"required_date"`
	Notes        string         `json:"notes"`
	Items        []CreatePRItem `json:"items"`
}

type CreatePRItem struct {
	MaterialID    *string  `json:"material_id"`
	MaterialCode  string   `json:"material_code"`
	MaterialName  string   `json:"material_name" binding:"required"`
	Specification string   `json:"specification"`
	Category      string   `json:"category"`
	Quantity      float64  `json:"quantity" binding:"required"`
	Unit          string   `json:"unit"`
	ExpectedDate  *time.Time `json:"expected_date"`
	Notes         string   `json:"notes"`
}

// CreatePR 创建采购需求
func (s *ProcurementService) CreatePR(ctx context.Context, userID string, req *CreatePRRequest) (*entity.PurchaseRequest, error) {
	code, err := s.prRepo.GenerateCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("生成PR编码失败: %w", err)
	}

	pr := &entity.PurchaseRequest{
		ID:           uuid.New().String()[:32],
		PRCode:       code,
		Title:        req.Title,
		Type:         req.Type,
		Priority:     req.Priority,
		Status:       entity.PRStatusSourcing,
		ProjectID:    req.ProjectID,
		Phase:        req.Phase,
		RequiredDate: req.RequiredDate,
		RequestedBy:  userID,
		Notes:        req.Notes,
	}

	if pr.Priority == "" {
		pr.Priority = "normal"
	}

	// 创建行项
	for i, item := range req.Items {
		unit := item.Unit
		if unit == "" {
			unit = "pcs"
		}
		pr.Items = append(pr.Items, entity.PRItem{
			ID:            uuid.New().String()[:32],
			PRID:          pr.ID,
			MaterialID:    item.MaterialID,
			MaterialCode:  item.MaterialCode,
			MaterialName:  item.MaterialName,
			Specification: item.Specification,
			Category:      item.Category,
			Quantity:      item.Quantity,
			Unit:          unit,
			Status:        entity.PRItemStatusPending,
			ExpectedDate:  item.ExpectedDate,
			Notes:         item.Notes,
			SortOrder:     i + 1,
		})
	}

	if err := s.prRepo.Create(ctx, pr); err != nil {
		return nil, err
	}

	s.logActivity(ctx, "pr", pr.ID, pr.PRCode, "create", "", entity.PRStatusSourcing,
		fmt.Sprintf("创建采购需求: %s", pr.Title), userID)

	return pr, nil
}

// UpdatePRRequest 更新PR请求
type UpdatePRRequest struct {
	Title        *string    `json:"title"`
	Type         *string    `json:"type"`
	Priority     *string    `json:"priority"`
	Phase        *string    `json:"phase"`
	RequiredDate *time.Time `json:"required_date"`
	Notes        *string    `json:"notes"`
}

// UpdatePR 更新采购需求
func (s *ProcurementService) UpdatePR(ctx context.Context, id string, req *UpdatePRRequest) (*entity.PurchaseRequest, error) {
	pr, err := s.prRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		pr.Title = *req.Title
	}
	if req.Type != nil {
		pr.Type = *req.Type
	}
	if req.Priority != nil {
		pr.Priority = *req.Priority
	}
	if req.Phase != nil {
		pr.Phase = *req.Phase
	}
	if req.RequiredDate != nil {
		pr.RequiredDate = req.RequiredDate
	}
	if req.Notes != nil {
		pr.Notes = *req.Notes
	}

	if err := s.prRepo.Update(ctx, pr); err != nil {
		return nil, err
	}
	return pr, nil
}

// ApprovePR 审批PR
func (s *ProcurementService) ApprovePR(ctx context.Context, id, userID string) (*entity.PurchaseRequest, error) {
	pr, err := s.prRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	oldStatus := pr.Status
	pr.Status = entity.PRStatusApproved
	pr.ApprovedBy = &userID
	pr.ApprovedAt = &now

	if err := s.prRepo.Update(ctx, pr); err != nil {
		return nil, err
	}

	s.logActivity(ctx, "pr", pr.ID, pr.PRCode, "status_change", oldStatus, entity.PRStatusApproved,
		"采购需求审批通过", userID)

	return pr, nil
}

// CreatePRFromBOM 从BOM创建采购需求
func (s *ProcurementService) CreatePRFromBOM(ctx context.Context, projectID, bomID, userID string, bomItems []BOMItemInfo, phase string) (*entity.PurchaseRequest, error) {
	// 防重复：检查是否已有该BOM的PR
	existing, err := s.prRepo.FindByBOMID(ctx, bomID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	code, err := s.prRepo.GenerateCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("生成PR编码失败: %w", err)
	}

	// Look up PLM project phase (EVT/DVT/PVT/MP)
	projectPhase := phase
	var plmProject plmentity.Project
	if err := s.db.WithContext(ctx).Select("current_phase").Where("id = ?", projectID).First(&plmProject).Error; err == nil && plmProject.Phase != "" {
		projectPhase = plmProject.Phase
	}

	pr := &entity.PurchaseRequest{
		ID:          uuid.New().String()[:32],
		PRCode:      code,
		Title:       fmt.Sprintf("BOM自动生成采购需求 - %s", phase),
		Type:        entity.PRTypeSample,
		Priority:    "normal",
		Status:      entity.PRStatusSourcing,
		ProjectID:   &projectID,
		BOMID:       &bomID,
		Phase:       projectPhase,
		RequestedBy: userID,
	}

	for i, item := range bomItems {
		pr.Items = append(pr.Items, entity.PRItem{
			ID:            uuid.New().String()[:32],
			PRID:          pr.ID,
			MaterialID:    strPtr(item.MaterialID),
			MaterialCode:  item.MaterialCode,
			MaterialName:  item.MaterialName,
			Specification: item.Specification,
			Category:      item.Category,
			Quantity:      item.Quantity,
			Unit:          item.Unit,
			Status:        entity.PRItemStatusPending,
			SortOrder:     i + 1,
		})
	}

	if err := s.prRepo.Create(ctx, pr); err != nil {
		return nil, err
	}
	return pr, nil
}

// AutoCreatePRFromBOM 从BOM自动创建打样采购需求（PLM审批通过后调用）
// 直接读取PLM数据库表获取项目名、BOM信息和BOM行项
// 返回创建的PR ID和错误
func (s *ProcurementService) AutoCreatePRFromBOM(ctx context.Context, projectID, bomID, userID string) (string, error) {
	// 1. 读取PLM项目信息
	var project plmentity.Project
	if err := s.db.WithContext(ctx).Where("id = ?", projectID).First(&project).Error; err != nil {
		return "", fmt.Errorf("读取项目信息失败: %w", err)
	}

	// 2. 读取PLM BOM信息
	var bom plmentity.ProjectBOM
	if err := s.db.WithContext(ctx).Where("id = ?", bomID).First(&bom).Error; err != nil {
		return "", fmt.Errorf("读取BOM信息失败: %w", err)
	}

	// 3. 读取BOM行项
	var bomItems []plmentity.ProjectBOMItem
	if err := s.db.WithContext(ctx).Where("bom_id = ?", bomID).Order("item_number").Find(&bomItems).Error; err != nil {
		return "", fmt.Errorf("读取BOM行项失败: %w", err)
	}
	if len(bomItems) == 0 {
		log.Printf("[SRM] BOM %s 无行项，跳过PR创建", bomID)
		return "", nil
	}

	// 4. 转换为BOMItemInfo并调用已有的CreatePRFromBOM
	var items []BOMItemInfo
	for _, bi := range bomItems {
		materialID := ""
		if bi.MaterialID != nil {
			materialID = *bi.MaterialID
		}
		materialCode := ""
		if bi.ExtendedAttrs != nil {
			if v, ok := bi.ExtendedAttrs["manufacturer_pn"]; ok {
				if s, ok := v.(string); ok {
					materialCode = s
				}
			}
		}
		specification := ""
		if bi.ExtendedAttrs != nil {
			if v, ok := bi.ExtendedAttrs["specification"]; ok {
				if s, ok := v.(string); ok {
					specification = s
				}
			}
		}
		items = append(items, BOMItemInfo{
			MaterialID:    materialID,
			MaterialCode:  materialCode,
			MaterialName:  bi.Name,
			Specification: specification,
			Category:      bi.Category,
			Quantity:      bi.Quantity,
			Unit:          bi.Unit,
		})
	}

	// 5. 防重复检查
	existing, err := s.prRepo.FindByBOMID(ctx, bomID)
	if err != nil {
		return "", err
	}
	if existing != nil {
		log.Printf("[SRM] BOM %s 已有PR %s，跳过", bomID, existing.PRCode)
		return existing.ID, nil
	}

	// 6. 生成PR编码
	code, err := s.prRepo.GenerateCode(ctx)
	if err != nil {
		return "", fmt.Errorf("生成PR编码失败: %w", err)
	}

	// 7. 创建PR，标题格式: {project_name} - {bom_name} 打样采购
	title := fmt.Sprintf("%s - %s 打样采购", project.Name, bom.Name)
	// Use project phase (EVT/DVT/PVT/MP), not BOM type
	prPhase := project.Phase
	if prPhase == "" {
		prPhase = bom.BOMType
	}
	pr := &entity.PurchaseRequest{
		ID:          uuid.New().String()[:32],
		PRCode:      code,
		Title:       title,
		Type:        entity.PRTypeSample,
		Priority:    "normal",
		Status:      entity.PRStatusSourcing,
		ProjectID:   &projectID,
		BOMID:       &bomID,
		Phase:       prPhase,
		RequestedBy: userID,
	}

	// 根据BOM类型确定source_bom_type和material_group
	sourceBOMType := bom.BOMType // e.g. EBOM/SBOM/ABOM
	materialGroup := bomTypeToMaterialGroup(sourceBOMType)

	for i, item := range items {
		unit := item.Unit
		if unit == "" {
			unit = "pcs"
		}
		itemMaterialGroup := materialGroup
		// ABOM情况下按item category推断
		if sourceBOMType == "ABOM" && item.Category != "" {
			itemMaterialGroup = categoryToMaterialGroup(item.Category)
		}
		pr.Items = append(pr.Items, entity.PRItem{
			ID:            uuid.New().String()[:32],
			PRID:          pr.ID,
			MaterialID:    strPtr(item.MaterialID),
			MaterialCode:  item.MaterialCode,
			MaterialName:  item.MaterialName,
			Specification: item.Specification,
			Category:      item.Category,
			Quantity:      item.Quantity,
			Unit:          unit,
			Status:        entity.PRItemStatusPending,
			SortOrder:     i + 1,
			SourceBOMType: sourceBOMType,
			MaterialGroup: itemMaterialGroup,
		})
	}

	if err := s.prRepo.Create(ctx, pr); err != nil {
		return "", fmt.Errorf("创建打样PR失败: %w", err)
	}

	log.Printf("[SRM] 自动创建打样PR: %s (项目=%s, BOM=%s, %d项, group=%s)", pr.PRCode, project.Name, bom.Name, len(items), materialGroup)

	// 发送飞书通知
	go s.sendPRCreatedNotification(context.Background(), project.Name, pr.PRCode, len(items))

	return pr.ID, nil
}

// sendPRCreatedNotification 发送PR创建飞书通知
func (s *ProcurementService) sendPRCreatedNotification(ctx context.Context, projectName, prCode string, itemCount int) {
	if s.feishuClient == nil {
		return
	}

	// 硬编码管理员用户ID
	adminUserID := "ou_5b159fc157d4042f1e8088b1ffebb2da"

	// SRM采购需求页面链接
	rawURL := "http://43.134.86.237:8080/srm/purchase-requests"
	detailURL := fmt.Sprintf("https://applink.feishu.cn/client/web_url/open?url=%s&mode=window", url.QueryEscape(rawURL))

	card := feishu.InteractiveCard{
		Config: &feishu.CardConfig{WideScreenMode: true},
		Header: &feishu.CardHeader{
			Title:    feishu.CardText{Tag: "plain_text", Content: "新打样采购需求"},
			Template: "blue",
		},
		Elements: []feishu.CardElement{
			{
				Tag: "div",
				Fields: []feishu.CardField{
					{IsShort: true, Text: feishu.CardText{Tag: "lark_md", Content: fmt.Sprintf("**PLM项目**\n%s", projectName)}},
					{IsShort: true, Text: feishu.CardText{Tag: "lark_md", Content: fmt.Sprintf("**采购需求编码**\n%s", prCode)}},
					{IsShort: true, Text: feishu.CardText{Tag: "lark_md", Content: fmt.Sprintf("**零件数量**\n%d 个", itemCount)}},
				},
			},
			{
				Tag:  "div",
				Text: &feishu.CardText{Tag: "lark_md", Content: fmt.Sprintf("PLM项目 **%s** 的BOM已审批通过，自动创建了采购需求 **%s**，共 **%d** 个零件需要采购。", projectName, prCode, itemCount)},
			},
			{Tag: "hr"},
			{
				Tag: "action",
				Actions: []feishu.CardAction{
					{
						Tag:  "button",
						Text: feishu.CardText{Tag: "plain_text", Content: "查看采购需求"},
						Type: "primary",
						URL:  detailURL,
					},
				},
			},
		},
	}

	if err := s.feishuClient.SendUserCard(ctx, adminUserID, card); err != nil {
		log.Printf("[SRM] 发送飞书PR创建通知失败: %v", err)
	} else {
		log.Printf("[SRM] 飞书PR创建通知已发送: %s", prCode)
	}
}

// sendPOApprovedNotification 发送PO审批通过飞书通知
func (s *ProcurementService) sendPOApprovedNotification(ctx context.Context, po *entity.PurchaseOrder) {
	if s.feishuClient == nil {
		return
	}

	// 硬编码管理员用户ID
	adminUserID := "ou_5b159fc157d4042f1e8088b1ffebb2da"

	// SRM采购订单页面链接
	rawURL := "http://43.134.86.237:8080/srm/purchase-orders"
	detailURL := fmt.Sprintf("https://applink.feishu.cn/client/web_url/open?url=%s&mode=window", url.QueryEscape(rawURL))

	totalText := "未定价"
	if po.TotalAmount != nil {
		totalText = fmt.Sprintf("¥%.2f", *po.TotalAmount)
	}

	itemCount := len(po.Items)

	card := feishu.InteractiveCard{
		Config: &feishu.CardConfig{WideScreenMode: true},
		Header: &feishu.CardHeader{
			Title:    feishu.CardText{Tag: "plain_text", Content: "✅ 采购订单已审批"},
			Template: "green",
		},
		Elements: []feishu.CardElement{
			{
				Tag: "div",
				Fields: []feishu.CardField{
					{IsShort: true, Text: feishu.CardText{Tag: "lark_md", Content: fmt.Sprintf("**订单编码**\n%s", po.POCode)}},
					{IsShort: true, Text: feishu.CardText{Tag: "lark_md", Content: fmt.Sprintf("**订单金额**\n%s", totalText)}},
					{IsShort: true, Text: feishu.CardText{Tag: "lark_md", Content: fmt.Sprintf("**行项数量**\n%d 项", itemCount)}},
				},
			},
			{
				Tag:  "div",
				Text: &feishu.CardText{Tag: "lark_md", Content: fmt.Sprintf("采购订单 **%s** 已审批通过，可以发送给供应商确认。", po.POCode)},
			},
			{Tag: "hr"},
			{
				Tag: "action",
				Actions: []feishu.CardAction{
					{
						Tag:  "button",
						Text: feishu.CardText{Tag: "plain_text", Content: "查看采购订单"},
						Type: "primary",
						URL:  detailURL,
					},
				},
			},
		},
	}

	if err := s.feishuClient.SendUserCard(ctx, adminUserID, card); err != nil {
		log.Printf("[SRM] 发送飞书PO审批通知失败: %v", err)
	} else {
		log.Printf("[SRM] 飞书PO审批通知已发送: %s", po.POCode)
	}
}

// AssignSupplierRequest 分配供应商请求
type AssignSupplierRequest struct {
	SupplierID   string     `json:"supplier_id" binding:"required"`
	UnitPrice    *float64   `json:"unit_price"`
	ExpectedDate *time.Time `json:"expected_date"`
}

// AssignSupplierToItem 为PR行项分配供应商
func (s *ProcurementService) AssignSupplierToItem(ctx context.Context, prID, itemID string, req *AssignSupplierRequest) (*entity.PRItem, error) {
	// 验证PR存在
	_, err := s.prRepo.FindByID(ctx, prID)
	if err != nil {
		return nil, fmt.Errorf("采购需求不存在")
	}

	// 查找行项
	item, err := s.prRepo.FindItemByID(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("行项不存在")
	}

	// 验证行项属于该PR
	if item.PRID != prID {
		return nil, fmt.Errorf("行项不属于该采购需求")
	}

	// 更新供应商信息
	item.SupplierID = &req.SupplierID
	if req.UnitPrice != nil {
		item.UnitPrice = req.UnitPrice
		total := *req.UnitPrice * item.Quantity
		item.TotalAmount = &total
	}
	if req.ExpectedDate != nil {
		item.ExpectedDate = req.ExpectedDate
	}

	// 更新状态
	if item.Status == entity.PRItemStatusPending {
		item.Status = entity.PRItemStatusSourcing
	}

	if err := s.prRepo.UpdateItem(ctx, item); err != nil {
		return nil, fmt.Errorf("更新行项失败: %w", err)
	}

	s.logActivity(ctx, "pr_item", item.ID, "", "assign_supplier", "",
		item.Status, fmt.Sprintf("为零件 %s 分配供应商", item.MaterialName), "")

	return item, nil
}

// GeneratePOsFromPR 从PR生成采购订单（按供应商分组）
func (s *ProcurementService) GeneratePOsFromPR(ctx context.Context, prID, userID string) ([]*entity.PurchaseOrder, error) {
	pr, err := s.prRepo.FindByID(ctx, prID)
	if err != nil {
		return nil, fmt.Errorf("采购需求不存在")
	}

	// 筛选已分配供应商的行项
	supplierGroups := make(map[string][]entity.PRItem)
	for _, item := range pr.Items {
		if item.SupplierID != nil && *item.SupplierID != "" {
			supplierGroups[*item.SupplierID] = append(supplierGroups[*item.SupplierID], item)
		}
	}

	if len(supplierGroups) == 0 {
		return nil, fmt.Errorf("没有已分配供应商的行项")
	}

	var createdPOs []*entity.PurchaseOrder

	// 先获取基准编码，后续在循环中递增
	baseCode, err := s.poRepo.GenerateCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("生成PO编码失败: %w", err)
	}
	// 解析基准序号
	year := time.Now().Format("2006")
	var baseSeq int
	fmt.Sscanf(baseCode, "PO-"+year+"-%04d", &baseSeq)
	poIndex := 0

	// 在事务中为每个供应商创建PO
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for supplierID, items := range supplierGroups {
			code := fmt.Sprintf("PO-%s-%04d", year, baseSeq+poIndex)
			poIndex++

			// 找最早的交期
			var earliestDate *time.Time
			for _, item := range items {
				if item.ExpectedDate != nil {
					if earliestDate == nil || item.ExpectedDate.Before(*earliestDate) {
						earliestDate = item.ExpectedDate
					}
				}
			}

			po := &entity.PurchaseOrder{
				ID:           uuid.New().String()[:32],
				POCode:       code,
				SupplierID:   supplierID,
				PRID:         &prID,
				Type:         pr.Type,
				Status:       entity.POStatusDraft,
				Currency:     "CNY",
				ExpectedDate: earliestDate,
				CreatedBy:    userID,
			}

			var totalAmount float64
			for i, item := range items {
				var itemTotal *float64
				if item.UnitPrice != nil {
					t := *item.UnitPrice * item.Quantity
					itemTotal = &t
					totalAmount += t
				}
				po.Items = append(po.Items, entity.POItem{
					ID:            uuid.New().String()[:32],
					POID:          po.ID,
					PRItemID:      &item.ID,
					MaterialID:    item.MaterialID,
					MaterialCode:  item.MaterialCode,
					MaterialName:  item.MaterialName,
					Specification: item.Specification,
					Quantity:      item.Quantity,
					Unit:          item.Unit,
					UnitPrice:     item.UnitPrice,
					TotalAmount:   itemTotal,
					Status:        entity.POItemStatusPending,
					SortOrder:     i + 1,
				})
			}

			if totalAmount > 0 {
				po.TotalAmount = &totalAmount
			}

			if err := tx.Create(po).Error; err != nil {
				return fmt.Errorf("创建PO失败: %w", err)
			}
			createdPOs = append(createdPOs, po)

			// 更新PRItem状态为ordered
			for _, item := range items {
				if err := tx.Model(&entity.PRItem{}).Where("id = ?", item.ID).
					Update("status", entity.PRItemStatusOrdered).Error; err != nil {
					return fmt.Errorf("更新PRItem状态失败: %w", err)
				}
			}
		}

		// 更新PR状态为sourcing
		if err := tx.Model(&entity.PurchaseRequest{}).Where("id = ?", prID).
			Update("status", entity.PRStatusSourcing).Error; err != nil {
			return fmt.Errorf("更新PR状态失败: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	for _, po := range createdPOs {
		s.logActivity(ctx, "po", po.ID, po.POCode, "create", "", entity.POStatusDraft,
			fmt.Sprintf("从PR生成采购订单, 供应商ID: %s", po.SupplierID), userID)
	}

	return createdPOs, nil
}

// === 采购订单(PO) ===

// ListPOs 获取PO列表
func (s *ProcurementService) ListPOs(ctx context.Context, page, pageSize int, filters map[string]string) ([]entity.PurchaseOrder, int64, error) {
	return s.poRepo.FindAll(ctx, page, pageSize, filters)
}

// GetPO 获取PO详情
func (s *ProcurementService) GetPO(ctx context.Context, id string) (*entity.PurchaseOrder, error) {
	return s.poRepo.FindByID(ctx, id)
}

// CreatePORequest 创建PO请求
type CreatePORequest struct {
	SupplierID      string         `json:"supplier_id" binding:"required"`
	PRID            *string        `json:"pr_id"`
	Type            string         `json:"type" binding:"required"`
	ExpectedDate    *time.Time     `json:"expected_date"`
	ShippingAddress string         `json:"shipping_address"`
	PaymentTerms    string         `json:"payment_terms"`
	Notes           string         `json:"notes"`
	Items           []CreatePOItem `json:"items"`
}

type CreatePOItem struct {
	PRItemID      *string  `json:"pr_item_id"`
	MaterialID    *string  `json:"material_id"`
	MaterialCode  string   `json:"material_code"`
	MaterialName  string   `json:"material_name" binding:"required"`
	Specification string   `json:"specification"`
	Quantity      float64  `json:"quantity" binding:"required"`
	Unit          string   `json:"unit"`
	UnitPrice     *float64 `json:"unit_price"`
	Notes         string   `json:"notes"`
}

// CreatePO 创建采购订单
func (s *ProcurementService) CreatePO(ctx context.Context, userID string, req *CreatePORequest) (*entity.PurchaseOrder, error) {
	code, err := s.poRepo.GenerateCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("生成PO编码失败: %w", err)
	}

	po := &entity.PurchaseOrder{
		ID:              uuid.New().String()[:32],
		POCode:          code,
		SupplierID:      req.SupplierID,
		PRID:            req.PRID,
		Type:            req.Type,
		Status:          entity.POStatusDraft,
		Currency:        "CNY",
		ExpectedDate:    req.ExpectedDate,
		ShippingAddress: req.ShippingAddress,
		PaymentTerms:    req.PaymentTerms,
		CreatedBy:       userID,
		Notes:           req.Notes,
	}

	var totalAmount float64
	for i, item := range req.Items {
		unit := item.Unit
		if unit == "" {
			unit = "pcs"
		}
		var itemTotal *float64
		if item.UnitPrice != nil {
			t := *item.UnitPrice * item.Quantity
			itemTotal = &t
			totalAmount += t
		}
		po.Items = append(po.Items, entity.POItem{
			ID:            uuid.New().String()[:32],
			POID:          po.ID,
			PRItemID:      item.PRItemID,
			MaterialID:    item.MaterialID,
			MaterialCode:  item.MaterialCode,
			MaterialName:  item.MaterialName,
			Specification: item.Specification,
			Quantity:      item.Quantity,
			Unit:          unit,
			UnitPrice:     item.UnitPrice,
			TotalAmount:   itemTotal,
			Status:        entity.POItemStatusPending,
			SortOrder:     i + 1,
			Notes:         item.Notes,
		})
	}

	if totalAmount > 0 {
		po.TotalAmount = &totalAmount
	}

	if err := s.poRepo.Create(ctx, po); err != nil {
		return nil, err
	}
	return po, nil
}

// UpdatePORequest 更新PO请求
type UpdatePORequest struct {
	ExpectedDate    *time.Time `json:"expected_date"`
	ShippingAddress *string    `json:"shipping_address"`
	PaymentTerms    *string    `json:"payment_terms"`
	Notes           *string    `json:"notes"`
}

// UpdatePO 更新采购订单
func (s *ProcurementService) UpdatePO(ctx context.Context, id string, req *UpdatePORequest) (*entity.PurchaseOrder, error) {
	po, err := s.poRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.ExpectedDate != nil {
		po.ExpectedDate = req.ExpectedDate
	}
	if req.ShippingAddress != nil {
		po.ShippingAddress = *req.ShippingAddress
	}
	if req.PaymentTerms != nil {
		po.PaymentTerms = *req.PaymentTerms
	}
	if req.Notes != nil {
		po.Notes = *req.Notes
	}

	if err := s.poRepo.Update(ctx, po); err != nil {
		return nil, err
	}
	return po, nil
}

// ApprovePO 审批PO
func (s *ProcurementService) ApprovePO(ctx context.Context, id, userID string) (*entity.PurchaseOrder, error) {
	po, err := s.poRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if po.Status == entity.POStatusApproved {
		return nil, fmt.Errorf("该采购订单已审批通过，不可重复审批")
	}

	now := time.Now()
	oldStatus := po.Status
	po.Status = entity.POStatusApproved
	po.ApprovedBy = &userID
	po.ApprovedAt = &now

	if err := s.poRepo.Update(ctx, po); err != nil {
		return nil, err
	}

	s.logActivity(ctx, "po", po.ID, po.POCode, "status_change", oldStatus, entity.POStatusApproved,
		"采购订单审批通过", userID)

	// 发送PO审批通过飞书通知
	go s.sendPOApprovedNotification(context.Background(), po)

	return po, nil
}

// SubmitPO 提交PO审批
func (s *ProcurementService) SubmitPO(ctx context.Context, id string) (*entity.PurchaseOrder, error) {
	po, err := s.poRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if po.Status != entity.POStatusDraft {
		return nil, fmt.Errorf("只有草稿状态的订单可以提交")
	}
	po.Status = entity.POStatusSubmitted
	if err := s.poRepo.Update(ctx, po); err != nil {
		return nil, err
	}
	return po, nil
}

// DeletePO 删除PO（仅draft状态）
func (s *ProcurementService) DeletePO(ctx context.Context, id string) error {
	po, err := s.poRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if po.Status != entity.POStatusDraft {
		return fmt.Errorf("只有草稿状态的订单可以删除")
	}
	return s.poRepo.Delete(ctx, id)
}

// GenerateFromBOMRequest 从BOM生成PO请求
type GenerateFromBOMRequest struct {
	BOMID   string   `json:"bom_id" binding:"required"`
	ItemIDs []string `json:"item_ids" binding:"required"`
}

// GeneratePOsFromBOM 从BOM生成采购订单（按供应商分组）
func (s *ProcurementService) GeneratePOsFromBOM(ctx context.Context, userID string, req *GenerateFromBOMRequest) ([]*entity.PurchaseOrder, error) {
	// Fetch specified BOM items
	var bomItems []struct {
		ID             string   `gorm:"column:id"`
		Name           string   `gorm:"column:name"`
		MPN            string   `gorm:"column:mpn"`
		Quantity       float64  `gorm:"column:quantity"`
		Unit           string   `gorm:"column:unit"`
		UnitPrice      *float64 `gorm:"column:unit_price"`
		SupplierID     *string  `gorm:"column:supplier_id"`
		Supplier       string   `gorm:"column:supplier"`
		Category       string   `gorm:"column:category"`
		SubCategory    string   `gorm:"column:sub_category"`
	}
	if err := s.db.Table("project_bom_items").
		Where("bom_id = ? AND id IN ?", req.BOMID, req.ItemIDs).
		Find(&bomItems).Error; err != nil {
		return nil, fmt.Errorf("获取BOM行项失败: %w", err)
	}
	if len(bomItems) == 0 {
		return nil, fmt.Errorf("未找到指定的BOM行项")
	}

	// Group by supplier_id
	type supplierGroup struct {
		supplierID string
		items      []struct {
			ID             string
			Name           string
			MPN            string
			Quantity       float64
			Unit           string
			UnitPrice      *float64
			Category       string
			SubCategory    string
		}
	}
	groups := map[string]*supplierGroup{}
	var noSupplierItems []struct {
		ID             string
		Name           string
		MPN            string
		Quantity       float64
		Unit           string
		UnitPrice      *float64
		Category       string
		SubCategory    string
	}

	for _, item := range bomItems {
		sid := ""
		if item.SupplierID != nil && *item.SupplierID != "" {
			sid = *item.SupplierID
		}
		entry := struct {
			ID             string
			Name           string
			MPN            string
			Quantity       float64
			Unit           string
			UnitPrice      *float64
			Category       string
			SubCategory    string
		}{item.ID, item.Name, item.MPN, item.Quantity, item.Unit, item.UnitPrice, item.Category, item.SubCategory}

		if sid == "" {
			noSupplierItems = append(noSupplierItems, entry)
		} else {
			if groups[sid] == nil {
				groups[sid] = &supplierGroup{supplierID: sid}
			}
			groups[sid].items = append(groups[sid].items, entry)
		}
	}

	// Create POs per supplier
	var result []*entity.PurchaseOrder
	createPO := func(supplierID string, items []struct {
		ID             string
		Name           string
		MPN            string
		Quantity       float64
		Unit           string
		UnitPrice      *float64
		Category       string
		SubCategory    string
	}) error {
		code, err := s.poRepo.GenerateCode(ctx)
		if err != nil {
			return err
		}
		poID := uuid.New().String()[:32]
		var totalAmount float64
		var poItems []entity.POItem
		for i, item := range items {
			itemAmount := 0.0
			if item.UnitPrice != nil {
				itemAmount = item.Quantity * *item.UnitPrice
			}
			totalAmount += itemAmount
			bomID := item.ID
			var amt *float64
			if itemAmount > 0 {
				amt = &itemAmount
			}
			poItems = append(poItems, entity.POItem{
				ID:           uuid.New().String()[:32],
				POID:         poID,
				BOMItemID:    &bomID,
				MaterialName: item.Name,
				Specification: item.MPN,
				Quantity:     item.Quantity,
				Unit:         item.Unit,
				UnitPrice:    item.UnitPrice,
				TotalAmount:  amt,
				SortOrder:    i + 1,
			})
		}
		po := &entity.PurchaseOrder{
			ID:          poID,
			POCode:      code,
			SupplierID:  supplierID,
			Type:        "production",
			Status:      entity.POStatusDraft,
			TotalAmount: &totalAmount,
			Currency:    "CNY",
			CreatedBy:   userID,
			Items:       poItems,
		}
		if err := s.poRepo.Create(ctx, po); err != nil {
			return err
		}
		// Re-read to get supplier info
		created, _ := s.poRepo.FindByID(ctx, poID)
		if created != nil {
			result = append(result, created)
		} else {
			result = append(result, po)
		}
		return nil
	}

	for sid, group := range groups {
		if err := createPO(sid, group.items); err != nil {
			return nil, fmt.Errorf("生成采购订单失败: %w", err)
		}
	}

	// Items without supplier go into a single "unassigned" PO if any
	if len(noSupplierItems) > 0 {
		// Create PO with empty supplier_id — user will assign later
		code, err := s.poRepo.GenerateCode(ctx)
		if err != nil {
			return nil, err
		}
		poID := uuid.New().String()[:32]
		var totalAmount float64
		var poItems []entity.POItem
		for i, item := range noSupplierItems {
			itemAmount := 0.0
			if item.UnitPrice != nil {
				itemAmount = item.Quantity * *item.UnitPrice
			}
			totalAmount += itemAmount
			bomID := item.ID
			var amt *float64
			if itemAmount > 0 {
				amt = &itemAmount
			}
			poItems = append(poItems, entity.POItem{
				ID:           uuid.New().String()[:32],
				POID:         poID,
				BOMItemID:    &bomID,
				MaterialName: item.Name,
				Specification: item.MPN,
				Quantity:     item.Quantity,
				Unit:         item.Unit,
				UnitPrice:    item.UnitPrice,
				TotalAmount:  amt,
				SortOrder:    i + 1,
			})
		}
		po := &entity.PurchaseOrder{
			ID:          poID,
			POCode:      code,
			SupplierID:  "unassigned",
			Type:        "production",
			Status:      entity.POStatusDraft,
			TotalAmount: &totalAmount,
			Currency:    "CNY",
			CreatedBy:   userID,
			Notes:       "从BOM生成，部分物料未指定供应商",
			Items:       poItems,
		}
		if err := s.poRepo.Create(ctx, po); err != nil {
			return nil, fmt.Errorf("生成未指定供应商订单失败: %w", err)
		}
		result = append(result, po)
	}

	return result, nil
}

// ReceiveItemRequest 收货请求
type ReceiveItemRequest struct {
	ReceivedQty float64 `json:"received_qty" binding:"required"`
}

// ReceiveItem 收货
func (s *ProcurementService) ReceiveItem(ctx context.Context, poID, itemID string) error {
	// 验证PO存在
	_, err := s.poRepo.FindByID(ctx, poID)
	if err != nil {
		return err
	}
	return nil // 实际收货在handler层调用repo
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// bomTypeToMaterialGroup BOM类型映射到物料分组
func bomTypeToMaterialGroup(bomType string) string {
	switch bomType {
	case "EBOM":
		return "electronic"
	case "SBOM":
		return "structural"
	case "ABOM":
		return "assembly"
	case "TOOLING":
		return "tooling"
	default:
		return "electronic" // 默认电子料
	}
}

// categoryToMaterialGroup 根据物料类别推断物料分组（ABOM场景）
func categoryToMaterialGroup(category string) string {
	switch category {
	case "electronic", "电子", "IC", "PCB", "PCBA", "connector", "passive":
		return "electronic"
	case "structural", "结构", "机构", "外壳", "housing", "bracket":
		return "structural"
	case "tooling", "治具", "模具", "jig", "fixture", "mold":
		return "tooling"
	default:
		return "assembly"
	}
}
