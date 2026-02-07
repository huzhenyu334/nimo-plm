package feishu

import (
	"context"
	"fmt"
)

// =============================================================================
// 会议服务 — 管理飞书日历事件（会议）
// 从旧代码 internal/plm/service/feishu_service.go 的 CreateCalendarEvent 迁移并增强
// =============================================================================

// CreateMeeting 创建日历事件（会议）
// 支持设置标题、描述（含文档链接）、参会人、时间、通知
// 返回日历事件ID
func (c *FeishuClient) CreateMeeting(ctx context.Context, req CreateMeetingReq) (string, error) {
	// 构造参会人列表
	attendees := make([]map[string]interface{}, 0, len(req.AttendeeIDs))
	for _, id := range req.AttendeeIDs {
		attendees = append(attendees, map[string]interface{}{
			"type":    "user",
			"user_id": id,
		})
	}

	// 构造请求体
	reqBody := map[string]interface{}{
		"summary":     req.Summary,
		"description": req.Description,
		"start_time": map[string]interface{}{
			"timestamp": fmt.Sprintf("%d", req.StartTime.Unix()),
		},
		"end_time": map[string]interface{}{
			"timestamp": fmt.Sprintf("%d", req.EndTime.Unix()),
		},
		"attendees":         attendees,
		"need_notification": req.NeedNotification,
	}

	// 发起请求
	var resp CreateMeetingResponse
	err := c.doRequest(ctx, "POST", "/open-apis/calendar/v4/calendars/primary/events", reqBody, &resp)
	if err != nil {
		return "", fmt.Errorf("创建日历事件失败: %w", err)
	}

	return resp.Data.Event.EventID, nil
}
