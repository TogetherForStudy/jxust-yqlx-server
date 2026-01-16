package router

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/config"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/middleware"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/cache"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func NewRouter(ctx context.Context, db *gorm.DB, cfg *config.Config) *gin.Engine {
	r := gin.Default()
	ca := cache.GlobalCache

	// 添加中间件
	r.Use(middleware.CORS())
	r.Use(middleware.Logger())
	r.Use(gin.Recovery())

	// 初始化服务
	rbacService := services.NewRBACService(db)
	if cfg.InitRbac {
		if err := rbacService.SeedDefaults(context.Background()); err != nil {
			logger.Warnf("RBAC seed 初始化失败: %v", err)
		}
	}

	authService := services.NewAuthService(db, cfg, rbacService)
	pointsService := services.NewPointsService(db)
	reviewService := services.NewReviewService(db, pointsService)
	courseTableService := services.NewCourseTableService(db)
	failRateService := services.NewFailRateService(db)
	heroService := services.NewHeroService(db)
	configService := services.NewConfigService(db)
	ossService := services.NewOSSService(cfg)
	s3Service := services.NewS3Service(db, cfg)
	notificationService := services.NewNotificationService(db, rbacService)
	contributionService := services.NewContributionService(db, pointsService)
	countdownService := services.NewCountdownService(db)
	studyTaskService := services.NewStudyTaskService(db)
	featureService := services.NewFeatureService(db)
	materialService := services.NewMaterialService(db)
	questionService := services.NewQuestionService(db)
	go questionService.StartSyncWorker(ctx)
	statService := services.NewStatService()

	// 初始化处理器
	rbacHandler := handlers.NewRBACHandler(rbacService)
	authHandler := handlers.NewAuthHandler(authService, rbacService)
	reviewHandler := handlers.NewReviewHandler(reviewService)
	courseTableHandler := handlers.NewCourseTableHandler(courseTableService)
	failRateHandler := handlers.NewFailRateHandler(failRateService)
	heroHandler := handlers.NewHeroHandler(heroService)
	configHandler := handlers.NewConfigHandler(configService)
	ossHandler := handlers.NewOSSHandler(ossService)
	storeHandler := handlers.NewStoreHandler(s3Service)
	notificationHandler := handlers.NewNotificationHandler(notificationService)
	contributionHandler := handlers.NewContributionHandler(contributionService)
	pointsHandler := handlers.NewPointsHandler(pointsService)
	countdownHandler := handlers.NewCountdownHandler(countdownService)
	studyTaskHandler := handlers.NewStudyTaskHandler(studyTaskService)
	featureHandler := handlers.NewFeatureHandler(featureService)
	materialHandler := handlers.NewMaterialHandler(materialService)
	questionHandler := handlers.NewQuestionHandler(questionService)
	statHandler := handlers.NewStatHandler(statService)

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

		// 需要认证的路由
		authorized := v0.Group("/")
		authorized.Use(middleware.AuthMiddleware(cfg))
		authorized.Use(middleware.RequestRecordMiddleware(db, pointsService)) // 通用请求记录中间件（每日登录、在线人数统计）
		{
			// 用户（需认证）
			user := authorized.Group("/user")
			{
				user.GET("/profile", middleware.RequirePermission(rbacService, models.PermissionUserGet), authHandler.GetProfile)
				user.PUT("/profile", middleware.RequirePermission(rbacService, models.PermissionUserUpdate), authHandler.UpdateProfile)
				user.GET("/features", middleware.RequirePermission(rbacService, models.PermissionUserGet), featureHandler.GetUserFeatures) // 获取用户功能列表
			}
			// OSS/CDN Token （需认证）
			oss := authorized.Group("/oss")
			{
				oss.POST("/token", middleware.RequirePermission(rbacService, models.PermissionOSSTokenGet), ossHandler.GetToken)
			}
			// 评价（需认证）
			authReviews := authorized.Group("/reviews")
			{
				authReviews.POST("/", middleware.RequirePermission(rbacService, models.PermissionReviewCreate), middleware.IdempotencyRecommended(ca), reviewHandler.CreateReview)
				authReviews.GET("/user", middleware.RequirePermission(rbacService, models.PermissionReviewGetSelf), reviewHandler.GetUserReviews)

				// 管理员
				adminReviews := authReviews.Group("")
				adminReviews.Use(middleware.RequirePermission(rbacService, models.PermissionReviewManage))
				{
					adminReviews.GET("/", reviewHandler.GetReviews)
					adminReviews.POST("/:id/approve", middleware.IdempotencyRecommended(ca), reviewHandler.ApproveReview)
					adminReviews.POST("/:id/reject", middleware.IdempotencyRecommended(ca), reviewHandler.RejectReview)
					adminReviews.DELETE("/:id", reviewHandler.DeleteReview)
				}
			}

			// 课程表（需认证）
			courseTable := authorized.Group("/coursetable")
			{
				courseTable.GET("/", middleware.RequirePermission(rbacService, models.PermissionCourseTableGet), courseTableHandler.GetCourseTable)               // 获取用户课程表
				courseTable.GET("/search", middleware.RequirePermission(rbacService, models.PermissionCourseTableClassSearch), courseTableHandler.SearchClasses)  // 搜索班级
				courseTable.PUT("/class", middleware.RequirePermission(rbacService, models.PermissionCourseTableClassUpdate), courseTableHandler.UpdateUserClass) // 更新用户班级
				courseTable.PUT("/", middleware.RequirePermission(rbacService, models.PermissionCourseTableUpdate), courseTableHandler.EditCourseCell)            // 编辑个人课表的单个格子

				// 管理员
				adminCourseTable := courseTable.Group("")
				adminCourseTable.Use(middleware.RequirePermission(rbacService, models.PermissionCourseTableManage))
				{
					adminCourseTable.POST("/reset/:id", courseTableHandler.ResetUserBindCountToOne)
				}
			}

			// 挂科率（需认证）
			failrate := authorized.Group("/failrate")
			{
				failrate.GET("/search", middleware.RequirePermission(rbacService, models.PermissionFailRate), failRateHandler.SearchFailRate)
				failrate.GET("/rand", middleware.RequirePermission(rbacService, models.PermissionFailRate), failRateHandler.RandFailRate)
			}

			// 英雄榜（管理员）
			heroes := authorized.Group("/heroes")
			{
				adminHeroes := heroes.Group("")
				adminHeroes.Use(middleware.RequirePermission(rbacService, models.PermissionHeroManage))
				{
					adminHeroes.POST("/", middleware.IdempotencyRecommended(ca), heroHandler.Create)
					adminHeroes.PUT("/:id", heroHandler.Update)
					adminHeroes.DELETE("/:id", heroHandler.Delete)
					adminHeroes.GET("/search", heroHandler.SearchHeroes)
				}
			}

			// 配置写（管理员）
			configWrite := authorized.Group("/config")
			{

				adminConfig := configWrite.Group("")
				adminConfig.Use(middleware.RequirePermission(rbacService, models.PermissionConfigManage))
				{
					adminConfig.POST("/", middleware.IdempotencyRecommended(ca), configHandler.Create)
					adminConfig.PUT("/:key", middleware.IdempotencyRecommended(ca), configHandler.Update)
					adminConfig.DELETE("/:key", middleware.IdempotencyRecommended(ca), configHandler.Delete)
					adminConfig.GET("/search", configHandler.SearchConfigs)
				}
			}

			// 存储（需认证）
			store := authorized.Group("/store")
			{
				store.GET("/:resource_id/url", storeHandler.GetFileURL)
				store.GET("/:resource_id/stream", storeHandler.GetFileStream)

				adminStore := store.Group("")
				adminStore.Use(middleware.RequirePermission(rbacService, models.PermissionS3Manage))
				{
					adminStore.POST("", storeHandler.UploadFile)
					adminStore.DELETE("/:resource_id", storeHandler.DeleteFile)
					adminStore.GET("/list", storeHandler.ListFiles)
					adminStore.GET("/expired", storeHandler.ListExpiredFiles)
				}
			}

			// 积分（需认证）
			points := authorized.Group("/points")
			{
				points.GET("/", middleware.RequirePermission(rbacService, models.PermissionPointGet), pointsHandler.GetUserPoints)
				points.GET("/transactions", middleware.RequirePermission(rbacService, models.PermissionPointGet), pointsHandler.GetPointsTransactions)
				points.POST("/spend", middleware.RequirePermission(rbacService, models.PermissionPointSpend), middleware.IdempotencyRecommended(ca), pointsHandler.SpendPoints)
				points.GET("/stats", middleware.RequirePermission(rbacService, models.PermissionPointGet), pointsHandler.GetUserPointsStats)

				// 管理员
				adminPoints := points.Group("")
				adminPoints.Use(middleware.RequirePermission(rbacService, models.PermissionPointManage))
				{
					adminPoints.POST("/grant", middleware.IdempotencyRecommended(ca), pointsHandler.GrantPoints) // 管理员手动赋予积分
				}
			}

			// 投稿（需认证）
			contributions := authorized.Group("/contributions")
			{
				contributions.POST("/", middleware.RequirePermission(rbacService, models.PermissionContributionCreate), middleware.IdempotencyRecommended(ca), contributionHandler.CreateContribution)
				contributions.GET("/", middleware.RequirePermission(rbacService, models.PermissionContributionGet), contributionHandler.GetContributions)
				contributions.GET("/:id", middleware.RequirePermission(rbacService, models.PermissionContributionGet), contributionHandler.GetContributionByID)
				contributions.GET("/stats", middleware.RequirePermission(rbacService, models.PermissionContributionGet), contributionHandler.GetUserContributionStats)

				// 管理员
				adminContributions := contributions.Group("")
				adminContributions.Use(middleware.RequirePermission(rbacService, models.PermissionContributionManage))
				{
					adminContributions.POST("/:id/review", middleware.IdempotencyRecommended(ca), contributionHandler.ReviewContribution) // 审核投稿（幂等性保护）
					adminContributions.GET("/stats-admin", contributionHandler.GetAdminContributionStats)                                 // 管理员投稿统计
				}
			}

			// 倒数日（需认证）
			countdowns := authorized.Group("/countdowns")
			{
				countdowns.POST("/", middleware.RequirePermission(rbacService, models.PermissionCountdown), middleware.IdempotencyRecommended(ca), countdownHandler.CreateCountdown)
				countdowns.GET("/", middleware.RequirePermission(rbacService, models.PermissionCountdown), countdownHandler.GetCountdowns)
				countdowns.GET("/:id", middleware.RequirePermission(rbacService, models.PermissionCountdown), countdownHandler.GetCountdownByID)
				countdowns.PUT("/:id", middleware.RequirePermission(rbacService, models.PermissionCountdown), middleware.IdempotencyRecommended(ca), countdownHandler.UpdateCountdown)
				countdowns.DELETE("/:id", middleware.RequirePermission(rbacService, models.PermissionCountdown), countdownHandler.DeleteCountdown)
			}

			// 学习清单（需认证）
			studyTasks := authorized.Group("/study-tasks")
			{
				studyTasks.POST("/", middleware.RequirePermission(rbacService, models.PermissionStudyTask), middleware.IdempotencyRecommended(ca), studyTaskHandler.CreateStudyTask)
				studyTasks.GET("/", middleware.RequirePermission(rbacService, models.PermissionStudyTask), studyTaskHandler.GetStudyTasks)
				studyTasks.GET("/:id", middleware.RequirePermission(rbacService, models.PermissionStudyTask), studyTaskHandler.GetStudyTaskByID)
				studyTasks.PUT("/:id", middleware.RequirePermission(rbacService, models.PermissionStudyTask), middleware.IdempotencyRecommended(ca), studyTaskHandler.UpdateStudyTask)
				studyTasks.DELETE("/:id", middleware.RequirePermission(rbacService, models.PermissionStudyTask), studyTaskHandler.DeleteStudyTask)
				studyTasks.GET("/stats", middleware.RequirePermission(rbacService, models.PermissionStudyTask), studyTaskHandler.GetStudyTaskStats)
				studyTasks.GET("/completed", middleware.RequirePermission(rbacService, models.PermissionStudyTask), studyTaskHandler.GetCompletedTasks)
			}

			// 资料（需认证）
			materials := authorized.Group("/materials")
			{
				materials.GET("/", middleware.RequirePermission(rbacService, models.PermissionMaterialGet), materialHandler.GetMaterialList)                     // 获取资料列表
				materials.GET("/top", middleware.RequirePermission(rbacService, models.PermissionMaterialGet), materialHandler.GetTopMaterials)                  // 获取热门资料
				materials.GET("/hot-words", middleware.RequirePermission(rbacService, models.PermissionMaterialGet), materialHandler.GetHotWords)                // 获取热词
				materials.GET("/search", middleware.RequirePermission(rbacService, models.PermissionMaterialGet), materialHandler.SearchMaterials)               // 搜索资料
				materials.GET("/:md5", middleware.RequirePermission(rbacService, models.PermissionMaterialGet), materialHandler.GetMaterialDetail)               // 获取资料详情
				materials.POST("/:md5/rating", middleware.RequirePermission(rbacService, models.PermissionMaterialRate), materialHandler.RateMaterial)           // 资料评分
				materials.POST("/:md5/download", middleware.RequirePermission(rbacService, models.PermissionMaterialDownload), materialHandler.DownloadMaterial) // 下载资料
			}

			// 资料分类（需认证）
			materialCategories := authorized.Group("/material-categories")
			{
				materialCategories.GET("/", middleware.RequirePermission(rbacService, models.PermissionMaterialCategoryGet), materialHandler.GetCategories) // 获取分类列表
			}

			// 通知（需认证）
			notifications := authorized.Group("/notifications")
			{
				notifications.GET("/", middleware.RequirePermission(rbacService, models.PermissionNotificationGet), notificationHandler.GetNotifications)       // 获取通知列表
				notifications.GET("/:id", middleware.RequirePermission(rbacService, models.PermissionNotificationGet), notificationHandler.GetNotificationByID) // 获取通知详情
			}

			// 通知分类（需认证）
			categories := authorized.Group("/categories")
			{
				categories.GET("/", middleware.RequirePermission(rbacService, models.PermissionNotificationGet), notificationHandler.GetCategories) // 获取所有分类
			}

			// 刷题（需认证）
			questions := authorized.Group("/questions")
			{
				questions.GET("/projects", middleware.RequirePermission(rbacService, models.PermissionQuestion), questionHandler.GetProjects)     // 获取项目列表
				questions.GET("/list", middleware.RequirePermission(rbacService, models.PermissionQuestion), questionHandler.GetQuestions)        // 获取题目ID列表
				questions.GET("/:id", middleware.RequirePermission(rbacService, models.PermissionQuestion), questionHandler.GetQuestionByID)      // 获取题目详情
				questions.POST("/study", middleware.RequirePermission(rbacService, models.PermissionQuestion), questionHandler.RecordStudy)       // 记录学习次数
				questions.POST("/practice", middleware.RequirePermission(rbacService, models.PermissionQuestion), questionHandler.SubmitPractice) // 记录做题次数
			}

			// 统计（需认证）
			stat := authorized.Group("/stat")
			{
				stat.GET("/system/online", middleware.RequirePermission(rbacService, models.PermissionStatisticGet), statHandler.GetSystemOnlineCount)               // 获取系统在线人数
				stat.GET("/project/:project_id/online", middleware.RequirePermission(rbacService, models.PermissionStatisticGet), statHandler.GetProjectOnlineCount) // 获取项目在线人数
			}

			// 通知管理（管理员）
			notificationAdmin := authorized.Group("/admin/notifications")
			{
				notificationAdmin.GET("/", middleware.RequirePermission(rbacService, models.PermissionNotificationGetAdmin), notificationHandler.GetAdminNotifications)                                                                 // 获取管理员通知列表
				notificationAdmin.GET("/stats", middleware.RequirePermission(rbacService, models.PermissionNotificationGetAdmin), notificationHandler.GetNotificationStats)                                                             // 获取通知统计信息
				notificationAdmin.GET("/:id", middleware.RequirePermission(rbacService, models.PermissionNotificationGetAdmin), notificationHandler.GetNotificationAdminByID)                                                           // 获取通知详情
				notificationAdmin.POST("/", middleware.RequirePermission(rbacService, models.PermissionNotificationCreate), middleware.IdempotencyRecommended(ca), notificationHandler.CreateNotification)                              // 创建通知（幂等性保护）
				notificationAdmin.POST("/:id/publish", middleware.RequirePermission(rbacService, models.PermissionNotificationPublish), middleware.IdempotencyRecommended(ca), notificationHandler.PublishNotification)                 // 发布通知（幂等性保护）
				notificationAdmin.PUT("/:id", middleware.RequirePermission(rbacService, models.PermissionNotificationUpdate), notificationHandler.UpdateNotification)                                                                   // 更新通知
				notificationAdmin.POST("/:id/approve", middleware.RequirePermission(rbacService, models.PermissionNotificationApprove), middleware.IdempotencyRecommended(ca), notificationHandler.ApproveNotification)                 // 审核通知（幂等性保护）
				notificationAdmin.POST("/:id/schedule", middleware.RequirePermission(rbacService, models.PermissionNotificationSchedule), middleware.IdempotencyRecommended(ca), notificationHandler.ConvertToSchedule)                 // 转换为日程（幂等性保护）
				notificationAdmin.DELETE("/:id", middleware.RequirePermission(rbacService, models.PermissionNotificationDelete), notificationHandler.DeleteNotification)                                                                // 删除通知
				notificationAdmin.POST("/:id/publish-admin", middleware.RequirePermission(rbacService, models.PermissionNotificationPublishAdmin), middleware.IdempotencyRecommended(ca), notificationHandler.PublishNotificationAdmin) // 管理员直接发布通知（跳过审核，幂等性保护）
				notificationAdmin.POST("/:id/pin", middleware.RequirePermission(rbacService, models.PermissionNotificationPin), middleware.IdempotencyRecommended(ca), notificationHandler.PinNotification)                             // 置顶通知（幂等性保护）
				notificationAdmin.POST("/:id/unpin", middleware.RequirePermission(rbacService, models.PermissionNotificationPin), middleware.IdempotencyRecommended(ca), notificationHandler.UnpinNotification)                         // 取消置顶通知（幂等性保护）
			}

			// 通知分类管理（管理员）
			categoryAdmin := authorized.Group("/admin/categories")
			categoryAdmin.Use(middleware.RequirePermission(rbacService, models.PermissionNotificationCategoryManage))
			{
				categoryAdmin.POST("/", middleware.IdempotencyRecommended(ca), notificationHandler.CreateCategory) // 创建分类（幂等性保护）
				categoryAdmin.PUT("/:id", notificationHandler.UpdateCategory)                                      // 更新分类
			}

			// 功能管理（管理员）
			featureAdmin := authorized.Group("/admin/features")
			featureAdmin.Use(middleware.RequirePermission(rbacService, models.PermissionFeatureManage))
			{
				featureAdmin.GET("", featureHandler.ListFeatures)                                                                   // 获取所有功能列表
				featureAdmin.GET("/:key", featureHandler.GetFeature)                                                                // 获取功能详情
				featureAdmin.POST("", middleware.IdempotencyRecommended(ca), featureHandler.CreateFeature)                          // 创建功能（幂等性保护）
				featureAdmin.PUT("/:key", featureHandler.UpdateFeature)                                                             // 更新功能
				featureAdmin.DELETE("/:key", featureHandler.DeleteFeature)                                                          // 删除功能
				featureAdmin.GET("/:key/whitelist", featureHandler.ListWhitelist)                                                   // 获取白名单列表
				featureAdmin.POST("/:key/whitelist", middleware.IdempotencyRecommended(ca), featureHandler.GrantFeature)            // 授予权限（幂等性保护）
				featureAdmin.POST("/:key/whitelist/batch", middleware.IdempotencyRecommended(ca), featureHandler.BatchGrantFeature) // 批量授予权限（幂等性保护）
				featureAdmin.DELETE("/:key/whitelist/:uid", featureHandler.RevokeFeature)                                           // 撤销权限
			}

			// 用户管理（管理员）
			userFeatureAdmin := authorized.Group("/admin/users")
			userFeatureAdmin.Use(middleware.RequirePermission(rbacService, models.PermissionUserManage))
			{
				userFeatureAdmin.GET("/:id/features", featureHandler.GetUserFeatureDetails) // 查看用户功能权限详情
			}

			// RBAC 管理（管理员）
			rbacAdmin := authorized.Group("/admin/rbac")
			rbacAdmin.Use(middleware.RequirePermission(rbacService, models.PermissionUserManage))
			{
				rbacAdmin.GET("/roles", rbacHandler.ListRoles)
				rbacAdmin.GET("/roles/permissions", rbacHandler.ListRolesWithPermissions) // 获取所有角色及其权限列表
				rbacAdmin.POST("/roles", rbacHandler.CreateRole)
				rbacAdmin.PUT("/roles/:id", rbacHandler.UpdateRole)
				// rbacAdmin.DELETE("/roles/:id", rbacHandler.DeleteRole)
				rbacAdmin.GET("/permissions", rbacHandler.ListPermissions)
				rbacAdmin.POST("/permissions", rbacHandler.CreatePermission)
				rbacAdmin.POST("/roles/:id/permissions", rbacHandler.UpdateRolePermissions)
				rbacAdmin.POST("/users/:id/roles", rbacHandler.UpdateUserRoles)
				rbacAdmin.GET("/users/:id/permissions", rbacHandler.GetUserPermissions)
			}

			// 资料管理（管理员）
			materialAdmin := authorized.Group("/admin")
			materialAdmin.Use(middleware.RequirePermission(rbacService, models.PermissionMaterialManage))
			{
				// 资料管理
				materialAdmin.DELETE("/materials/:md5", materialHandler.DeleteMaterial) // 删除资料

				// 资料描述管理
				materialAdmin.PUT("/material-desc/:md5", materialHandler.UpdateMaterialDesc) // 更新资料描述
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
