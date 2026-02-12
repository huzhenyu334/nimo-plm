package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/bitfantasy/nimo/internal/middleware"
	"github.com/bitfantasy/nimo/internal/plm/entity"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	TestSchema = "test_plm"
	JWTSecret  = "nimo-plm-jwt-secret-key-2024"
)

// TestEnv holds test environment resources
type TestEnv struct {
	DB     *gorm.DB
	Router *gin.Engine
	T      *testing.T
}

// projectRoot returns the project root directory by looking for go.mod
func projectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// loadEnv loads .env from the project root
func loadEnv() {
	root := projectRoot()
	if root != "" {
		godotenv.Load(filepath.Join(root, ".env"))
	}
}

// SetupTestDB creates a test database connection using a dedicated test schema.
// Each test gets an isolated schema that is cleaned up after the test.
func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	loadEnv()

	host := getEnv("DB_HOST", "127.0.0.1")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "nimo")
	password := getEnv("DB_PASSWORD", "nimo123")
	dbname := getEnv("DB_NAME", "nimo_plm")

	baseDSN := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// Create a unique test schema for isolation
	schemaName := fmt.Sprintf("%s_%d", TestSchema, time.Now().UnixNano()%1000000)

	// First: create schema using a temporary connection
	setupDB, err := gorm.Open(postgres.Open(baseDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database for schema setup: %v", err)
	}
	setupDB.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schemaName))
	sqlSetup, _ := setupDB.DB()
	sqlSetup.Close()

	// Second: open connection with search_path in DSN so ALL pooled connections use test schema
	testDSN := fmt.Sprintf("%s search_path=%s", baseDSN, schemaName)
	db, err := gorm.Open(postgres.Open(testDSN), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Migrate test tables
	err = db.AutoMigrate(
		&entity.User{},
		&entity.Department{},
		&entity.Role{},
		&entity.Permission{},
		&entity.UserRole{},
		&entity.RolePermission{},
		&entity.Project{},
		&entity.ProjectPhase{},
		&entity.Task{},
		&entity.TaskDependency{},
		&entity.TaskComment{},
		&entity.TaskRole{},
		&entity.ProjectBOM{},
		&entity.ProjectBOMItem{},
		&entity.PartDrawing{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate test tables: %v", err)
	}

	// Cleanup on test completion
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		// Reconnect to drop the schema
		cleanDB, cleanErr := gorm.Open(postgres.Open(baseDSN), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if cleanErr == nil {
			cleanDB.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schemaName))
			sqlClean, _ := cleanDB.DB()
			if sqlClean != nil {
				sqlClean.Close()
			}
		}
	})

	return db
}

// SetupRouter creates a gin test router with JWT middleware
func SetupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	return r
}

// AuthGroup creates an API group with JWT auth middleware for testing
func AuthGroup(r *gin.Engine, path string) *gin.RouterGroup {
	return r.Group(path, middleware.JWTAuth(JWTSecret))
}

// GenerateTestToken creates a valid JWT token for testing
func GenerateTestToken(userID, name, email string, roles, permissions []string) string {
	if roles == nil {
		roles = []string{}
	}
	if permissions == nil {
		permissions = []string{}
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"sub":        userID,
		"uid":        userID,
		"name":       name,
		"email":      email,
		"feishu_uid": "test_feishu_uid",
		"roles":      roles,
		"perms":      permissions,
		"iss":        "nimo-plm",
		"iat":        now.Unix(),
		"exp":        now.Add(24 * time.Hour).Unix(),
		"jti":        fmt.Sprintf("test-jti-%d", now.UnixNano()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(JWTSecret))
	return tokenString
}

// DefaultTestToken returns a token for a default admin test user
func DefaultTestToken() string {
	return GenerateTestToken(
		"test-user-001",
		"Test Admin",
		"admin@test.com",
		[]string{"plm_admin"},
		[]string{"*"},
	)
}

// DoRequest executes an HTTP request against the test router
func DoRequest(r *gin.Engine, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBytes, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBytes)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, _ := http.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ParseResponse parses the JSON response body into a handler.Response-like map
func ParseResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	return result
}

// SeedTestUser creates a test user in the database
func SeedTestUser(t *testing.T, db *gorm.DB, id, name, email string) *entity.User {
	t.Helper()
	user := &entity.User{
		ID:           id,
		FeishuUserID: "feishu_" + id,
		Username:     "user_" + id,
		Name:         name,
		Email:        email,
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to seed test user: %v", err)
	}
	return user
}

// SeedTestRole creates a test role in the database
func SeedTestRole(t *testing.T, db *gorm.DB, id, code, name string, isSystem bool) *entity.Role {
	t.Helper()
	role := &entity.Role{
		ID:        id,
		Code:      code,
		Name:      name,
		Status:    "active",
		IsSystem:  isSystem,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("Failed to seed test role: %v", err)
	}
	return role
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
