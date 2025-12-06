package main

import (
	"context"
	"os"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/config"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/database"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/cache"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/redis"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/router"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
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

	// 初始化Redis和缓存
	initRedisCache(cfg)

	// 设置生产模式
	if os.Getenv(constant.ENV_GIN_MODE) == "release" {
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

// initRedisCache 初始化Redis缓存
func initRedisCache(cfg *config.Config) {
	// 检查Redis配置是否有效
	if cfg.RedisHost == "" {
		logger.Warnf("Redis host not configured, idempotency feature will be disabled")
		return
	}

	// 初始化Redis客户端
	redisCli := redis.NewRedisCli(cfg)
	if redisCli == nil {
		logger.Warnf("Failed to create Redis client, idempotency feature will be disabled")
		return
	}

	// 测试Redis连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisCli.GetRedisCli().Ping(ctx).Err(); err != nil {
		logger.Warnf("Redis connection failed: %v, idempotency feature will be disabled", err)
		return
	}

	// 初始化缓存
	cache.NewRedisCache(redisCli)
	logger.Infof("Redis cache initialized successfully, idempotency feature enabled")
}
