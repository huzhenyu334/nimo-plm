package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Logger 日志中间件
func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			zap.Duration("latency", latency),
			zap.String("request_id", c.GetString("request_id")),
		}

		if userID, exists := c.Get("user_id"); exists {
			fields = append(fields, zap.String("user_id", userID.(string)))
		}

		if status >= 500 {
			logger.Error("Server error", fields...)
		} else if status >= 400 {
			logger.Warn("Client error", fields...)
		} else {
			logger.Info("Request", fields...)
		}
	}
}

// CORS 跨域中间件
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Request-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RequestID 请求ID中间件
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.Request.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Next()
	}
}

// JWTClaims JWT claims
type JWTClaims struct {
	UserID      string   `json:"uid"`
	Name        string   `json:"name"`
	Email       string   `json:"email"`
	FeishuUID   string   `json:"feishu_uid"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"perms"`
	jwt.RegisteredClaims
}

// JWTAuth JWT认证中间件
func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// 先尝试从 Authorization header 获取
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}

		// 回退到 query param（SSE 等场景使用）
		if tokenString == "" {
			tokenString = c.Query("token")
		}

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    40100,
				"message": "Authorization is required",
			})
			c.Abort()
			return
		}

		// 解析token
		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    40102,
				"message": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
			c.Set("user_id", claims.UserID)
			c.Set("user_name", claims.Name)
			c.Set("user_email", claims.Email)
			c.Set("feishu_uid", claims.FeishuUID)
			c.Set("roles", claims.Roles)
			c.Set("permissions", claims.Permissions)
			c.Set("claims", claims)
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    40103,
				"message": "Invalid token claims",
			})
			c.Abort()
			return
		}
	}
}

// RequirePermission 权限检查中间件
func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		permissions, exists := c.Get("permissions")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    40300,
				"message": "No permissions found",
			})
			c.Abort()
			return
		}

		perms, ok := permissions.([]string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    40301,
				"message": "Invalid permissions format",
			})
			c.Abort()
			return
		}

		for _, p := range perms {
			if p == permission || p == "*" {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"code":    40302,
			"message": "Permission denied: " + permission,
		})
		c.Abort()
	}
}

// RequireRole 角色检查中间件
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roles, exists := c.Get("roles")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    40310,
				"message": "No roles found",
			})
			c.Abort()
			return
		}

		userRoles, ok := roles.([]string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    40311,
				"message": "Invalid roles format",
			})
			c.Abort()
			return
		}

		for _, r := range userRoles {
			if r == role || r == "plm_admin" {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"code":    40312,
			"message": "Role required: " + role,
		})
		c.Abort()
	}
}
