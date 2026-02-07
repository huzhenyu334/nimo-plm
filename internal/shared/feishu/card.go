package feishu

import (
	"context"
	"encoding/json"
	"fmt"
)

// =============================================================================
// 消息卡片服务 — 发送飞书交互式消息卡片
// 支持群聊和个人卡片发送，提供预设的通知卡片模板
// =============================================================================

// SendCard 向群聊发送消息卡片
// chatID: 群聊ID
// card: 交互式卡片内容
func (c *FeishuClient) SendCard(ctx context.Context, chatID string, card InteractiveCard) error {
	return c.sendCard(ctx, "chat_id", chatID, card)
}

// SendUserCard 向个人发送消息卡片
// userID: 用户ID（open_id）
// card: 交互式卡片内容
func (c *FeishuClient) SendUserCard(ctx context.Context, userID string, card InteractiveCard) error {
	return c.sendCard(ctx, "open_id", userID, card)
}

// sendCard 发送消息卡片的内部实现
func (c *FeishuClient) sendCard(ctx context.Context, idType, id string, card InteractiveCard) error {
	// 将卡片序列化为JSON字符串
	cardBytes, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("序列化卡片内容失败: %w", err)
	}

	// 构造请求体
	reqBody := map[string]interface{}{
		"receive_id_type": idType,
		"receive_id":      id,
		"msg_type":        "interactive",
		"content":         string(cardBytes),
	}

	// 发送消息，query参数通过URL传递
	path := fmt.Sprintf("/open-apis/im/v1/messages?receive_id_type=%s", idType)

	var resp SendMessageResponse
	if err := c.doRequest(ctx, "POST", path, reqBody, &resp); err != nil {
		return fmt.Errorf("发送消息卡片失败: %w", err)
	}

	return nil
}

// =============================================================================
// 预设卡片模板 — 常用业务通知卡片
// =============================================================================

// NewTaskAssignmentCard 创建任务指派通知卡片
// taskName: 任务名称
// projectName: 所属项目名称
// assigneeName: 被指派人名称
// dueDate: 截止日期（格式如 "2024-03-15"）
func NewTaskAssignmentCard(taskName, projectName, assigneeName, dueDate string) InteractiveCard {
	return InteractiveCard{
		Config: &CardConfig{WideScreenMode: true},
		Header: &CardHeader{
			Title:    CardText{Tag: "plain_text", Content: "📋 新任务指派通知"},
			Template: "blue",
		},
		Elements: []CardElement{
			{
				Tag: "div",
				Fields: []CardField{
					{IsShort: true, Text: CardText{Tag: "lark_md", Content: fmt.Sprintf("**任务名称**\n%s", taskName)}},
					{IsShort: true, Text: CardText{Tag: "lark_md", Content: fmt.Sprintf("**所属项目**\n%s", projectName)}},
					{IsShort: true, Text: CardText{Tag: "lark_md", Content: fmt.Sprintf("**执行人**\n%s", assigneeName)}},
					{IsShort: true, Text: CardText{Tag: "lark_md", Content: fmt.Sprintf("**截止日期**\n%s", dueDate)}},
				},
			},
			{Tag: "hr"},
			{
				Tag: "note",
				Elements: []CardElement{
					{Tag: "plain_text", Content: "请及时查看并开始处理此任务"},
				},
			},
		},
	}
}

// NewReviewResultCard 创建评审结果通知卡片
// reviewName: 评审名称
// result: 评审结果（通过/驳回）
// comment: 评审意见
func NewReviewResultCard(reviewName, result, comment string) InteractiveCard {
	// 根据结果选择颜色模板
	template := "green"
	emoji := "✅"
	if result != "通过" && result != "APPROVED" {
		template = "red"
		emoji = "❌"
	}

	elements := []CardElement{
		{
			Tag: "div",
			Fields: []CardField{
				{IsShort: true, Text: CardText{Tag: "lark_md", Content: fmt.Sprintf("**评审名称**\n%s", reviewName)}},
				{IsShort: true, Text: CardText{Tag: "lark_md", Content: fmt.Sprintf("**评审结果**\n%s %s", emoji, result)}},
			},
		},
	}

	// 添加评审意见（如果有）
	if comment != "" {
		elements = append(elements,
			CardElement{Tag: "hr"},
			CardElement{
				Tag:  "div",
				Text: &CardText{Tag: "lark_md", Content: fmt.Sprintf("**评审意见**\n%s", comment)},
			},
		)
	}

	return InteractiveCard{
		Config: &CardConfig{WideScreenMode: true},
		Header: &CardHeader{
			Title:    CardText{Tag: "plain_text", Content: "📝 评审结果通知"},
			Template: template,
		},
		Elements: elements,
	}
}

// NewPhaseGateCard 创建阶段门控通知卡片
// 用于提醒相关角色进行阶段评审
// phaseName: 阶段名称
// projectName: 项目名称
// roles: 需要参与评审的角色列表
func NewPhaseGateCard(phaseName, projectName string, roles []string) InteractiveCard {
	// 将角色列表拼接为字符串
	roleStr := ""
	for i, role := range roles {
		if i > 0 {
			roleStr += "、"
		}
		roleStr += role
	}

	return InteractiveCard{
		Config: &CardConfig{WideScreenMode: true},
		Header: &CardHeader{
			Title:    CardText{Tag: "plain_text", Content: "🚪 阶段门控评审通知"},
			Template: "orange",
		},
		Elements: []CardElement{
			{
				Tag: "div",
				Fields: []CardField{
					{IsShort: true, Text: CardText{Tag: "lark_md", Content: fmt.Sprintf("**项目名称**\n%s", projectName)}},
					{IsShort: true, Text: CardText{Tag: "lark_md", Content: fmt.Sprintf("**当前阶段**\n%s", phaseName)}},
				},
			},
			{
				Tag:  "div",
				Text: &CardText{Tag: "lark_md", Content: fmt.Sprintf("**评审角色**\n%s", roleStr)},
			},
			{Tag: "hr"},
			{
				Tag: "note",
				Elements: []CardElement{
					{Tag: "plain_text", Content: "请相关角色及时完成阶段门控评审"},
				},
			},
		},
	}
}

// NewRollbackCard 创建任务回退通知卡片
// taskName: 被回退的任务名称
// reason: 回退原因
// rolledBackBy: 操作人名称
func NewRollbackCard(taskName, reason, rolledBackBy string) InteractiveCard {
	return InteractiveCard{
		Config: &CardConfig{WideScreenMode: true},
		Header: &CardHeader{
			Title:    CardText{Tag: "plain_text", Content: "⏪ 任务回退通知"},
			Template: "red",
		},
		Elements: []CardElement{
			{
				Tag: "div",
				Fields: []CardField{
					{IsShort: true, Text: CardText{Tag: "lark_md", Content: fmt.Sprintf("**任务名称**\n%s", taskName)}},
					{IsShort: true, Text: CardText{Tag: "lark_md", Content: fmt.Sprintf("**操作人**\n%s", rolledBackBy)}},
				},
			},
			{
				Tag:  "div",
				Text: &CardText{Tag: "lark_md", Content: fmt.Sprintf("**回退原因**\n%s", reason)},
			},
			{Tag: "hr"},
			{
				Tag: "note",
				Elements: []CardElement{
					{Tag: "plain_text", Content: "请查看回退原因并重新处理任务"},
				},
			},
		},
	}
}
