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
	erpEntity "github.com/bitfantasy/nimo-plm/internal/erp/entity"
	erpHandler "github.com/bitfantasy/nimo-plm/internal/erp/handler"
	erpRepo "github.com/bitfantasy/nimo-plm/internal/erp/repository"
	erpService "github.com/bitfantasy/nimo-plm/internal/erp/service"
	"github.com/bitfantasy/nimo-plm/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
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
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found, using environment variables")
	}

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

	zapLogger.Info("Starting nimo-erp service",
		zap.String("version", Version),
		zap.String("build_time", BuildTime),
	)

	// 初始化数据库
	db, err := initDatabase(cfg.Database)
	if err != nil {
		zapLogger.Fatal("Failed to connect to database", zap.Error(err))
	}

	// AutoMigrate ERP tables
	if err := erpEntity.AutoMigrate(db); err != nil {
		zapLogger.Fatal("Failed to auto-migrate ERP tables", zap.Error(err))
	}
	zapLogger.Info("ERP database migration completed")

	// 初始化 ERP 依赖
	repos := erpRepo.NewRepositories(db)
	services := erpService.NewServices(repos, db)
	handlers := erpHandler.NewHandlers(services)

	// 确定端口
	port := os.Getenv("ERP_PORT")
	if port == "" {
		port = "8081"
	}

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

	// 健康检查
	router.GET("/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "nimo-erp"})
	})
	router.GET("/health/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "nimo-erp"})
	})

	// 版本信息
	router.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":    "nimo-erp",
			"version":    Version,
			"build_time": BuildTime,
		})
	})

	// 静态文件 - ERP 前端
	router.StaticFile("/erp", "./web/erp.html")
	router.StaticFile("/erp/", "./web/erp.html")

	// ERP API v1
	v1 := router.Group("/api/v1/erp")
	v1.Use(middleware.JWTAuth(cfg.JWT.Secret))
	{
		// 供应商管理
		suppliers := v1.Group("/suppliers")
		{
			suppliers.GET("", handlers.Supplier.List)
			suppliers.POST("", handlers.Supplier.Create)
			suppliers.GET("/:id", handlers.Supplier.Get)
			suppliers.PUT("/:id", handlers.Supplier.Update)
			suppliers.DELETE("/:id", handlers.Supplier.Delete)
			suppliers.PUT("/:id/score", handlers.Supplier.UpdateScore)
		}

		// 采购需求
		prs := v1.Group("/purchase-requisitions")
		{
			prs.GET("", handlers.Procurement.ListPRs)
			prs.POST("", handlers.Procurement.CreatePR)
			prs.POST("/:id/approve", handlers.Procurement.ApprovePR)
		}

		// 采购订单
		pos := v1.Group("/purchase-orders")
		{
			pos.GET("", handlers.Procurement.ListPOs)
			pos.POST("", handlers.Procurement.CreatePO)
			pos.GET("/:id", handlers.Procurement.GetPO)
			pos.POST("/:id/submit", handlers.Procurement.SubmitPO)
			pos.POST("/:id/approve", handlers.Procurement.ApprovePO)
			pos.POST("/:id/reject", handlers.Procurement.RejectPO)
			pos.POST("/:id/send", handlers.Procurement.SendPO)
			pos.POST("/:id/receive", handlers.Procurement.ReceivePO)
		}

		// 库存管理
		inventory := v1.Group("/inventory")
		{
			inventory.GET("", handlers.Inventory.List)
			inventory.GET("/:material_id", handlers.Inventory.GetByMaterial)
			inventory.POST("/inbound", handlers.Inventory.Inbound)
			inventory.POST("/outbound", handlers.Inventory.Outbound)
			inventory.POST("/adjust", handlers.Inventory.Adjust)
			inventory.GET("/alerts", handlers.Inventory.Alerts)
			inventory.GET("/transactions", handlers.Inventory.Transactions)
		}

		// MRP
		mrp := v1.Group("/mrp")
		{
			mrp.POST("/run", handlers.MRP.Run)
			mrp.GET("/result", handlers.MRP.GetResult)
			mrp.GET("/runs", handlers.MRP.ListRuns)
			mrp.POST("/apply", handlers.MRP.Apply)
		}

		// 生产管理
		workOrders := v1.Group("/work-orders")
		{
			workOrders.GET("", handlers.Manufacturing.List)
			workOrders.POST("", handlers.Manufacturing.Create)
			workOrders.GET("/:id", handlers.Manufacturing.Get)
			workOrders.POST("/:id/release", handlers.Manufacturing.Release)
			workOrders.POST("/:id/pick", handlers.Manufacturing.Pick)
			workOrders.POST("/:id/report", handlers.Manufacturing.Report)
			workOrders.POST("/:id/complete", handlers.Manufacturing.Complete)
		}

		// 客户管理
		customers := v1.Group("/customers")
		{
			customers.GET("", handlers.Sales.ListCustomers)
			customers.POST("", handlers.Sales.CreateCustomer)
			customers.GET("/:id", handlers.Sales.GetCustomer)
			customers.DELETE("/:id", handlers.Sales.DeleteCustomer)
		}

		// 销售订单
		salesOrders := v1.Group("/sales-orders")
		{
			salesOrders.GET("", handlers.Sales.ListSOs)
			salesOrders.POST("", handlers.Sales.CreateSO)
			salesOrders.GET("/:id", handlers.Sales.GetSO)
			salesOrders.POST("/:id/confirm", handlers.Sales.ConfirmSO)
			salesOrders.POST("/:id/ship", handlers.Sales.ShipSO)
			salesOrders.POST("/:id/cancel", handlers.Sales.CancelSO)
		}

		// 服务工单
		serviceOrders := v1.Group("/service-orders")
		{
			serviceOrders.GET("", handlers.Sales.ListServiceOrders)
			serviceOrders.POST("", handlers.Sales.CreateServiceOrder)
			serviceOrders.GET("/:id", handlers.Sales.GetServiceOrder)
			serviceOrders.POST("/:id/assign", handlers.Sales.AssignServiceOrder)
			serviceOrders.POST("/:id/complete", handlers.Sales.CompleteServiceOrder)
		}
	}

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// 启动服务器
	go func() {
		zapLogger.Info("ERP Server starting", zap.String("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zapLogger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zapLogger.Info("Shutting down ERP server...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		zapLogger.Error("Server forced to shutdown", zap.Error(err))
	}

	zapLogger.Info("ERP Server exited")
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
