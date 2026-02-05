package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/bitfantasy/nimo-plm/internal/config"
	"github.com/bitfantasy/nimo-plm/internal/handler"
	"github.com/bitfantasy/nimo-plm/internal/middleware"
	"github.com/bitfantasy/nimo-plm/internal/repository"
	"github.com/bitfantasy/nimo-plm/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化日志
	zapLogger, err := initLogger(cfg.Log)
	if err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}
	defer zapLogger.Sync()

	zapLogger.Info("Starting nimo-plm service",
		zap.String("version", Version),
		zap.String("build_time", BuildTime),
	)

	// 初始化数据库
	db, err := initDatabase(cfg.Database)
	if err != nil {
		zapLogger.Fatal("Failed to connect to database", zap.Error(err))
	}

	// 初始化Redis
	rdb := initRedis(cfg.Redis)

	// 初始化依赖
	repos := repository.NewRepositories(db)
	services := service.NewServices(repos, rdb, cfg)
	handlers := handler.NewHandlers(services, cfg)

	// 设置Gin模式
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.Logger(zapLogger))
	router.Use(middleware.CORS())
	router.Use(middleware.RequestID())

	// 注册路由
	registerRoutes(router, handlers, services, cfg)

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// 启动服务器
	go func() {
		zapLogger.Info("Server starting", zap.Int("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zapLogger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zapLogger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		zapLogger.Error("Server forced to shutdown", zap.Error(err))
	}

	zapLogger.Info("Server exited")
}

func initLogger(cfg config.LogConfig) (*zap.Logger, error) {
	var zapCfg zap.Config

	if cfg.Format == "json" {
		zapCfg = zap.NewProductionConfig()
	} else {
		zapCfg = zap.NewDevelopmentConfig()
	}

	switch cfg.Level {
	case "debug":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	}

	return zapCfg.Build()
}

func initDatabase(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	return db, nil
}

func initRedis(cfg config.RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})
}

func registerRoutes(r *gin.Engine, h *handler.Handlers, svc *service.Services, cfg *config.Config) {
	// 健康检查
	r.GET("/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/health/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 版本信息
	r.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"version":    Version,
			"build_time": BuildTime,
		})
	})

	// API v1
	v1 := r.Group("/api/v1")
	{
		// 认证 (无需登录)
		auth := v1.Group("/auth")
		{
			auth.GET("/feishu/login", h.Auth.FeishuLogin)
			auth.GET("/feishu/callback", h.Auth.FeishuCallback)
			auth.POST("/refresh", h.Auth.RefreshToken)
		}

		// 需要认证的接口
		authorized := v1.Group("")
		authorized.Use(middleware.JWTAuth(cfg.JWT.Secret))
		{
			// 当前用户
			authorized.GET("/auth/me", h.Auth.GetCurrentUser)
			authorized.POST("/auth/logout", h.Auth.Logout)

			// 用户管理
			users := authorized.Group("/users")
			{
				users.GET("", h.User.List)
				users.GET("/:id", h.User.Get)
			}

			// 产品管理
			products := authorized.Group("/products")
			{
				products.GET("", h.Product.List)
				products.POST("", h.Product.Create)
				products.GET("/:id", h.Product.Get)
				products.PUT("/:id", h.Product.Update)
				products.DELETE("/:id", h.Product.Delete)
				products.POST("/:id/release", h.Product.Release)

				// BOM
				products.GET("/:id/bom", h.BOM.Get)
				products.GET("/:id/bom/versions", h.BOM.ListVersions)
				products.POST("/:id/bom/items", h.BOM.AddItem)
				products.PUT("/:id/bom/items/:itemId", h.BOM.UpdateItem)
				products.DELETE("/:id/bom/items/:itemId", h.BOM.DeleteItem)
				products.POST("/:id/bom/release", h.BOM.Release)
				products.GET("/:id/bom/compare", h.BOM.Compare)
			}

			// 产品类别
			authorized.GET("/product-categories", h.Product.ListCategories)

			// 物料管理
			materials := authorized.Group("/materials")
			{
				materials.GET("", h.Material.List)
				materials.POST("", h.Material.Create)
				materials.GET("/:id", h.Material.Get)
				materials.PUT("/:id", h.Material.Update)
			}

			// 物料类别
			authorized.GET("/material-categories", h.Material.ListCategories)

			// 项目管理
			projects := authorized.Group("/projects")
			{
				projects.GET("", h.Project.ListProjects)
				projects.POST("", h.Project.CreateProject)
				projects.GET("/:id", h.Project.GetProject)
				projects.PUT("/:id", h.Project.UpdateProject)
				projects.DELETE("/:id", h.Project.DeleteProject)
				projects.PUT("/:id/status", h.Project.UpdateProjectStatus)

				// 项目阶段
				projects.GET("/:id/phases", h.Project.ListPhases)
				projects.PUT("/:id/phases/:phaseId/status", h.Project.UpdatePhaseStatus)

				// 项目任务
				projects.GET("/:id/tasks", h.Project.ListTasks)
				projects.POST("/:id/tasks", h.Project.CreateTask)
				projects.GET("/:id/tasks/:taskId", h.Project.GetTask)
				projects.PUT("/:id/tasks/:taskId", h.Project.UpdateTask)
				projects.DELETE("/:id/tasks/:taskId", h.Project.DeleteTask)
				projects.PUT("/:id/tasks/:taskId/status", h.Project.UpdateTaskStatus)
				projects.GET("/:id/tasks/:taskId/subtasks", h.Project.ListSubTasks)
				projects.GET("/:id/tasks/:taskId/comments", h.Project.ListTaskComments)
				projects.POST("/:id/tasks/:taskId/comments", h.Project.AddTaskComment)
				projects.GET("/:id/tasks/:taskId/dependencies", h.Project.ListTaskDependencies)
				projects.POST("/:id/tasks/:taskId/dependencies", h.Project.AddTaskDependency)
				projects.DELETE("/:id/tasks/:taskId/dependencies/:depId", h.Project.RemoveTaskDependency)
				projects.GET("/:id/overdue-tasks", h.Project.GetOverdueTasks)
			}

			// 我的任务
			authorized.GET("/my/tasks", h.Project.GetMyTasks)

			// ECN管理
			ecns := authorized.Group("/ecns")
			{
				ecns.GET("", h.ECN.List)
				ecns.POST("", h.ECN.Create)
				ecns.GET("/:id", h.ECN.Get)
				ecns.PUT("/:id", h.ECN.Update)
				ecns.POST("/:id/submit", h.ECN.Submit)
				ecns.POST("/:id/approve", h.ECN.Approve)
				ecns.POST("/:id/reject", h.ECN.Reject)
				ecns.POST("/:id/implement", h.ECN.Implement)
				ecns.GET("/:id/affected-items", h.ECN.ListAffectedItems)
				ecns.POST("/:id/affected-items", h.ECN.AddAffectedItem)
				ecns.DELETE("/:id/affected-items/:itemId", h.ECN.RemoveAffectedItem)
				ecns.GET("/:id/approvals", h.ECN.ListApprovals)
				ecns.POST("/:id/approvers", h.ECN.AddApprover)
			}

			// 文档管理
			documents := authorized.Group("/documents")
			{
				documents.GET("", h.Document.List)
				documents.POST("", h.Document.Upload)
				documents.GET("/:id", h.Document.Get)
				documents.PUT("/:id", h.Document.Update)
				documents.DELETE("/:id", h.Document.Delete)
				documents.GET("/:id/download", h.Document.Download)
				documents.POST("/:id/release", h.Document.Release)
				documents.POST("/:id/obsolete", h.Document.Obsolete)
				documents.GET("/:id/versions", h.Document.ListVersions)
				documents.POST("/:id/versions", h.Document.UploadNewVersion)
				documents.GET("/:id/versions/:versionId/download", h.Document.DownloadVersion)
			}

			// 文档分类
			authorized.GET("/document-categories", h.Document.ListCategories)
		}
	}
}
