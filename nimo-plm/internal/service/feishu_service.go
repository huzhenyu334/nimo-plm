package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// FeishuIntegrationService 飞书集成服务
type FeishuIntegrationService struct {
	appID       string
	appSecret   string
	accessToken string
	tokenExpiry time.Time
	tokenMutex  sync.RWMutex
	httpClient  *http.Client
}

// NewFeishuIntegrationService 创建飞书集成服务
func NewFeishuIntegrationService(appID, appSecret string) *FeishuIntegrationService {
	return &FeishuIntegrationService{
		appID:      appID,
		appSecret:  appSecret,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// FeishuUser 飞书用户信息
type FeishuUser struct {
	UserID   string `json:"user_id"`
	OpenID   string `json:"open_id"`
	UnionID  string `json:"union_id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Mobile   string `json:"mobile"`
	Avatar   string `json:"avatar"`
	DeptIDs  []string `json:"department_ids"`
}

// FeishuDepartment 飞书部门信息
type FeishuDepartment struct {
	DeptID   string `json:"department_id"`
	Name     string `json:"name"`
	ParentID string `json:"parent_department_id"`
	LeaderID string `json:"leader_user_id"`
}

// GetTenantAccessToken 获取企业访问令牌
func (s *FeishuIntegrationService) GetTenantAccessToken(ctx context.Context) (string, error) {
	s.tokenMutex.RLock()
	if s.accessToken != "" && time.Now().Before(s.tokenExpiry) {
		token := s.accessToken
		s.tokenMutex.RUnlock()
		return token, nil
	}
	s.tokenMutex.RUnlock()

	s.tokenMutex.Lock()
	defer s.tokenMutex.Unlock()

	// 再次检查（double-check）
	if s.accessToken != "" && time.Now().Before(s.tokenExpiry) {
		return s.accessToken, nil
	}

	reqBody := map[string]string{
		"app_id":     s.appID,
		"app_secret": s.appSecret,
	}
	body, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Code    int    `json:"code"`
		Msg     string `json:"msg"`
		Token   string `json:"tenant_access_token"`
		Expire  int    `json:"expire"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if result.Code != 0 {
		return "", fmt.Errorf("feishu error: %s", result.Msg)
	}

	s.accessToken = result.Token
	s.tokenExpiry = time.Now().Add(time.Duration(result.Expire-60) * time.Second)

	return s.accessToken, nil
}

// GetUserInfo 获取用户信息
func (s *FeishuIntegrationService) GetUserInfo(ctx context.Context, userID string) (*FeishuUser, error) {
	token, err := s.GetTenantAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/contact/v3/users/%s?user_id_type=user_id", userID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Code int `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			User FeishuUser `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("feishu error: %s", result.Msg)
	}

	return &result.Data.User, nil
}

// SyncDepartments 同步部门列表
func (s *FeishuIntegrationService) SyncDepartments(ctx context.Context, parentID string) ([]FeishuDepartment, error) {
	token, err := s.GetTenantAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	if parentID == "" {
		parentID = "0"
	}

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/contact/v3/departments/%s/children?fetch_child=true", parentID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Items []FeishuDepartment `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("feishu error: %s", result.Msg)
	}

	return result.Data.Items, nil
}

// SyncDepartmentUsers 同步部门用户
func (s *FeishuIntegrationService) SyncDepartmentUsers(ctx context.Context, deptID string) ([]FeishuUser, error) {
	token, err := s.GetTenantAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/contact/v3/users/find_by_department?department_id=%s", deptID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Items []FeishuUser `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("feishu error: %s", result.Msg)
	}

	return result.Data.Items, nil
}

// SendMessage 发送消息
func (s *FeishuIntegrationService) SendMessage(ctx context.Context, receiveIDType, receiveID, msgType, content string) error {
	token, err := s.GetTenantAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("get token: %w", err)
	}

	reqBody := map[string]string{
		"receive_id": receiveID,
		"msg_type":   msgType,
		"content":    content,
	}
	body, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=%s", receiveIDType)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	if result.Code != 0 {
		return fmt.Errorf("feishu error: %s", result.Msg)
	}

	return nil
}

// NotifyUser 通知用户
func (s *FeishuIntegrationService) NotifyUser(ctx context.Context, userID, title, content string) error {
	// 构建卡片消息
	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"title": map[string]interface{}{
				"tag":     "plain_text",
				"content": title,
			},
		},
		"elements": []map[string]interface{}{
			{
				"tag": "div",
				"text": map[string]interface{}{
					"tag":     "plain_text",
					"content": content,
				},
			},
		},
	}

	cardJSON, _ := json.Marshal(card)
	return s.SendMessage(ctx, "user_id", userID, "interactive", string(cardJSON))
}

// CreateApprovalInstance 创建审批实例
func (s *FeishuIntegrationService) CreateApprovalInstance(ctx context.Context, approvalCode, userID string, form map[string]interface{}) (string, error) {
	token, err := s.GetTenantAccessToken(ctx)
	if err != nil {
		return "", fmt.Errorf("get token: %w", err)
	}

	formJSON, _ := json.Marshal(form)

	reqBody := map[string]interface{}{
		"approval_code": approvalCode,
		"user_id":       userID,
		"form":          string(formJSON),
	}
	body, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://open.feishu.cn/open-apis/approval/v4/instances", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			InstanceCode string `json:"instance_code"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if result.Code != 0 {
		return "", fmt.Errorf("feishu error: %s", result.Msg)
	}

	return result.Data.InstanceCode, nil
}

// GetApprovalInstanceDetail 获取审批实例详情
func (s *FeishuIntegrationService) GetApprovalInstanceDetail(ctx context.Context, instanceCode string) (map[string]interface{}, error) {
	token, err := s.GetTenantAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/approval/v4/instances/%s", instanceCode)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Code int                    `json:"code"`
		Msg  string                 `json:"msg"`
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("feishu error: %s", result.Msg)
	}

	return result.Data, nil
}

// CreateCalendarEvent 创建日历事件
func (s *FeishuIntegrationService) CreateCalendarEvent(ctx context.Context, calendarID string, event map[string]interface{}) (string, error) {
	token, err := s.GetTenantAccessToken(ctx)
	if err != nil {
		return "", fmt.Errorf("get token: %w", err)
	}

	body, _ := json.Marshal(event)

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/calendar/v4/calendars/%s/events", calendarID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Event struct {
				EventID string `json:"event_id"`
			} `json:"event"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if result.Code != 0 {
		return "", fmt.Errorf("feishu error: %s", result.Msg)
	}

	return result.Data.Event.EventID, nil
}

// CreateTask 创建飞书任务
func (s *FeishuIntegrationService) CreateTask(ctx context.Context, summary, description string, dueTime *time.Time, memberIDs []string) (string, error) {
	token, err := s.GetTenantAccessToken(ctx)
	if err != nil {
		return "", fmt.Errorf("get token: %w", err)
	}

	task := map[string]interface{}{
		"summary":     summary,
		"description": description,
	}

	if dueTime != nil {
		task["due"] = map[string]interface{}{
			"timestamp": fmt.Sprintf("%d", dueTime.Unix()),
			"is_all_day": false,
		}
	}

	if len(memberIDs) > 0 {
		members := make([]map[string]interface{}, len(memberIDs))
		for i, id := range memberIDs {
			members[i] = map[string]interface{}{
				"id":   id,
				"type": "user",
				"role": "assignee",
			}
		}
		task["members"] = members
	}

	body, _ := json.Marshal(task)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://open.feishu.cn/open-apis/task/v2/tasks", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Task struct {
				GUID string `json:"guid"`
			} `json:"task"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if result.Code != 0 {
		return "", fmt.Errorf("feishu error: %s", result.Msg)
	}

	return result.Data.Task.GUID, nil
}

// UpdateTask 更新飞书任务
func (s *FeishuIntegrationService) UpdateTask(ctx context.Context, taskID string, updates map[string]interface{}) error {
	token, err := s.GetTenantAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("get token: %w", err)
	}

	body, _ := json.Marshal(updates)

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/task/v2/tasks/%s", taskID)
	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	if result.Code != 0 {
		return fmt.Errorf("feishu error: %s", result.Msg)
	}

	return nil
}

// CompleteTask 完成飞书任务
func (s *FeishuIntegrationService) CompleteTask(ctx context.Context, taskID string) error {
	return s.UpdateTask(ctx, taskID, map[string]interface{}{
		"completed_at": fmt.Sprintf("%d", time.Now().Unix()),
	})
}
