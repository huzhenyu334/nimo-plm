package service

import (
	"context"

	"github.com/bitfantasy/nimo/internal/config"
	"github.com/bitfantasy/nimo/internal/plm/entity"
	"github.com/bitfantasy/nimo/internal/plm/repository"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
)

// Services 服务集合
type Services struct {
	Auth       *AuthService
	User       *UserService
	Product    *ProductService
	Material   *MaterialService
	BOM        *BOMService
	Project    *ProjectService
	ECN        *ECNService
	Document   *DocumentService
	Feishu     *FeishuIntegrationService
	Template   *TemplateService
	Automation *AutomationService
	// V2 新增
	ProjectBOM *ProjectBOMService
}

// NewServices 创建服务集合
func NewServices(repos *repository.Repositories, rdb *redis.Client, cfg *config.Config) *Services {
	// 初始化飞书集成服务
	var feishuSvc *FeishuIntegrationService
	if cfg.Feishu.AppID != "" && cfg.Feishu.AppSecret != "" {
		feishuSvc = NewFeishuIntegrationService(cfg.Feishu.AppID, cfg.Feishu.AppSecret)
	}

	// 初始化MinIO客户端
	var minioClient *minio.Client
	if cfg.MinIO.Endpoint != "" {
		var err error
		minioClient, err = minio.New(cfg.MinIO.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.MinIO.AccessKey, cfg.MinIO.SecretKey, ""),
			Secure: cfg.MinIO.UseSSL,
		})
		if err != nil {
			// Log warning but continue without MinIO
			minioClient = nil
		}
	}

	templateSvc := NewTemplateService(repos.Template, repos.Project, repos.TaskForm)

	return &Services{
		Auth:       NewAuthService(repos.User, rdb, cfg),
		User:       NewUserService(repos.User, rdb),
		Product:    NewProductService(repos.Product, repos.ProductCategory, rdb),
		Material:   NewMaterialService(repos.Material, repos.MaterialCategory, rdb),
		BOM:        NewBOMService(repos.BOM, repos.Material, rdb),
		Project:    NewProjectService(repos.Project, repos.Task, repos.Product, feishuSvc, repos.TaskForm),
		ECN:        NewECNService(repos.ECN, repos.Product, feishuSvc),
		Document:   NewDocumentService(repos.Document, repos.DocumentCategory, minioClient, cfg.MinIO.Bucket),
		Feishu:     feishuSvc,
		Template:   templateSvc,
		Automation: nil, // Will be initialized with logger later if needed
		// V2 新增
		ProjectBOM: NewProjectBOMService(repos.ProjectBOM, repos.Project, repos.Deliverable),
	}
}

// UserService 用户服务
type UserService struct {
	repo *repository.UserRepository
	rdb  *redis.Client
}

// NewUserService 创建用户服务
func NewUserService(repo *repository.UserRepository, rdb *redis.Client) *UserService {
	return &UserService{repo: repo, rdb: rdb}
}

// ListAll 获取所有活跃用户
func (s *UserService) ListAll(ctx context.Context) ([]entity.User, error) {
	return s.repo.ListActive(ctx)
}

// Search 搜索用户（按名字/邮箱模糊匹配）
func (s *UserService) Search(ctx context.Context, query string) ([]entity.User, error) {
	return s.repo.Search(ctx, query)
}

// MaterialService 物料服务
type MaterialService struct {
	repo    *repository.MaterialRepository
	catRepo *repository.MaterialCategoryRepository
	rdb     *redis.Client
}

// NewMaterialService 创建物料服务
func NewMaterialService(repo *repository.MaterialRepository, catRepo *repository.MaterialCategoryRepository, rdb *redis.Client) *MaterialService {
	return &MaterialService{repo: repo, catRepo: catRepo, rdb: rdb}
}

// BOMService BOM服务
type BOMService struct {
	bomRepo      *repository.BOMRepository
	materialRepo *repository.MaterialRepository
	rdb          *redis.Client
}

// NewBOMService 创建BOM服务
func NewBOMService(bomRepo *repository.BOMRepository, materialRepo *repository.MaterialRepository, rdb *redis.Client) *BOMService {
	return &BOMService{
		bomRepo:      bomRepo,
		materialRepo: materialRepo,
		rdb:          rdb,
	}
}
