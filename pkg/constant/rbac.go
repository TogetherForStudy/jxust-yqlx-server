package constant

// Role Tags
const (
	RoleTagUserBasic    = "user_basic"
	RoleTagUserActive   = "user_active"
	RoleTagUserVerified = "user_verified"
	RoleTagOperator     = "operator"
	RoleTagAdmin        = "admin"
)

// Permission Tags
const (
	PermissionUserGet                   = "user.get"                     // basic_user
	PermissionUserUpdate                = "user.update"                  // basic_user
	PermissionOSSTokenGet               = "oss.token.get"                // basic_user
	PermissionReviewCreate              = "review.create"                // basic_user
	PermissionReviewGetSelf             = "review.get.self"              // basic_user
	PermissionCourseTableGet            = "coursetable.get"              // basic_user
	PermissionCourseTableClassSearch    = "coursetable.class.search"     // basic_user
	PermissionCourseTableClassUpdate    = "coursetable.class.update.own" // basic_user
	PermissionCourseTableClassUpdateAll = "coursetable.class.update.all" // active_user
	PermissionCourseTableUpdate         = "coursetable.update"           // basic_user
	PermissionFailRate                  = "failrate"                     // basic_user
	PermissionPointGet                  = "point.get"                    // basic_user
	PermissionPointSpend                = "point.spend"                  // basic_user
	PermissionStatisticGet              = "statistic.get"                // basic_user

	PermissionContributionGet     = "contribution.get"      // basic_user
	PermissionContributionCreate  = "contribution.create"   // basic_user
	PermissionCountdown           = "countdown"             // basic_user
	PermissionStudyTask           = "studytask"             // basic_user
	PermissionMaterialGet         = "material.get"          // basic_user
	PermissionMaterialRate        = "material.rate"         // basic_user
	PermissionMaterialDownload    = "material.download"     // basic_user
	PermissionMaterialCategoryGet = "material.category.get" // basic_user
	PermissionQuestion            = "question"              // basic_user
	PermissionPomodoro            = "pomodoro"              // basic_user
	PermissionDictionary          = "dictionary"            // basic_user
	PermissionChatStudy           = "chat.study"            // basic_user

	PermissionReviewManage               = "review.manage"
	PermissionCourseTableManage          = "coursetable.manage"
	PermissionHeroManage                 = "hero.manage"
	PermissionConfigManage               = "config.manage"
	PermissionPointManage                = "point.manage"
	PermissionContributionManage         = "contribution.manage"    // operator
	PermissionNotificationGet            = "notification.get"       // basic_user
	PermissionNotificationGetAdmin       = "notification.get.admin" // operator
	PermissionNotificationCreate         = "notification.create"    // operator
	PermissionNotificationPublish        = "notification.publish"   // operator
	PermissionNotificationUpdate         = "notification.update"    // operator
	PermissionNotificationApprove        = "notification.approve"   // operator
	PermissionNotificationSchedule       = "notification.schedule"  // operator
	PermissionNotificationPin            = "notification.pin"
	PermissionNotificationDelete         = "notification.delete"
	PermissionNotificationPublishAdmin   = "notification.publish.admin"
	PermissionNotificationCategoryManage = "notification.category.manage"
	PermissionFeatureManage              = "feature.manage"
	PermissionUserManage                 = "user.manage"
	PermissionMaterialManage             = "material.manage"
	PermissionS3Manage                   = "s3.manage"
)
