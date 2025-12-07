package router

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/config"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/middleware"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/cache"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func NewRouter(db *gorm.DB, cfg *config.Config) *gin.Engine {
	r := gin.Default()
	ca := cache.GlobalCache

	// 添加中间件
	r.Use(middleware.CORS())
	r.Use(middleware.Logger())
	r.Use(gin.Recovery())

	// 初始化服务
	authService := services.NewAuthService(db, cfg)
	reviewService := services.NewReviewService(db)
	courseTableService := services.NewCourseTableService(db)
	failRateService := services.NewFailRateService(db)
	heroService := services.NewHeroService(db)
	configService := services.NewConfigService(db)
	ossService := services.NewOSSService(cfg)
	s3Service := services.NewS3Service(db, cfg)

	notificationService := services.NewNotificationService(db)
	contributionService := services.NewContributionService(db)
	pointsService := services.NewPointsService(db)
	countdownService := services.NewCountdownService(db)
	studyTaskService := services.NewStudyTaskService(db)

	// 初始化处理器
	authHandler := handlers.NewAuthHandler(authService)
	reviewHandler := handlers.NewReviewHandler(reviewService)
	courseTableHandler := handlers.NewCourseTableHandler(courseTableService)
	failRateHandler := handlers.NewFailRateHandler(failRateService)
	heroHandler := handlers.NewHeroHandler(heroService)
	configHandler := handlers.NewConfigHandler(configService)
	ossHandler := handlers.NewOSSHandler(ossService)
	storeHandler := handlers.NewStoreHandler(s3Service)

	// 新增处理器
	notificationHandler := handlers.NewNotificationHandler(notificationService)
	contributionHandler := handlers.NewContributionHandler(contributionService)
	pointsHandler := handlers.NewPointsHandler(pointsService)
	countdownHandler := handlers.NewCountdownHandler(countdownService)
	studyTaskHandler := handlers.NewStudyTaskHandler(studyTaskService)

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "GoJxust API is running",
		})
	})

	// API路由组
	api := r.Group("/api")

	// MCP endpoint for LLM tool calling
	mcpGroup := api.Group("/mcp")
	{
		mcpGroup.Use(middleware.RequestID())
		mcpGroup.Use(middleware.AuthMiddleware(cfg))
		mcpHandler := handlers.NewMCPHandler(
			heroService,
			notificationService,
			authService,
			reviewService,
			courseTableService,
			failRateService,
			countdownService,
			studyTaskService,
		)
		mcpGroup.Any("", mcpHandler.Handle)
	}

	v0 := api.Group("/v0")
	v0.Use(middleware.RequestID())
	{ // 认证相关路由
		auth := v0.Group("/auth")
		{
			auth.POST("/wechat-login", authHandler.WechatLogin)
			if os.Getenv(constant.ENV_GIN_MODE) != "release" {
				auth.POST("/mock-wechat-login", authHandler.MockWechatLogin)
			}
		}

		// 评价相关路由（公开查询）
		reviews := v0.Group("/reviews")
		{
			reviews.GET("/teacher", reviewHandler.GetReviewsByTeacher)
		}

		// 配置相关路由（公开查询）
		configs := v0.Group("/config")
		{
			configs.GET("/:key", configHandler.GetByKey)
		}

		// 英雄榜相关路由（公开查询）
		heroes := v0.Group("/heroes")
		{
			heroes.GET("/", heroHandler.ListAll)
		}

		// 通知相关路由（公开查询）
		notifications := v0.Group("/notifications")
		{
			notifications.GET("/", notificationHandler.GetNotifications)       // 获取通知列表
			notifications.GET("/:id", notificationHandler.GetNotificationByID) // 获取通知详情
		}

		// 通知分类路由（公开查询）
		categories := v0.Group("/categories")
		{
			categories.GET("/", notificationHandler.GetCategories) // 获取所有分类
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
			// OSS/CDN Token （需认证）
			oss := authorized.Group("/oss")
			{
				oss.POST("/token", ossHandler.GetToken)
			}
			// 评价相关路由（需认证）
			authReviews := authorized.Group("/reviews")
			{
				authReviews.POST("/", middleware.IdempotencyRecommended(ca), reviewHandler.CreateReview)
				authReviews.GET("/user", reviewHandler.GetUserReviews)

				// 管理员相关路由
				adminReviews := authReviews.Group("")
				adminReviews.Use(middleware.RequireRole(2))
				{
					adminReviews.GET("/", reviewHandler.GetReviews)
					adminReviews.POST("/:id/approve", middleware.IdempotencyRecommended(ca), reviewHandler.ApproveReview)
					adminReviews.POST("/:id/reject", middleware.IdempotencyRecommended(ca), reviewHandler.RejectReview)
					adminReviews.DELETE("/:id", reviewHandler.DeleteReview)
				}
			}

			// 课程表相关路由（需认证）
			courseTable := authorized.Group("/coursetable")
			{
				courseTable.GET("/", courseTableHandler.GetCourseTable)       // 获取用户课程表
				courseTable.GET("/search", courseTableHandler.SearchClasses)  // 搜索班级
				courseTable.PUT("/class", courseTableHandler.UpdateUserClass) // 更新用户班级
				courseTable.PUT("/", courseTableHandler.EditCourseCell)       // 编辑个人课表的单个格子

				// 管理员-用户绑定次数维护
				adminCourseTable := courseTable.Group("")
				adminCourseTable.Use(middleware.RequireRole(2))
				{
					adminCourseTable.POST("/reset/:id", courseTableHandler.ResetUserBindCountToOne)
				}
			}

			// 挂科率（需认证）
			failrate := authorized.Group("/failrate")
			{
				failrate.GET("/search", failRateHandler.SearchFailRate)
				failrate.GET("/rand", failRateHandler.RandFailRate)
			}

			// heroes（需认证）
			heroes := authorized.Group("/heroes")
			{
				// 仅管理员可改写
				adminHeroes := heroes.Group("")
				adminHeroes.Use(middleware.RequireRole(2))
				{
					adminHeroes.POST("/", middleware.IdempotencyRecommended(ca), heroHandler.Create)
					adminHeroes.PUT("/:id", heroHandler.Update)
					adminHeroes.DELETE("/:id", heroHandler.Delete)
					adminHeroes.GET("/search", heroHandler.SearchHeroes)
				}
			}

			// 配置写（需管理员）
			configWrite := authorized.Group("/config")
			{

				adminConfig := configWrite.Group("")
				adminConfig.Use(middleware.RequireRole(2))
				{
					adminConfig.POST("/", middleware.IdempotencyRecommended(ca), configHandler.Create)
					adminConfig.PUT("/:key", middleware.IdempotencyRecommended(ca), configHandler.Update)
					adminConfig.DELETE("/:key", middleware.IdempotencyRecommended(ca), configHandler.Delete)
					adminConfig.GET("/search", configHandler.SearchConfigs)
				}
			}

			// 存储相关路由
			store := authorized.Group("/store")
			{
				store.GET("/:resource_id/url", storeHandler.GetFileURL)
				store.GET("/:resource_id/stream", storeHandler.GetFileStream)

				adminStore := store.Group("")
				adminStore.Use(middleware.RequireRole(2))
				{
					adminStore.POST("", storeHandler.UploadFile)
					adminStore.DELETE("/:resource_id", storeHandler.DeleteFile)
					adminStore.GET("/list", storeHandler.ListFiles)
					adminStore.GET("/expired", storeHandler.ListExpiredFiles)
				}
			}

			// 积分相关路由（需认证）
			points := authorized.Group("/points")
			{
				points.GET("/", pointsHandler.GetUserPoints)                                            // 获取用户积分
				points.GET("/transactions", pointsHandler.GetPointsTransactions)                        // 获取积分交易记录
				points.POST("/spend", middleware.IdempotencyRecommended(ca), pointsHandler.SpendPoints) // 消费积分（幂等性保护）
				points.GET("/stats", pointsHandler.GetUserPointsStats)                                  // 获取积分统计
			}

			// 投稿相关路由（需认证）
			contributions := authorized.Group("/contributions")
			{
				contributions.POST("/", middleware.IdempotencyRecommended(ca), contributionHandler.CreateContribution) // 创建投稿（幂等性保护）
				contributions.GET("/", contributionHandler.GetContributions)                                           // 获取投稿列表
				contributions.GET("/:id", contributionHandler.GetContributionByID)                                     // 获取投稿详情
				contributions.GET("/stats", contributionHandler.GetUserContributionStats)                              // 投稿统计

				// 管理员/运营专用路由
				adminContributions := contributions.Group("")
				adminContributions.Use(middleware.RequireRole(2, 3))
				{
					adminContributions.POST("/:id/review", middleware.IdempotencyRecommended(ca), contributionHandler.ReviewContribution) // 审核投稿（幂等性保护）
					adminContributions.GET("/stats-admin", contributionHandler.GetAdminContributionStats)                                 // 管理员投稿统计
				}
			}

			// 倒数日相关路由（需认证）
			countdowns := authorized.Group("/countdowns")
			{
				countdowns.POST("/", middleware.IdempotencyRecommended(ca), countdownHandler.CreateCountdown)   // 创建倒数日（幂等性保护）
				countdowns.GET("/", countdownHandler.GetCountdowns)                                             // 获取倒数日列表
				countdowns.GET("/:id", countdownHandler.GetCountdownByID)                                       // 获取倒数日详情
				countdowns.PUT("/:id", middleware.IdempotencyRecommended(ca), countdownHandler.UpdateCountdown) // 更新倒数日
				countdowns.DELETE("/:id", countdownHandler.DeleteCountdown)                                     // 删除倒数日
			}

			// 学习清单相关路由（需认证）
			studyTasks := authorized.Group("/study-tasks")
			{
				studyTasks.POST("/", middleware.IdempotencyRecommended(ca), studyTaskHandler.CreateStudyTask)   // 创建学习任务（幂等性保护）
				studyTasks.GET("/", studyTaskHandler.GetStudyTasks)                                             // 获取任务列表
				studyTasks.GET("/:id", studyTaskHandler.GetStudyTaskByID)                                       // 获取任务详情
				studyTasks.PUT("/:id", middleware.IdempotencyRecommended(ca), studyTaskHandler.UpdateStudyTask) // 更新任务
				studyTasks.DELETE("/:id", studyTaskHandler.DeleteStudyTask)                                     // 删除任务
				studyTasks.GET("/stats", studyTaskHandler.GetStudyTaskStats)                                    // 获取统计
				studyTasks.GET("/completed", studyTaskHandler.GetCompletedTasks)                                // 已完成的任务
			}

			// 通知管理路由（需要运营权限）
			notificationAdmin := authorized.Group("/admin/notifications")
			notificationAdmin.Use(middleware.RequireRole(2, 3))
			{
				notificationAdmin.GET("/", notificationHandler.GetAdminNotifications)                                                  // 获取管理员通知列表
				notificationAdmin.GET("/stats", notificationHandler.GetNotificationStats)                                              // 获取通知统计信息
				notificationAdmin.GET("/:id", notificationHandler.GetNotificationAdminByID)                                            // 获取通知详情
				notificationAdmin.POST("/", middleware.IdempotencyRecommended(ca), notificationHandler.CreateNotification)             // 创建通知（幂等性保护）
				notificationAdmin.POST("/:id/publish", middleware.IdempotencyRecommended(ca), notificationHandler.PublishNotification) // 发布通知（幂等性保护）
				notificationAdmin.PUT("/:id", notificationHandler.UpdateNotification)                                                  // 更新通知
				notificationAdmin.POST("/:id/approve", middleware.IdempotencyRecommended(ca), notificationHandler.ApproveNotification) // 审核通知（幂等性保护）
				notificationAdmin.POST("/:id/schedule", middleware.IdempotencyRecommended(ca), notificationHandler.ConvertToSchedule)  // 转换为日程（幂等性保护）

			}
			// 通知管理路由（需要管理员权限）
			notificationUpdate := authorized.Group("/admin/notifications")
			notificationUpdate.Use(middleware.RequireRole(2))
			{

				notificationUpdate.DELETE("/:id", notificationHandler.DeleteNotification)                                                          // 删除通知
				notificationUpdate.POST("/:id/publish-admin", middleware.IdempotencyRecommended(ca), notificationHandler.PublishNotificationAdmin) // 管理员直接发布通知（跳过审核，幂等性保护）
				notificationUpdate.POST("/:id/pin", middleware.IdempotencyRecommended(ca), notificationHandler.PinNotification)                    // 置顶通知（幂等性保护）
				notificationUpdate.POST("/:id/unpin", middleware.IdempotencyRecommended(ca), notificationHandler.UnpinNotification)                // 取消置顶通知（幂等性保护）
			}

			// 分类管理路由（需要管理员权限）
			categoryAdmin := authorized.Group("/admin/categories")
			categoryAdmin.Use(middleware.RequireRole(2))
			{
				categoryAdmin.POST("/", middleware.IdempotencyRecommended(ca), notificationHandler.CreateCategory) // 创建分类（幂等性保护）
				categoryAdmin.PUT("/:id", notificationHandler.UpdateCategory)                                      // 更新分类
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
		// MinIO会根据Host头来验证签名。
		// 为了确保签名验证通过，需要重写请求的Host头，使其与MinIO原始的主机名匹配。
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			req.Host = remote.Host
		}

		minioProxy.Any("/*proxyPath", func(c *gin.Context) {
			proxy.ServeHTTP(c.Writer, c.Request)
		})
	}
	return r
}
