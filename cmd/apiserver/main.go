package main

import (
	"os"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/config"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/database"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/router"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		logger.Warnf(".env file not found: %v", err)
	}

	// 初始化配置
	cfg := config.NewConfig()

	// 初始化数据库
	db, err := database.NewDatabase(cfg)
	if err != nil {
		logger.Fatalf("Failed to initialize database: %v", err)
	}

	// 自动迁移数据库表
	if err := database.AutoMigrate(db); err != nil {
		logger.Fatalf("Failed to migrate database: %v", err)
	}

	// 设置生产模式
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 初始化路由
	r := router.NewRouter(db, cfg)

	// 启动服务器
	port := cfg.ServerPort
	if port == "" {
		port = "8085"
	}

	logger.Infof("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		logger.Fatalf("Failed to start server: %v", err)
	}
}
