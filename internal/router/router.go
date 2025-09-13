package router

import (
	"fmt"
	"net/http/httputil"
	"net/url"

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
	courseTableService := services.NewCourseTableService(db)
	s3Service := services.NewS3Service(db, cfg)

	// 初始化处理器
	authHandler := handlers.NewAuthHandler(authService)
	reviewHandler := handlers.NewReviewHandler(reviewService)
	courseTableHandler := handlers.NewCourseTableHandler(courseTableService)
	storeHandler := handlers.NewStoreHandler(s3Service)

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
			auth.POST("/mock-wechat-login", authHandler.MockWechatLogin)
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

				// 管理员相关路由
				adminReviews := authReviews.Group("")
				adminReviews.Use(middleware.AdminMiddleware())
				{
					adminReviews.GET("/", reviewHandler.GetReviews)
					adminReviews.POST("/:id/approve", reviewHandler.ApproveReview)
					adminReviews.POST("/:id/reject", reviewHandler.RejectReview)
					adminReviews.DELETE("/:id", reviewHandler.DeleteReview)
				}
			}

			// 课程表相关路由（需认证）
			courseTable := authorized.Group("/coursetable")
			{
				courseTable.GET("/", courseTableHandler.GetCourseTable)       // 获取用户课程表
				courseTable.GET("/search", courseTableHandler.SearchClasses)  // 搜索班级
				courseTable.PUT("/class", courseTableHandler.UpdateUserClass) // 更新用户班级
			}

			// 存储相关路由
			store := authorized.Group("/store")
			{
				store.GET("/:resource_id/url", storeHandler.GetFileURL)
				store.GET("/:resource_id/stream", storeHandler.GetFileStream)

				adminStore := store.Group("")
				adminStore.Use(middleware.AdminMiddleware())
				{
					adminStore.POST("/", storeHandler.UploadFile)
					adminStore.DELETE("/:resource_id", storeHandler.DeleteFile)
					adminStore.GET("/", storeHandler.ListFiles)
					adminStore.GET("/expired", storeHandler.ListExpiredFiles)
				}
			}
		}
	}

	// Minio proxy:
	minioProxy := r.Group(fmt.Sprintf("/%s", cfg.BucketName))
	minioProxy.Use(middleware.RequestID())
	// todo: 这里需要添加认证中间件，确保只有授权用户可以访问MinIO资源
	//  然后做下资源区分，哪些是公开资源，哪些是私有资源
	//  公开资源可以直接访问，私有资源需要做权限校验
	{
		scheme := "http"
		if cfg.MinIO.MinIOUseSSL {
			scheme = "https"
		}
		remote, err := url.Parse(fmt.Sprintf("%s://%s", scheme, cfg.MinIO.MinIOEndpoint))
		if err != nil {
			panic(err)
		}
		proxy := httputil.NewSingleHostReverseProxy(remote)
		minioProxy.Any("/*proxyPath", func(c *gin.Context) {
			proxy.ServeHTTP(c.Writer, c.Request)
		})
	}
	return r
}
