package main

import (
	"flag"
	"fmt"
	"gochat/internal/api"
	"gochat/internal/config"
	"gochat/internal/infra"
	"gochat/internal/repository"
	"gochat/internal/service"
	"log"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("c", "configs/config.yaml", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化日志
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}
	defer logger.Sync() // 把缓冲区里的日志全部写入磁盘

	// ── 初始化 MySQL ──
	db, err := infra.NewMySQLPool(&cfg.MySQL)
	if err != nil {
		logger.Fatal("连接 MySQL 失败", zap.Error(err))
	}
	if err := db.Ping(); err != nil {
		logger.Fatal("Ping MySQL 失败", zap.Error(err))
	}
	logger.Info("MySQL 已连接")

	// ── 初始化仓库层 ──
	mysqlRepo := repository.NewMySQLRepo(db)

	// ── 初始化服务层 ──
	authSvc := service.NewAuthService(mysqlRepo, cfg.JWT.Secret, cfg.JWT.AccessExpHours, cfg.JWT.RefreshExpDays)

	// ── 初始化处理器层 ──
	authHandler := api.NewAuthHandler(authSvc)

	r := gin.Default()
	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		api.Success(c, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")
	authHandler.RegisterRoutes(v1)

	logger.Info("服务器启动")
	err = r.Run(fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		logger.Fatal("服务器启动失败", zap.Error(err))
	}
}
