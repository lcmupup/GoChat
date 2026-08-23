package main

import (
	"flag"
	"fmt"
	"gochat/internal/api"
	"gochat/internal/config"
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

	r := gin.New()
	r.Use(gin.Recovery())
	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		api.Success(c, gin.H{"status": "ok"})
	})

	logger.Info("服务器启动")
	err = r.Run(fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		logger.Fatal("服务器启动失败", zap.Error(err))
	}
}
