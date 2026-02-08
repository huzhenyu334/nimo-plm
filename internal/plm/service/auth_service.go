package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/bitfantasy/nimo/internal/config"
	"github.com/bitfantasy/nimo/internal/plm/entity"
	"github.com/bitfantasy/nimo/internal/plm/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// FeishuOAuthURL 飞书OAuth授权URL
const FeishuOAuthURL = "https://open.feishu.cn/open-apis/authen/v1/authorize"

// FeishuTokenURL 飞书获取Token URL
const FeishuTokenURL = "https://open.feishu.cn/open-apis/authen/v1/oidc/access_token"

// FeishuUserInfoURL 飞书获取用户信息URL
const FeishuUserInfoURL = "https://open.feishu.cn/open-apis/authen/v1/user_info"

// FeishuAppTokenURL 飞书获取应用Token URL
const FeishuAppTokenURL = "https://open.feishu.cn/open-apis/auth/v3/app_access_token/internal"

// AuthService 认证服务
type AuthService struct {
	userRepo *repository.UserRepository
	rdb      *redis.Client
	cfg      *config.Config
}

// NewAuthService 创建认证服务
func NewAuthService(userRepo *repository.UserRepository, rdb *redis.Client, cfg *config.Config) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		rdb:      rdb,
		cfg:      cfg,
	}
}

// TokenPair Token对
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// FeishuTokenResponse 飞书Token响应
type FeishuTokenResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		AccessToken      string `json:"access_token"`
		RefreshToken     string `json:"refresh_token"`
		TokenType        string `json:"token_type"`
		ExpiresIn        int    `json:"expires_in"`
		RefreshExpiresIn int    `json:"refresh_expires_in"`
	} `json:"data"`
}

// FeishuUserInfoResponse 飞书用户信息响应
type FeishuUserInfoResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Name         string `json:"name"`
		EnName       string `json:"en_name"`
		AvatarUrl    string `json:"avatar_url"`
		AvatarThumb  string `json:"avatar_thumb"`
		AvatarMiddle string `json:"avatar_middle"`
		AvatarBig    string `json:"avatar_big"`
		OpenId       string `json:"open_id"`
		UnionId      string `json:"union_id"`
		Email        string `json:"email"`
		UserId       string `json:"user_id"`
		Mobile       string `json:"mobile"`
		TenantKey    string `json:"tenant_key"`
	} `json:"data"`
}

// FeishuAppTokenResponse 飞书应用Token响应
type FeishuAppTokenResponse struct {
	Code              int    `json:"code"`
	Msg               string `json:"msg"`
	AppAccessToken    string `json:"app_access_token"`
	Expire            int    `json:"expire"`
	TenantAccessToken string `json:"tenant_access_token"`
}

// GetFeishuLoginURL 获取飞书登录URL
func (s *AuthService) GetFeishuLoginURL(state string) string {
	params := url.Values{}
	params.Set("app_id", s.cfg.Feishu.AppID)
	params.Set("redirect_uri", s.cfg.Feishu.RedirectURI)
	params.Set("response_type", "code")
	params.Set("state", state)

	return fmt.Sprintf("%s?%s", FeishuOAuthURL, params.Encode())
}

// HandleFeishuCallback 处理飞书回调
func (s *AuthService) HandleFeishuCallback(ctx context.Context, code string) (*entity.User, *TokenPair, error) {
	// 1. 获取应用access_token
	appToken, err := s.getFeishuAppToken(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("get app token: %w", err)
	}

	// 2. 使用code换取用户access_token
	userToken, err := s.getFeishuUserToken(ctx, appToken, code)
	if err != nil {
		return nil, nil, fmt.Errorf("get user token: %w", err)
	}

	// 3. 获取用户信息
	feishuUser, err := s.getFeishuUserInfo(ctx, userToken)
	if err != nil {
		return nil, nil, fmt.Errorf("get user info: %w", err)
	}

	// 4. 创建或更新用户
	user, err := s.createOrUpdateUser(ctx, feishuUser)
	if err != nil {
		return nil, nil, fmt.Errorf("create or update user: %w", err)
	}

	// 5. 生成JWT Token
	tokenPair, err := s.generateTokenPair(user)
	if err != nil {
		return nil, nil, fmt.Errorf("generate token: %w", err)
	}

	return user, tokenPair, nil
}

// getFeishuAppToken 获取飞书应用Token
func (s *AuthService) getFeishuAppToken(ctx context.Context) (string, error) {
	// 先尝试从Redis获取
	cacheKey := "feishu:app_token"
	cached, err := s.rdb.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		return cached, nil
	}

	// 请求飞书API
	reqBody := map[string]string{
		"app_id":     s.cfg.Feishu.AppID,
		"app_secret": s.cfg.Feishu.AppSecret,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	resp, err := http.Post(FeishuAppTokenURL, "application/json", 
		io.NopCloser(jsonReader(bodyBytes)))
	if err != nil {
		return "", fmt.Errorf("request feishu: %w", err)
	}
	defer resp.Body.Close()

	var result FeishuAppTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if result.Code != 0 {
		return "", fmt.Errorf("feishu error: %s", result.Msg)
	}

	// 缓存到Redis
	s.rdb.Set(ctx, cacheKey, result.AppAccessToken, 
		time.Duration(result.Expire-60)*time.Second)

	return result.AppAccessToken, nil
}

// getFeishuUserToken 获取用户Token
func (s *AuthService) getFeishuUserToken(ctx context.Context, appToken, code string) (string, error) {
	reqBody := map[string]string{
		"grant_type": "authorization_code",
		"code":       code,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", FeishuTokenURL, 
		io.NopCloser(jsonReader(bodyBytes)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+appToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request feishu: %w", err)
	}
	defer resp.Body.Close()

	var result FeishuTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if result.Code != 0 {
		return "", fmt.Errorf("feishu error: %s", result.Msg)
	}

	return result.Data.AccessToken, nil
}

// getFeishuUserInfo 获取用户信息
func (s *AuthService) getFeishuUserInfo(ctx context.Context, userToken string) (*FeishuUserInfoResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", FeishuUserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+userToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request feishu: %w", err)
	}
	defer resp.Body.Close()

	var result FeishuUserInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("feishu error: %s", result.Msg)
	}

	return &result, nil
}

// createOrUpdateUser 创建或更新用户
func (s *AuthService) createOrUpdateUser(ctx context.Context, feishuUser *FeishuUserInfoResponse) (*entity.User, error) {
	// 查找已存在的用户：先按 user_id，再按 open_id
	user, err := s.userRepo.FindByFeishuUserID(ctx, feishuUser.Data.UserId)
	if err != nil && err != repository.ErrNotFound {
		return nil, err
	}
	if user == nil && feishuUser.Data.OpenId != "" {
		user, err = s.userRepo.FindByOpenID(ctx, feishuUser.Data.OpenId)
		if err != nil && err != repository.ErrNotFound {
			return nil, err
		}
	}

	now := time.Now()

	if user == nil {
		// 创建新用户
		email := feishuUser.Data.Email
		if email == "" {
			email = fmt.Sprintf("feishu_%s@placeholder.local", feishuUser.Data.OpenId[:15])
		}
		username := feishuUser.Data.UserId
		if username == "" {
			username = fmt.Sprintf("feishu_%s", feishuUser.Data.OpenId[:15])
		}

		user = &entity.User{
			ID:            generateID(),
			FeishuUserID:  feishuUser.Data.UserId,
			FeishuUnionID: feishuUser.Data.UnionId,
			FeishuOpenID:  feishuUser.Data.OpenId,
			Username:      username,
			Name:          feishuUser.Data.Name,
			Email:         email,
			Mobile:        feishuUser.Data.Mobile,
			AvatarURL:     feishuUser.Data.AvatarUrl,
			Status:        "active",
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("create user: %w", err)
		}

		// 分配默认角色
		if err := s.userRepo.AssignRole(ctx, user.ID, "role_plm_viewer"); err != nil {
			// 不阻断流程，只记录错误
		}
	} else {
		// 更新用户信息（同时补全之前缺失的飞书字段）
		if user.FeishuUserID == "" && feishuUser.Data.UserId != "" {
			user.FeishuUserID = feishuUser.Data.UserId
		}
		if user.FeishuOpenID == "" && feishuUser.Data.OpenId != "" {
			user.FeishuOpenID = feishuUser.Data.OpenId
		}
		if user.FeishuUnionID == "" && feishuUser.Data.UnionId != "" {
			user.FeishuUnionID = feishuUser.Data.UnionId
		}
		user.Name = feishuUser.Data.Name
		if feishuUser.Data.Email != "" {
			user.Email = feishuUser.Data.Email
		}
		user.Mobile = feishuUser.Data.Mobile
		user.AvatarURL = feishuUser.Data.AvatarUrl
		user.LastLoginAt = &now
		user.UpdatedAt = now

		if err := s.userRepo.Update(ctx, user); err != nil {
			return nil, fmt.Errorf("update user: %w", err)
		}
	}

	// 加载用户角色和权限
	if err := s.userRepo.LoadRolesAndPermissions(ctx, user); err != nil {
		return nil, fmt.Errorf("load roles: %w", err)
	}

	return user, nil
}

// generateTokenPair 生成Token对
func (s *AuthService) generateTokenPair(user *entity.User) (*TokenPair, error) {
	now := time.Now()
	jti := uuid.New().String()

	// Access Token
	accessClaims := jwt.MapClaims{
		"sub":        user.ID,
		"uid":        user.ID,
		"name":       user.Name,
		"email":      user.Email,
		"feishu_uid": user.FeishuUserID,
		"roles":      user.RoleCodes,
		"perms":      user.PermissionCodes,
		"iss":        s.cfg.JWT.Issuer,
		"iat":        now.Unix(),
		"exp":        now.Add(s.cfg.JWT.AccessTokenExpire).Unix(),
		"jti":        jti,
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(s.cfg.JWT.Secret))
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	// Refresh Token
	refreshJti := uuid.New().String()
	refreshClaims := jwt.MapClaims{
		"sub":  user.ID,
		"type": "refresh",
		"iss":  s.cfg.JWT.Issuer,
		"iat":  now.Unix(),
		"exp":  now.Add(s.cfg.JWT.RefreshTokenExpire).Unix(),
		"jti":  refreshJti,
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(s.cfg.JWT.Secret))
	if err != nil {
		return nil, fmt.Errorf("sign refresh token: %w", err)
	}

	// 存储Refresh Token到Redis
	ctx := context.Background()
	s.rdb.Set(ctx, "token:refresh:"+refreshJti, user.ID, s.cfg.JWT.RefreshTokenExpire)

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    int64(s.cfg.JWT.AccessTokenExpire.Seconds()),
	}, nil
}

// RefreshToken 刷新Token
func (s *AuthService) RefreshToken(ctx context.Context, refreshTokenString string) (*TokenPair, error) {
	// 解析Refresh Token
	token, err := jwt.Parse(refreshTokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.JWT.Secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// 检查Token类型
	if claims["type"] != "refresh" {
		return nil, fmt.Errorf("invalid token type")
	}

	// 检查Redis中是否存在
	jti := claims["jti"].(string)
	userID, err := s.rdb.Get(ctx, "token:refresh:"+jti).Result()
	if err != nil {
		return nil, fmt.Errorf("refresh token expired or invalid")
	}

	// 获取用户
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// 加载角色和权限
	if err := s.userRepo.LoadRolesAndPermissions(ctx, user); err != nil {
		return nil, fmt.Errorf("load roles: %w", err)
	}

	// 删除旧的Refresh Token
	s.rdb.Del(ctx, "token:refresh:"+jti)

	// 生成新的Token对
	return s.generateTokenPair(user)
}

// Logout 登出
func (s *AuthService) Logout(ctx context.Context, userID string) error {
	// 可以在这里实现Token黑名单等逻辑
	return nil
}

// GetCurrentUser 获取当前用户
func (s *AuthService) GetCurrentUser(ctx context.Context, userID string) (*entity.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if err := s.userRepo.LoadRolesAndPermissions(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Helper functions
func generateID() string {
	return uuid.New().String()[:32]
}

type jsonReaderStruct struct {
	data []byte
	pos  int
}

func (r *jsonReaderStruct) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func jsonReader(data []byte) io.Reader {
	return &jsonReaderStruct{data: data}
}
