package feishu

import (
	"encoding/json"
	"fmt"
)

// =============================================================================
// Webhook处理 — 解析飞书回调事件
// 支持审批实例事件、URL验证事件
// =============================================================================

// HandleApprovalEvent 解析飞书审批实例事件
// 从webhook回调的请求体中提取审批事件信息
// 支持v1和v2两种事件格式
func HandleApprovalEvent(body []byte) (*ApprovalEvent, error) {
	// 先尝试解析事件信封
	var envelope WebhookEvent
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("解析审批事件信封失败: %w", err)
	}

	// v2格式：有header和event字段
	if envelope.Header != nil && envelope.Header.EventType != "" {
		var instanceEvent ApprovalInstanceEvent
		if err := json.Unmarshal(envelope.Event, &instanceEvent); err != nil {
			return nil, fmt.Errorf("解析v2审批事件体失败: %w", err)
		}

		return &ApprovalEvent{
			ApprovalCode: instanceEvent.ApprovalCode,
			InstanceCode: instanceEvent.InstanceCode,
			Status:       instanceEvent.Status,
			OpenID:       instanceEvent.OpenID,
		}, nil
	}

	// v1格式：直接从event字段解析
	if envelope.Event != nil {
		var event ApprovalEvent
		if err := json.Unmarshal(envelope.Event, &event); err != nil {
			return nil, fmt.Errorf("解析v1审批事件体失败: %w", err)
		}
		return &event, nil
	}

	return nil, fmt.Errorf("无法识别的审批事件格式")
}

// HandleVerification 处理飞书URL验证事件
// 飞书在首次订阅事件时会发送验证请求，需要返回challenge值
// 返回值为challenge字符串，需原样返回给飞书
func HandleVerification(body []byte) (string, error) {
	// 尝试解析URL验证事件
	var event URLVerificationEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return "", fmt.Errorf("解析URL验证事件失败: %w", err)
	}

	// 检查事件类型
	if event.Type != EventTypeURLVerification {
		return "", fmt.Errorf("非URL验证事件，类型为: %s", event.Type)
	}

	if event.Challenge == "" {
		return "", fmt.Errorf("URL验证事件缺少challenge字段")
	}

	return event.Challenge, nil
}

// IsVerificationEvent 判断请求体是否为URL验证事件
// 可用于webhook handler中快速判断事件类型
func IsVerificationEvent(body []byte) bool {
	var check struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(body, &check); err != nil {
		return false
	}
	return check.Type == EventTypeURLVerification
}

// GetEventType 从webhook请求体中提取事件类型
// 支持v1和v2格式
func GetEventType(body []byte) string {
	// 先尝试v2格式
	var v2 struct {
		Header *struct {
			EventType string `json:"event_type"`
		} `json:"header"`
	}
	if err := json.Unmarshal(body, &v2); err == nil && v2.Header != nil {
		return v2.Header.EventType
	}

	// 再尝试v1格式
	var v1 struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(body, &v1); err == nil {
		return v1.Type
	}

	return ""
}
