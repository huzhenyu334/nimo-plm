package handler

import (
	"net/http"

	"github.com/bitfantasy/nimo-plm/internal/config"
	"github.com/bitfantasy/nimo-plm/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	svc *service.AuthService
	cfg *config.Config
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(svc *service.AuthService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{svc: svc, cfg: cfg}
}

// FeishuLogin 飞书登录
func (h *AuthHandler) FeishuLogin(c *gin.Context) {
	// 生成state用于防止CSRF
	state := uuid.New().String()

	// TODO: 将state存储到Redis，设置短期过期

	// 获取重定向URL
	redirectURI := c.Query("redirect_uri")
	if redirectURI == "" {
		redirectURI = h.cfg.Feishu.RedirectURI
	}

	// 生成飞书授权URL
	loginURL := h.svc.GetFeishuLoginURL(state)

	c.Redirect(http.StatusFound, loginURL)
}

// FeishuCallback 飞书授权回调
func (h *AuthHandler) FeishuCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		Error(c, 40001, "Missing authorization code")
		return
	}

	// TODO: 验证state

	// 处理回调
	user, tokenPair, err := h.svc.HandleFeishuCallback(c.Request.Context(), code)
	if err != nil {
		Error(c, 40002, "Failed to authenticate: "+err.Error())
		return
	}

	// 返回登录结果
	Success(c, gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_in":    tokenPair.ExpiresIn,
		"user": gin.H{
			"id":         user.ID,
			"name":       user.Name,
			"email":      user.Email,
			"avatar_url": user.AvatarURL,
			"roles":      user.RoleCodes,
		},
	})
}

// RefreshTokenRequest 刷新Token请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshToken 刷新Token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body")
		return
	}

	tokenPair, err := h.svc.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		Unauthorized(c, "Invalid or expired refresh token")
		return
	}

	Success(c, gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_in":    tokenPair.ExpiresIn,
	})
}

// GetCurrentUser 获取当前用户信息
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID := GetUserID(c)
	if userID == "" {
		Unauthorized(c, "User not authenticated")
		return
	}

	user, err := h.svc.GetCurrentUser(c.Request.Context(), userID)
	if err != nil {
		NotFound(c, "User not found")
		return
	}

	Success(c, gin.H{
		"id":          user.ID,
		"username":    user.Username,
		"name":        user.Name,
		"email":       user.Email,
		"mobile":      user.Mobile,
		"avatar_url":  user.AvatarURL,
		"status":      user.Status,
		"roles":       user.RoleCodes,
		"permissions": user.PermissionCodes,
		"created_at":  user.CreatedAt,
	})
}

// Logout 退出登录
func (h *AuthHandler) Logout(c *gin.Context) {
	userID := GetUserID(c)
	if userID == "" {
		Unauthorized(c, "User not authenticated")
		return
	}

	if err := h.svc.Logout(c.Request.Context(), userID); err != nil {
		InternalError(c, "Failed to logout")
		return
	}

	Success(c, nil)
}
