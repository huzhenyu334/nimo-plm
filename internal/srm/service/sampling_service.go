package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bitfantasy/nimo/internal/shared/feishu"
	"github.com/bitfantasy/nimo/internal/srm/entity"
	"github.com/bitfantasy/nimo/internal/srm/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SamplingService 打样服务
type SamplingService struct {
	samplingRepo    *repository.SamplingRepository
	prRepo          *repository.PRRepository
	supplierRepo    *repository.SupplierRepository
	activityLogRepo *repository.ActivityLogRepository
	feishuClient    *feishu.FeishuClient
	approvalCode    string // 飞书审批定义code（打样验证）
	db              *gorm.DB
}

func NewSamplingService(
	samplingRepo *repository.SamplingRepository,
	prRepo *repository.PRRepository,
	supplierRepo *repository.SupplierRepository,
	activityLogRepo *repository.ActivityLogRepository,
	db *gorm.DB,
) *SamplingService {
	return &SamplingService{
		samplingRepo:    samplingRepo,
		prRepo:          prRepo,
		supplierRepo:    supplierRepo,
		activityLogRepo: activityLogRepo,
		db:              db,
	}
}

// SetFeishuClient 注入飞书客户端
func (s *SamplingService) SetFeishuClient(fc *feishu.FeishuClient) {
	s.feishuClient = fc
}

// SetApprovalCode 设置打样验证审批定义code
func (s *SamplingService) SetApprovalCode(code string) {
	s.approvalCode = code
}

// CreateSamplingRequest 发起打样
type CreateSamplingReq struct {
	SupplierID string `json:"supplier_id" binding:"required"`
	SampleQty  int    `json:"sample_qty" binding:"required"`
	Notes      string `json:"notes"`
}

func (s *SamplingService) CreateSamplingRequest(
	ctx context.Context,
	prItemID string,
	req CreateSamplingReq,
	operatorID string,
) (*entity.SamplingRequest, error) {
	// 查找物料
	item, err := s.prRepo.FindItemByID(ctx, prItemID)
	if err != nil {
		return nil, fmt.Errorf("物料不存在")
	}

	// 验证状态：pending或sampling（重新打样）都允许
	if item.Status != entity.PRItemStatusPending && item.Status != entity.PRItemStatusSampling {
		return nil, fmt.Errorf("当前状态 %s 不允许发起打样", item.Status)
	}

	// 获取供应商名称
	var supplierName string
	supplier, err := s.supplierRepo.FindByID(ctx, req.SupplierID)
	if err == nil && supplier != nil {
		supplierName = supplier.Name
	}

	// 获取当前最大轮次
	maxRound, err := s.samplingRepo.GetMaxRound(ctx, prItemID)
	if err != nil {
		return nil, fmt.Errorf("查询打样轮次失败: %w", err)
	}
	round := maxRound + 1

	// 创建打样请求
	sampling := &entity.SamplingRequest{
		ID:          uuid.New().String()[:32],
		PRItemID:    prItemID,
		Round:       round,
		SupplierID:  req.SupplierID,
		SampleQty:   req.SampleQty,
		Status:      entity.SamplingStatusPreparing,
		RequestedBy: operatorID,
		Notes:       req.Notes,
	}

	if err := s.samplingRepo.Create(ctx, sampling); err != nil {
		return nil, fmt.Errorf("创建打样请求失败: %w", err)
	}

	// 更新物料状态为sampling
	if item.Status == entity.PRItemStatusPending {
		item.Status = entity.PRItemStatusSampling
		item.SupplierID = &req.SupplierID
		if err := s.prRepo.UpdateItem(ctx, item); err != nil {
			return nil, fmt.Errorf("更新物料状态失败: %w", err)
		}
	}

	// 记录操作日志
	if s.activityLogRepo != nil {
		content := fmt.Sprintf("发起打样 R%d，供应商: %s，样品数量: %d", round, supplierName, req.SampleQty)
		s.activityLogRepo.LogActivity(ctx, "pr_item", prItemID, item.MaterialCode, "sampling_create", item.Status, entity.PRItemStatusSampling, content, operatorID, "")
	}

	sampling.SupplierName = supplierName
	return sampling, nil
}

// ListSamplingRequests 获取物料的打样记录列表
func (s *SamplingService) ListSamplingRequests(ctx context.Context, prItemID string) ([]entity.SamplingRequest, error) {
	items, err := s.samplingRepo.FindByPRItemID(ctx, prItemID)
	if err != nil {
		return nil, fmt.Errorf("查询打样记录失败: %w", err)
	}

	// 填充供应商名称
	for i := range items {
		supplier, err := s.supplierRepo.FindByID(ctx, items[i].SupplierID)
		if err == nil && supplier != nil {
			items[i].SupplierName = supplier.Name
		}
	}

	return items, nil
}

// UpdateSamplingStatus 更新打样状态（shipping/arrived）
type UpdateSamplingStatusReq struct {
	Status string `json:"status" binding:"required"`
}

func (s *SamplingService) UpdateSamplingStatus(
	ctx context.Context,
	samplingID string,
	req UpdateSamplingStatusReq,
	operatorID string,
) (*entity.SamplingRequest, error) {
	sampling, err := s.samplingRepo.FindByID(ctx, samplingID)
	if err != nil {
		return nil, fmt.Errorf("打样记录不存在")
	}

	// 验证状态流转
	allowed, ok := entity.ValidSamplingTransitions[sampling.Status]
	if !ok {
		return nil, fmt.Errorf("当前状态 %s 不允许流转", sampling.Status)
	}
	valid := false
	for _, s := range allowed {
		if s == req.Status {
			valid = true
			break
		}
	}
	if !valid {
		return nil, fmt.Errorf("不允许从 %s 流转到 %s", sampling.Status, req.Status)
	}

	fromStatus := sampling.Status
	sampling.Status = req.Status

	if req.Status == entity.SamplingStatusArrived {
		now := time.Now()
		sampling.ArrivedAt = &now
	}

	if err := s.samplingRepo.Update(ctx, sampling); err != nil {
		return nil, fmt.Errorf("更新打样状态失败: %w", err)
	}

	// 记录操作日志
	if s.activityLogRepo != nil {
		content := fmt.Sprintf("打样R%d状态变更: %s → %s", sampling.Round, fromStatus, req.Status)
		s.activityLogRepo.LogActivity(ctx, "pr_item", sampling.PRItemID, "", "sampling_status", fromStatus, req.Status, content, operatorID, "")
	}

	return sampling, nil
}

// RequestVerify 发起研发验证（发飞书审批）
type RequestVerifyReq struct {
	ApproverOpenID string `json:"approver_open_id" binding:"required"` // 研发人员飞书OpenID
	InitiatorOpenID string `json:"initiator_open_id" binding:"required"` // 发起人飞书OpenID
}

func (s *SamplingService) RequestVerify(
	ctx context.Context,
	samplingID string,
	req RequestVerifyReq,
	operatorID string,
) (*entity.SamplingRequest, error) {
	sampling, err := s.samplingRepo.FindByID(ctx, samplingID)
	if err != nil {
		return nil, fmt.Errorf("打样记录不存在")
	}

	if sampling.Status != entity.SamplingStatusArrived {
		return nil, fmt.Errorf("只有已到货的打样才能发起验证")
	}

	// 获取物料信息
	item, err := s.prRepo.FindItemByID(ctx, sampling.PRItemID)
	if err != nil {
		return nil, fmt.Errorf("物料不存在")
	}

	// 获取供应商名称
	var supplierName string
	supplier, err := s.supplierRepo.FindByID(ctx, sampling.SupplierID)
	if err == nil && supplier != nil {
		supplierName = supplier.Name
	}

	// 发飞书审批
	if s.feishuClient != nil && s.approvalCode != "" {
		formData := []map[string]interface{}{
			{"id": "material_name", "type": "input", "value": item.MaterialName},
			{"id": "specification", "type": "input", "value": item.Specification},
			{"id": "supplier", "type": "input", "value": supplierName},
			{"id": "sample_qty", "type": "number", "value": fmt.Sprintf("%d", sampling.SampleQty)},
			{"id": "round", "type": "number", "value": fmt.Sprintf("%d", sampling.Round)},
		}
		formJSON, _ := json.Marshal(formData)

		instanceCode, err := s.feishuClient.CreateApprovalInstance(ctx, feishu.CreateApprovalInstanceReq{
			ApprovalCode: s.approvalCode,
			OpenID:       req.InitiatorOpenID,
			FormData:     string(formJSON),
			NodeApproverList: []feishu.NodeApproverItem{
				{Key: "approver_node", Value: []string{req.ApproverOpenID}},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("发起飞书审批失败: %w", err)
		}
		sampling.ApprovalID = instanceCode
	}

	sampling.Status = entity.SamplingStatusVerifying
	sampling.VerifiedBy = operatorID

	if err := s.samplingRepo.Update(ctx, sampling); err != nil {
		return nil, fmt.Errorf("更新打样状态失败: %w", err)
	}

	// 记录操作日志
	if s.activityLogRepo != nil {
		content := fmt.Sprintf("打样R%d发起研发验证", sampling.Round)
		s.activityLogRepo.LogActivity(ctx, "pr_item", sampling.PRItemID, item.MaterialCode, "sampling_verify_request", entity.SamplingStatusArrived, entity.SamplingStatusVerifying, content, operatorID, "")
	}

	return sampling, nil
}

// HandleVerifyCallback 处理飞书审批回调
func (s *SamplingService) HandleVerifyCallback(ctx context.Context, instanceCode, status string) error {
	sampling, err := s.samplingRepo.FindByApprovalID(ctx, instanceCode)
	if err != nil {
		return fmt.Errorf("未找到对应的打样记录: approval_id=%s", instanceCode)
	}

	if sampling.Status != entity.SamplingStatusVerifying {
		return fmt.Errorf("打样状态不是验证中，当前: %s", sampling.Status)
	}

	now := time.Now()
	sampling.VerifiedAt = &now

	if status == feishu.ApprovalStatusApproved {
		sampling.Status = entity.SamplingStatusPassed
		sampling.VerifyResult = "passed"

		// 物料状态 → quoting
		item, err := s.prRepo.FindItemByID(ctx, sampling.PRItemID)
		if err == nil {
			item.Status = entity.PRItemStatusQuoting
			s.prRepo.UpdateItem(ctx, item)

			if s.activityLogRepo != nil {
				content := fmt.Sprintf("打样R%d验证通过，进入报价阶段", sampling.Round)
				s.activityLogRepo.LogActivity(ctx, "pr_item", sampling.PRItemID, item.MaterialCode, "sampling_passed", entity.PRItemStatusSampling, entity.PRItemStatusQuoting, content, sampling.VerifiedBy, "")
			}
		}
	} else if status == feishu.ApprovalStatusRejected {
		sampling.Status = entity.SamplingStatusFailed
		sampling.VerifyResult = "failed"

		// 物料保持sampling状态，可以重新打样
		if s.activityLogRepo != nil {
			item, err := s.prRepo.FindItemByID(ctx, sampling.PRItemID)
			if err == nil {
				content := fmt.Sprintf("打样R%d验证不通过，可重新打样", sampling.Round)
				s.activityLogRepo.LogActivity(ctx, "pr_item", sampling.PRItemID, item.MaterialCode, "sampling_failed", entity.SamplingStatusVerifying, entity.SamplingStatusFailed, content, sampling.VerifiedBy, "")
			}
		}
	}

	return s.samplingRepo.Update(ctx, sampling)
}

// EnsureApprovalDefinition 确保打样验证审批定义存在
func (s *SamplingService) EnsureApprovalDefinition(ctx context.Context) error {
	if s.feishuClient == nil {
		return nil
	}
	if s.approvalCode != "" {
		return nil
	}

	def := feishu.ApprovalDefinition{
		Name:        "打样验证审批",
		Description: "SRM打样完成后，研发人员验证样品是否合格",
		FormFields: []feishu.ApprovalFormField{
			{ID: "material_name", Type: feishu.FieldTypeText, Name: "物料名称", Required: true},
			{ID: "specification", Type: feishu.FieldTypeText, Name: "规格"},
			{ID: "supplier", Type: feishu.FieldTypeText, Name: "供应商", Required: true},
			{ID: "sample_qty", Type: feishu.FieldTypeNumber, Name: "样品数量", Required: true},
			{ID: "round", Type: feishu.FieldTypeNumber, Name: "打样轮次"},
		},
		NodeList: []feishu.ApprovalNode{
			{ID: "approver_node", Name: "研发验证", Type: "AND"},
		},
	}

	code, err := s.feishuClient.CreateApprovalDefinition(ctx, def)
	if err != nil {
		return fmt.Errorf("创建打样审批定义失败: %w", err)
	}

	s.approvalCode = code
	return nil
}
