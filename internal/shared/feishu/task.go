package feishu

import (
	"context"
	"fmt"
)

// =============================================================================
// 任务服务 — 管理飞书任务的创建、更新和完成
// 使用飞书任务API v2版本
// =============================================================================

// CreateTask 创建飞书任务
// 支持设置标题、描述、执行人、截止日期、关联文档
// 返回任务全局唯一ID（guid）
func (c *FeishuClient) CreateTask(ctx context.Context, req CreateTaskReq) (string, error) {
	// 构造请求体
	reqBody := map[string]interface{}{
		"summary": req.Summary,
	}

	// 可选字段
	if req.Description != "" {
		reqBody["description"] = req.Description
	}

	// 任务成员（执行人/关注人）
	if len(req.Members) > 0 {
		members := make([]map[string]interface{}, 0, len(req.Members))
		for _, m := range req.Members {
			members = append(members, map[string]interface{}{
				"id":   m.ID,
				"role": m.Role,
			})
		}
		reqBody["members"] = members
	}

	// 截止时间
	if req.Due != nil {
		reqBody["due"] = map[string]interface{}{
			"time":       req.Due.Time,
			"is_all_day": req.Due.IsAllDay,
		}
	}

	// 任务来源（关联文档等）
	if req.Origin != nil {
		origin := map[string]interface{}{
			"platform_i18n_name": req.Origin.PlatformI18nName,
		}
		if req.Origin.Href != nil {
			origin["href"] = map[string]interface{}{
				"url":   req.Origin.Href.URL,
				"title": req.Origin.Href.Title,
			}
		}
		reqBody["origin"] = origin
	}

	// 发起请求
	var resp CreateTaskResponse
	err := c.doRequest(ctx, "POST", "/open-apis/task/v2/tasks", reqBody, &resp)
	if err != nil {
		return "", fmt.Errorf("创建飞书任务失败: %w", err)
	}

	return resp.Data.Task.Guid, nil
}

// UpdateTask 更新飞书任务
// 支持更新标题、描述、截止时间等字段
func (c *FeishuClient) UpdateTask(ctx context.Context, taskID string, req UpdateTaskReq) error {
	path := fmt.Sprintf("/open-apis/task/v2/tasks/%s", taskID)

	// 构造请求体（只包含需要更新的字段）
	reqBody := make(map[string]interface{})
	updatePaths := make([]string, 0)

	if req.Summary != nil {
		reqBody["summary"] = *req.Summary
		updatePaths = append(updatePaths, "summary")
	}
	if req.Description != nil {
		reqBody["description"] = *req.Description
		updatePaths = append(updatePaths, "description")
	}
	if req.Due != nil {
		reqBody["due"] = map[string]interface{}{
			"time":       req.Due.Time,
			"is_all_day": req.Due.IsAllDay,
		}
		updatePaths = append(updatePaths, "due")
	}

	// 飞书PATCH接口需要update_fields参数
	reqBody["update_fields"] = updatePaths

	var resp UpdateTaskResponse
	err := c.doRequest(ctx, "PATCH", path, reqBody, &resp)
	if err != nil {
		return fmt.Errorf("更新飞书任务失败: %w", err)
	}

	return nil
}

// CompleteTask 完成飞书任务
// 将任务标记为已完成状态
func (c *FeishuClient) CompleteTask(ctx context.Context, taskID string) error {
	path := fmt.Sprintf("/open-apis/task/v2/tasks/%s/complete", taskID)

	var resp CompleteTaskResponse
	err := c.doRequest(ctx, "POST", path, map[string]interface{}{}, &resp)
	if err != nil {
		return fmt.Errorf("完成飞书任务失败: %w", err)
	}

	return nil
}
