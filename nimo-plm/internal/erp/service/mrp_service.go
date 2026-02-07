package service

import (
	"fmt"
	"time"

	"github.com/bitfantasy/nimo-plm/internal/erp/entity"
	"github.com/bitfantasy/nimo-plm/internal/erp/repository"
	plmEntity "github.com/bitfantasy/nimo-plm/internal/model/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MRPService struct {
	mrpRepo       *repository.MRPRepository
	purchaseRepo  *repository.PurchaseRepository
	inventoryRepo *repository.InventoryRepository
	woRepo        *repository.WorkOrderRepository
	salesRepo     *repository.SalesRepository
	db            *gorm.DB // 直接访问PLM数据
}

func NewMRPService(
	mrpRepo *repository.MRPRepository,
	purchaseRepo *repository.PurchaseRepository,
	inventoryRepo *repository.InventoryRepository,
	woRepo *repository.WorkOrderRepository,
	salesRepo *repository.SalesRepository,
	db *gorm.DB,
) *MRPService {
	return &MRPService{
		mrpRepo:       mrpRepo,
		purchaseRepo:  purchaseRepo,
		inventoryRepo: inventoryRepo,
		woRepo:        woRepo,
		salesRepo:     salesRepo,
		db:            db,
	}
}

type RunMRPRequest struct {
	ProductID       string `json:"product_id"`       // 指定产品，空=全部
	PlanningHorizon int    `json:"planning_horizon"` // 计划范围（天），默认30
}

// Run 执行MRP计算
func (s *MRPService) Run(req RunMRPRequest, userID string) (*entity.MRPRun, error) {
	if req.PlanningHorizon <= 0 {
		req.PlanningHorizon = 30
	}

	now := time.Now()
	runCode := fmt.Sprintf("MRP-%s%04d", now.Format("20060102"), now.UnixNano()%10000)

	run := &entity.MRPRun{
		ID:              uuid.New().String(),
		RunCode:         runCode,
		Status:          entity.MRPStatusRunning,
		ProductID:       req.ProductID,
		PlanningHorizon: req.PlanningHorizon,
		StartedAt:       now,
		CreatedBy:       userID,
	}
	if err := s.mrpRepo.CreateRun(run); err != nil {
		return nil, fmt.Errorf("创建MRP运行记录失败: %w", err)
	}

	// 异步执行计算（但在当前简单实现中同步完成）
	results, err := s.calculate(run)
	if err != nil {
		run.Status = entity.MRPStatusFailed
		run.ErrorMessage = err.Error()
		completedAt := time.Now()
		run.CompletedAt = &completedAt
		s.mrpRepo.UpdateRun(run)
		return run, fmt.Errorf("MRP计算失败: %w", err)
	}

	// 保存结果
	if len(results) > 0 {
		if err := s.mrpRepo.BatchCreateResults(results); err != nil {
			run.Status = entity.MRPStatusFailed
			run.ErrorMessage = err.Error()
			s.mrpRepo.UpdateRun(run)
			return run, err
		}
	}

	completedAt := time.Now()
	run.Status = entity.MRPStatusCompleted
	run.CompletedAt = &completedAt
	run.TotalItems = len(results)
	s.mrpRepo.UpdateRun(run)

	return run, nil
}

// materialReq 物料需求汇总结构
type materialReq struct {
	MaterialID   string
	MaterialCode string
	MaterialName string
	GrossReq     float64
	SafetyStock  float64
	LeadTimeDays int
	Unit         string
	ActionType   string // PURCHASE or PRODUCE
}

// calculate 核心MRP计算逻辑
func (s *MRPService) calculate(run *entity.MRPRun) ([]entity.MRPResult, error) {
	// Step 1: 获取销售需求（按产品汇总）
	demand, err := s.salesRepo.GetPendingDemand()
	if err != nil {
		return nil, fmt.Errorf("获取销售需求失败: %w", err)
	}

	// 如果指定了产品，只计算该产品
	if run.ProductID != "" {
		if qty, ok := demand[run.ProductID]; ok {
			demand = map[string]float64{run.ProductID: qty}
		} else {
			// 即使没有销售需求，也按安全库存检查
			demand = map[string]float64{run.ProductID: 0}
		}
	}

	// Step 2: 获取所有已发布的BOM
	var allProducts []plmEntity.Product
	if run.ProductID != "" {
		s.db.Where("id = ?", run.ProductID).Find(&allProducts)
	} else {
		s.db.Where("status = 'active'").Find(&allProducts)
	}

	materialReqs := make(map[string]*materialReq)

	// Step 3: BOM展开计算毛需求
	for _, product := range allProducts {
		demandQty := demand[product.ID]

		// 获取该产品的最新已发布BOM
		var bomHeader plmEntity.BOMHeader
		if err := s.db.Where("product_id = ? AND status = 'released'", product.ID).
			Order("created_at DESC").First(&bomHeader).Error; err != nil {
			continue // 没有已发布BOM，跳过
		}

		// 获取BOM项
		var bomItems []plmEntity.BOMItem
		s.db.Where("bom_header_id = ?", bomHeader.ID).Find(&bomItems)

		// 展开BOM，计算每个物料的需求
		s.expandBOM(bomItems, demandQty, materialReqs)
	}

	// Step 4: 计算净需求
	var results []entity.MRPResult
	for matID, req := range materialReqs {
		// 获取现有库存
		onHandStock, _ := s.inventoryRepo.GetTotalStock(matID)

		// 获取在途数量（PO中已批准但未完成收货的）
		inTransitQty, _ := s.purchaseRepo.GetInTransitQty(matID)

		// 获取在制数量（工单中的计划数量 - 已完成数量）
		inProductionQty, _ := s.woRepo.GetInProductionQty(matID)

		// 加入安全库存需求
		grossReq := req.GrossReq + req.SafetyStock

		// 净需求 = 毛需求 - 现有库存 - 在途 - 在制
		netReq := grossReq - onHandStock - inTransitQty - inProductionQty
		if netReq < 0 {
			netReq = 0
		}

		// 计划订单数量（向上取整到最小批量）
		plannedQty := netReq
		// TODO: 可以按最小订货量取整

		// 计算需求日期和下单日期
		now := time.Now()
		requiredDate := now.AddDate(0, 0, run.PlanningHorizon)
		orderDate := requiredDate.AddDate(0, 0, -req.LeadTimeDays)

		result := entity.MRPResult{
			ID:               uuid.New().String(),
			MRPRunID:         run.ID,
			MaterialID:       matID,
			MaterialCode:     req.MaterialCode,
			MaterialName:     req.MaterialName,
			GrossRequirement: grossReq,
			OnHandStock:      onHandStock,
			InTransitQty:     inTransitQty,
			InProductionQty:  inProductionQty,
			SafetyStock:      req.SafetyStock,
			NetRequirement:   netReq,
			PlannedOrderQty:  plannedQty,
			ActionType:       req.ActionType,
			RequiredDate:     &requiredDate,
			LeadTimeDays:     req.LeadTimeDays,
			OrderDate:        &orderDate,
			Unit:             req.Unit,
		}
		results = append(results, result)
	}

	return results, nil
}

// expandBOM 递归展开BOM
func (s *MRPService) expandBOM(items []plmEntity.BOMItem, parentQty float64, reqs map[string]*materialReq) {
	for _, item := range items {
		requiredQty := item.Quantity * parentQty

		// 获取物料信息
		var mat plmEntity.Material
		if err := s.db.Where("id = ?", item.MaterialID).First(&mat).Error; err != nil {
			continue
		}

		if existing, ok := reqs[item.MaterialID]; ok {
			existing.GrossReq += requiredQty
		} else {
			reqs[item.MaterialID] = &materialReq{
				MaterialID:   item.MaterialID,
				MaterialCode: mat.Code,
				MaterialName: mat.Name,
				GrossReq:     requiredQty,
				SafetyStock:  mat.SafetyStock,
				LeadTimeDays: mat.LeadTimeDays,
				Unit:         mat.Unit,
				ActionType:   "PURCHASE", // 默认采购，有子BOM的为生产
			}
		}

		// 递归展开子级BOM项
		var childItems []plmEntity.BOMItem
		s.db.Where("parent_item_id = ?", item.ID).Find(&childItems)
		if len(childItems) > 0 {
			reqs[item.MaterialID].ActionType = "PRODUCE"
			s.expandBOM(childItems, requiredQty, reqs)
		}
	}
}

// GetResults 获取MRP运行结果
func (s *MRPService) GetResults(runID string) ([]entity.MRPResult, error) {
	return s.mrpRepo.GetResultsByRunID(runID)
}

// GetLatestRun 获取最近一次MRP运行
func (s *MRPService) GetLatestRun() (*entity.MRPRun, error) {
	return s.mrpRepo.GetLatestRun()
}

// ListRuns 列出MRP运行记录
func (s *MRPService) ListRuns(page, size int) ([]entity.MRPRun, int64, error) {
	return s.mrpRepo.ListRuns(page, size)
}

// Apply 确认MRP结果，自动创建PR和工单
func (s *MRPService) Apply(runID, userID string) error {
	run, err := s.mrpRepo.GetRunByID(runID)
	if err != nil {
		return fmt.Errorf("MRP运行记录不存在: %w", err)
	}
	if run.Status != entity.MRPStatusCompleted {
		return fmt.Errorf("MRP运行状态不允许应用: %s", run.Status)
	}

	results, err := s.mrpRepo.GetUnappliedResults(runID)
	if err != nil {
		return fmt.Errorf("获取MRP结果失败: %w", err)
	}

	prsGenerated := 0
	wosGenerated := 0

	for _, result := range results {
		if result.NetRequirement <= 0 {
			continue
		}

		if result.ActionType == "PURCHASE" {
			// 生成采购需求PR
			pr := &entity.PurchaseRequisition{
				ID:           uuid.New().String(),
				PRCode:       fmt.Sprintf("PR-%s%04d", time.Now().Format("20060102"), time.Now().UnixNano()%10000),
				MaterialID:   result.MaterialID,
				MaterialCode: result.MaterialCode,
				MaterialName: result.MaterialName,
				Quantity:     result.PlannedOrderQty,
				Unit:         result.Unit,
				RequiredDate: result.RequiredDate,
				Status:       entity.PRStatusDraft,
				Source:       "MRP",
				SourceID:     runID,
				Notes:        fmt.Sprintf("MRP自动生成 - %s", run.RunCode),
				CreatedBy:    userID,
			}
			if err := s.purchaseRepo.CreatePR(pr); err == nil {
				prsGenerated++
			}
		}
		// 工单生成暂时跳过（需要更复杂的逻辑来关联产品和BOM）
	}

	// 标记结果为已应用
	s.mrpRepo.MarkResultsApplied(runID)

	now := time.Now()
	run.Status = entity.MRPStatusApplied
	run.AppliedAt = &now
	run.PRsGenerated = prsGenerated
	run.WOsGenerated = wosGenerated
	return s.mrpRepo.UpdateRun(run)
}
