package router

import (
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/config"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/middleware"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func NewRouter(db *gorm.DB, cfg *config.Config) *gin.Engine {
	r := gin.Default()

	// 添加中间件
	r.Use(middleware.CORS())
	r.Use(middleware.Logger())
	r.Use(gin.Recovery())

	// 初始化服务
	authService := services.NewAuthService(db, cfg)
	reviewService := services.NewReviewService(db)

	// 初始化处理器
	authHandler := handlers.NewAuthHandler(authService)
	reviewHandler := handlers.NewReviewHandler(reviewService)
	adminHandler := handlers.NewAdminHandler(reviewService)

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "GoJxust API is running",
		})
	})

	// API路由组
	api := r.Group("/api")
	v0 := api.Group("/v0")
	v0.Use(middleware.RequestID())
	{ // 认证相关路由
		auth := v0.Group("/auth")
		{
			auth.POST("/wechat-login", authHandler.WechatLogin)
			auth.POST("/mock-wechat-login", authHandler.MockWechatLogin) // 模拟微信登录接口
		}

		// 评价相关路由（公开查询）
		reviews := v0.Group("/reviews")
		{
			reviews.GET("/teacher", reviewHandler.GetReviewsByTeacher)
		}

		// 需要认证的路由
		authorized := v0.Group("/")
		authorized.Use(middleware.AuthMiddleware(cfg))
		{
			// 用户相关路由
			user := authorized.Group("/user")
			{
				user.GET("/profile", authHandler.GetProfile)
				user.PUT("/profile", authHandler.UpdateProfile)
			}

			// 评价相关路由（需认证）
			authReviews := authorized.Group("/reviews")
			{
				authReviews.POST("/", reviewHandler.CreateReview)
				authReviews.GET("/user", reviewHandler.GetUserReviews)
			}

			// 管理员路由
			admin := authorized.Group("/admin")
			admin.Use(middleware.AdminMiddleware())
			{
				// 评价管理
				adminReviews := admin.Group("/reviews")
				{
					adminReviews.GET("/", adminHandler.GetReviews)
					adminReviews.POST("/:id/approve", adminHandler.ApproveReview)
					adminReviews.POST("/:id/reject", adminHandler.RejectReview)
					adminReviews.DELETE("/:id", adminHandler.DeleteReview)
				}
			}
		}
	}

	return r
}
