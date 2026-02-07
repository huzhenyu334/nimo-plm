package feishu

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

// 飞书开放平台API基础地址
const baseURL = "https://open.feishu.cn"

// =============================================================================
// FeishuClient — 飞书API基础客户端
// 提供token管理和通用HTTP请求，可被审批、任务、会议、卡片等子模块共用
// =============================================================================

// FeishuClient 飞书客户端
type FeishuClient struct {
	appID       string     // 应用ID
	appSecret   string     // 应用密钥
	tokenCache  string     // 缓存的app_access_token
	tokenExpire time.Time  // token过期时间
	mu          sync.RWMutex // 保护token缓存的读写锁
	httpClient  *http.Client // HTTP客户端
}

// NewClient 创建飞书客户端实例
func NewClient(appID, appSecret string) *FeishuClient {
	return &FeishuClient{
		appID:     appID,
		appSecret: appSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetAppAccessToken 获取应用访问令牌（自建应用）
// 使用双重检查锁定模式缓存token，提前60秒刷新避免过期
func (c *FeishuClient) GetAppAccessToken(ctx context.Context) (string, error) {
	// 先尝试从缓存获取（读锁）
	c.mu.RLock()
	if c.tokenCache != "" && time.Now().Before(c.tokenExpire) {
		token := c.tokenCache
		c.mu.RUnlock()
		return token, nil
	}
	c.mu.RUnlock()

	// 缓存失效，请求新token（写锁）
	c.mu.Lock()
	defer c.mu.Unlock()

	// 双重检查：其他goroutine可能已经刷新了token
	if c.tokenCache != "" && time.Now().Before(c.tokenExpire) {
		return c.tokenCache, nil
	}

	// 构造请求体
	reqBody := map[string]string{
		"app_id":     c.appID,
		"app_secret": c.appSecret,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	// 发起HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST",
		baseURL+"/open-apis/auth/v3/app_access_token/internal",
		bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("创建token请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求飞书token失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		Code           int    `json:"code"`
		Msg            string `json:"msg"`
		AppAccessToken string `json:"app_access_token"`
		Expire         int    `json:"expire"` // 过期时间（秒）
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析token响应失败: %w", err)
	}

	if result.Code != 0 {
		return "", fmt.Errorf("飞书token错误[%d]: %s", result.Code, result.Msg)
	}

	// 缓存token，提前60秒过期以保证安全
	c.tokenCache = result.AppAccessToken
	c.tokenExpire = time.Now().Add(time.Duration(result.Expire-60) * time.Second)

	return result.AppAccessToken, nil
}

// doRequest 执行飞书API请求
// 自动获取token并添加Authorization头，处理飞书统一错误码
// method: HTTP方法（GET/POST/PATCH/PUT/DELETE）
// path: API路径（如 /open-apis/approval/v4/instances）
// body: 请求体（会被JSON序列化，nil则不发送body）
// result: 响应结构体指针（会被JSON反序列化）
func (c *FeishuClient) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	// 获取token
	token, err := c.GetAppAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("获取访问令牌失败: %w", err)
	}

	// 构造请求体
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("序列化请求体失败: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// 构造HTTP请求
	url := baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+token)

	// 发起请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应体失败: %w", err)
	}

	// 先检查飞书通用错误码
	var baseResp BaseResponse
	if err := json.Unmarshal(respBody, &baseResp); err != nil {
		return fmt.Errorf("解析响应基础结构失败: %w", err)
	}
	if baseResp.Code != 0 {
		return fmt.Errorf("飞书API错误[%d]: %s (path=%s)", baseResp.Code, baseResp.Msg, path)
	}

	// 解析完整响应
	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("解析响应体失败: %w", err)
		}
	}

	return nil
}
