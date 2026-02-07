package feishu

import (
	"context"
	"fmt"
)

// =============================================================================
// 审批服务 — 管理飞书审批定义和审批实例
// 支持创建审批定义、发起审批、查询审批状态
// =============================================================================

// CreateApprovalDefinition 创建审批定义
// 在飞书后台创建一个新的审批定义，支持自定义表单字段
// 返回审批定义code，后续发起审批时需要使用
func (c *FeishuClient) CreateApprovalDefinition(ctx context.Context, def ApprovalDefinition) (string, error) {
	// 构造表单字段
	formFields := make([]map[string]interface{}, 0, len(def.FormFields))
	for _, field := range def.FormFields {
		f := map[string]interface{}{
			"id":   field.ID,
			"type": field.Type,
			"name": field.Name,
		}
		if field.Required {
			f["required"] = true
		}
		if len(field.Options) > 0 {
			options := make([]map[string]string, 0, len(field.Options))
			for _, opt := range field.Options {
				options = append(options, map[string]string{"text": opt})
			}
			f["option"] = options
		}
		if field.Placeholder != "" {
			f["placeholder"] = field.Placeholder
		}
		formFields = append(formFields, f)
	}

	// 构造审批节点
	nodeList := make([]map[string]interface{}, 0, len(def.NodeList))
	for _, node := range def.NodeList {
		n := map[string]interface{}{
			"id":   node.ID,
			"type": node.Type,
		}
		if node.Name != "" {
			n["name"] = node.Name
		}
		if len(node.ApproverIDs) > 0 {
			n["approver"] = node.ApproverIDs
		}
		nodeList = append(nodeList, n)
	}

	// 构造请求体
	reqBody := map[string]interface{}{
		"approval_name": def.Name,
		"form": map[string]interface{}{
			"form_content": formFields,
		},
		"node_list": nodeList,
	}
	if def.Description != "" {
		reqBody["description"] = def.Description
	}

	// 发起请求
	var resp CreateApprovalDefResponse
	err := c.doRequest(ctx, "POST", "/open-apis/approval/v4/approvals", reqBody, &resp)
	if err != nil {
		return "", fmt.Errorf("创建审批定义失败: %w", err)
	}

	return resp.Data.ApprovalCode, nil
}

// CreateApprovalInstance 创建审批实例（发起审批）
// 指定审批定义code、发起人OpenID、表单数据，可选指定各节点审批人
// 返回审批实例code，可用于后续查询审批状态
func (c *FeishuClient) CreateApprovalInstance(ctx context.Context, req CreateApprovalInstanceReq) (string, error) {
	// 构造请求体
	reqBody := map[string]interface{}{
		"approval_code": req.ApprovalCode,
		"open_id":       req.OpenID,
		"form":          req.FormData,
	}

	// 可选：指定节点审批人
	if len(req.NodeApproverList) > 0 {
		approverList := make([]map[string]interface{}, 0, len(req.NodeApproverList))
		for _, item := range req.NodeApproverList {
			approverList = append(approverList, map[string]interface{}{
				"key":   item.Key,
				"value": item.Value,
			})
		}
		reqBody["node_approver_open_id_list"] = approverList
	}

	// 发起请求
	var resp CreateApprovalInstanceResponse
	err := c.doRequest(ctx, "POST", "/open-apis/approval/v4/instances", reqBody, &resp)
	if err != nil {
		return "", fmt.Errorf("创建审批实例失败: %w", err)
	}

	return resp.Data.InstanceCode, nil
}

// GetApprovalInstance 查询审批实例详情
// 返回审批状态、表单数据、审批时间线（含审批人意见）
func (c *FeishuClient) GetApprovalInstance(ctx context.Context, instanceCode string) (*ApprovalInstance, error) {
	path := fmt.Sprintf("/open-apis/approval/v4/instances/%s", instanceCode)

	var resp GetApprovalInstanceResponse
	err := c.doRequest(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, fmt.Errorf("查询审批实例失败: %w", err)
	}

	return &resp.Data, nil
}
