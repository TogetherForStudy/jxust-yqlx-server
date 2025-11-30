package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gorm.io/datatypes"
)

// MCPHandler handles MCP protocol requests for LLM tool calling
type MCPHandler struct {
	server  *mcp.Server
	handler *mcp.StreamableHTTPHandler
}

// mcpServices holds all services needed by MCP tools
type mcpServices struct {
	heroService         *services.HeroService
	notificationService *services.NotificationService
	authService         *services.AuthService // userinfo
	reviewService       *services.ReviewService
	courseTableService  *services.CourseTableService
	failRateService     *services.FailRateService
	countdownService    *services.CountdownService
	studyTaskService    *services.StudyTaskService
}

// global services reference for MCP tool handlers
var _mcpSvc *mcpServices

// NewMCPHandler creates a new MCP handler with GoJxust service tools
func NewMCPHandler(
	heroService *services.HeroService,
	notificationService *services.NotificationService,
	authService *services.AuthService,
	reviewService *services.ReviewService,
	courseTableService *services.CourseTableService,
	failRateService *services.FailRateService,
	countdownService *services.CountdownService,
	studyTaskService *services.StudyTaskService,
) *MCPHandler {
	// Store services globally for tool handlers
	_mcpSvc = &mcpServices{
		heroService:         heroService,
		notificationService: notificationService,
		authService:         authService,
		reviewService:       reviewService,
		courseTableService:  courseTableService,
		failRateService:     failRateService,
		countdownService:    countdownService,
		studyTaskService:    studyTaskService,
	}

	// Create MCP server with GoJxust implementation info
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "gojxust-mcp-server",
		Title:   "江理一起来学智能助理 MCP 服务，提供各种校园服务、学习类服务工具调用接口",
		Version: "0.1.0",
	}, &mcp.ServerOptions{HasTools: true})

	// Register all tools
	registerMCPTools(server)

	// Create streamable HTTP handler
	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, nil)

	return &MCPHandler{
		server:  server,
		handler: handler,
	}
}

// Handle processes MCP requests via Gin context
func (h *MCPHandler) Handle(c *gin.Context) {
	// Get user info from Gin context (set by AuthMiddleware)
	userID := helper.GetUserID(c)
	userRole := helper.GetUserRole(c)

	// Inject user info into request context for MCP tool handlers
	ctx := context.WithValue(c.Request.Context(), mcpUserIDKey, userID)
	ctx = context.WithValue(ctx, mcpUserRoleKey, userRole)

	h.handler.ServeHTTP(c.Writer, c.Request.WithContext(ctx))
}

// Context keys for user info
type mcpContextKey string

const (
	mcpUserIDKey   mcpContextKey = "mcp_user_id"
	mcpUserRoleKey mcpContextKey = "mcp_user_role"
)

func getUserFromContext(ctx context.Context) (uint, models.UserRole) {
	userID, _ := ctx.Value(mcpUserIDKey).(uint)
	userRole, _ := ctx.Value(mcpUserRoleKey).(uint8)
	return userID, models.UserRole(userRole)
}

// registerMCPTools registers all MCP tools
func registerMCPTools(server *mcp.Server) {
	// 1. List heroes
	mcp.AddTool(server, &mcp.Tool{
		Name:        "listHeroes",
		Description: "列出英雄榜（列出所有为项目贡献代码、贡献学习资料的贡献者） - 获取所有显示的英雄名单",
	}, handleListHeroes)

	// 2. Notifications tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "notifications",
		Description: "通知管理 - 获取通知列表或获取单个通知详情",
	}, handleNotifications)

	// 3. User profile tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "userProfile",
		Description: "用户信息管理 - 获取或更新用户信息（昵称、真名、学院、年级、班级）",
	}, handleUserProfile)

	// 4. Teacher review tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "teacherReview",
		Description: "教师评价 - 创建教师评价或获取教师评价列表",
	}, handleTeacherReview)

	// 5. Course table tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "getCourseTable",
		Description: "获取用户课程表",
	}, handleGetCourseTable)

	// 6. Edit course cell tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "editCourseCell",
		Description: "编辑个人课程表中的单个格子",
	}, handleEditCourseCell)

	// 7. Fail rate tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "queryFailRate",
		Description: "查询挂科率 - 如果不指定课程名则随机返回10条挂科率数据",
	}, handleQueryFailRate)

	// 8. Countdown tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "countdown",
		Description: "倒数日管理 - 创建、获取、更新、删除倒数日",
	}, handleCountdown)

	// 9. Study task tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "studyTask",
		Description: "学习清单管理 - 创建、获取、更新、删除、统计学习任务",
	}, handleStudyTask)
}

// ============== Tool Parameter Structs ==============

// ListHeroesParams - no params needed
type ListHeroesParams struct{}

// NotificationsParams for notifications tool
type NotificationsParams struct {
	Action string `json:"action" jsonschema:"操作类型: list(获取列表) 或 get(获取详情)"`
	ID     uint   `json:"id,omitempty" jsonschema:"通知ID，当action为get时必填"`
	Page   int    `json:"page,omitempty" jsonschema:"页码，默认1"`
	Size   int    `json:"size,omitempty" jsonschema:"每页数量，默认20"`
}

// UserProfileParams for user profile tool
type UserProfileParams struct {
	Action   string `json:"action" jsonschema:"操作类型: get(获取信息) 或 update(更新信息)"`
	Nickname string `json:"nickname,omitempty" jsonschema:"昵称"`
	RealName string `json:"real_name,omitempty" jsonschema:"真名"`
	College  string `json:"college,omitempty" jsonschema:"学院"`
	Major    string `json:"major,omitempty" jsonschema:"专业"`
	ClassID  string `json:"class_id,omitempty" jsonschema:"班级"`
}

// TeacherReviewParams for teacher review tool
type TeacherReviewParams struct {
	Action      string `json:"action" jsonschema:"操作类型: create(创建评价) 或 get(获取评价)"`
	TeacherName string `json:"teacher_name" jsonschema:"教师姓名，当action为get或create时必填"`
	Campus      string `json:"campus,omitempty" jsonschema:"校区，当action为create时必填"`
	CourseName  string `json:"course_name,omitempty" jsonschema:"课程名称，当action为create时必填"`
	Content     string `json:"content,omitempty" jsonschema:"评价内容，当action为create时必填，最多200字"`
	Attitude    uint8  `json:"attitude,omitempty" jsonschema:"教师态度评分: 1好 2一般 3差，当action为create时必填"`
	Page        int    `json:"page,omitempty" jsonschema:"页码，默认1"`
	Size        int    `json:"size,omitempty" jsonschema:"每页数量，默认10"`
}

// GetCourseTableParams for getting course table
type GetCourseTableParams struct {
	Semester string `json:"semester" jsonschema:"学期，如2024-2025-1"`
}

// EditCourseCellParams for editing course cell
type EditCourseCellParams struct {
	Semester string `json:"semester" jsonschema:"学期，如2024-2025-1"`
	Index    string `json:"index" jsonschema:"格子索引，1到35之间的字符串"`
	Value    any    `json:"value" jsonschema:"格子数据"`
}

// QueryFailRateParams for querying fail rate
type QueryFailRateParams struct {
	Keyword string `json:"keyword,omitempty" jsonschema:"课程名关键词，不填则随机返回10条"`
	Page    int    `json:"page,omitempty" jsonschema:"页码，默认1"`
	Size    int    `json:"size,omitempty" jsonschema:"每页数量，默认10"`
}

// CountdownParams for countdown tool
type CountdownParams struct {
	Action      string `json:"action" jsonschema:"操作类型: create/list/get/update/delete"`
	ID          uint   `json:"id,omitempty" jsonschema:"倒数日ID，当action为get/update/delete时必填"`
	Title       string `json:"title,omitempty" jsonschema:"标题，当action为create时必填"`
	Description string `json:"description,omitempty" jsonschema:"描述"`
	TargetDate  string `json:"target_date,omitempty" jsonschema:"目标日期，格式YYYY-MM-DD，当action为create时必填"`
}

// StudyTaskParams for study task tool
type StudyTaskParams struct {
	Action      string `json:"action" jsonschema:"操作类型: create/list/get/update/delete/stats"`
	ID          uint   `json:"id,omitempty" jsonschema:"任务ID，当action为get/update/delete时必填"`
	Title       string `json:"title,omitempty" jsonschema:"任务标题，当action为create时必填"`
	Description string `json:"description,omitempty" jsonschema:"任务描述"`
	DueDate     string `json:"due_date,omitempty" jsonschema:"截止日期，格式YYYY-MM-DD HH:MM"`
	Priority    *uint8 `json:"priority,omitempty" jsonschema:"优先级: 1高 2中 3低"`
	Status      *uint8 `json:"status,omitempty" jsonschema:"状态: 1待完成 2已完成"`
	Page        int    `json:"page,omitempty" jsonschema:"页码，默认1"`
	Size        int    `json:"size,omitempty" jsonschema:"每页数量，默认20"`
}

// ============== Tool Handlers ==============

func handleListHeroes(ctx context.Context, req *mcp.CallToolRequest, params *ListHeroesParams) (*mcp.CallToolResult, any, error) {
	heroes, err := _mcpSvc.heroService.ListAll()
	if err != nil {
		return nil, nil, fmt.Errorf("获取英雄榜失败: %w", err)
	}

	data, _ := json.Marshal(heroes)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}

func handleNotifications(ctx context.Context, req *mcp.CallToolRequest, params *NotificationsParams) (*mcp.CallToolResult, any, error) {
	switch params.Action {
	case "list":
		page := params.Page
		if page <= 0 {
			page = 1
		}
		size := params.Size
		if size <= 0 {
			size = 20
		}
		result, err := _mcpSvc.notificationService.GetNotifications(&request.GetNotificationsRequest{
			Page: page,
			Size: size,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("获取通知列表失败: %w", err)
		}
		data, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil

	case "get":
		if params.ID == 0 {
			return nil, nil, fmt.Errorf("请提供通知ID")
		}
		result, err := _mcpSvc.notificationService.GetNotificationByID(params.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("获取通知详情失败: %w", err)
		}
		data, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil

	default:
		return nil, nil, fmt.Errorf("不支持的操作: %s", params.Action)
	}
}

func handleUserProfile(ctx context.Context, req *mcp.CallToolRequest, params *UserProfileParams) (*mcp.CallToolResult, any, error) {
	userID, _ := getUserFromContext(ctx)
	if userID == 0 {
		return nil, nil, fmt.Errorf("用户未认证")
	}

	switch params.Action {
	case "get":
		user, err := _mcpSvc.authService.GetUserByID(userID)
		if err != nil {
			return nil, nil, fmt.Errorf("获取用户信息失败: %w", err)
		}
		result := map[string]any{
			"id":        user.ID,
			"nickname":  user.Nickname,
			"real_name": user.RealName,
			"college":   user.College,
			"major":     user.Major,
			"class_id":  user.ClassID,
		}
		data, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil

	case "update":
		profile := &models.User{
			Nickname: params.Nickname,
			RealName: params.RealName,
			College:  params.College,
			Major:    params.Major,
			ClassID:  params.ClassID,
		}
		if err := _mcpSvc.authService.UpdateUserProfile(userID, profile); err != nil {
			return nil, nil, fmt.Errorf("更新用户信息失败: %w", err)
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: `{"message": "更新成功"}`}},
		}, nil, nil

	default:
		return nil, nil, fmt.Errorf("不支持的操作: %s", params.Action)
	}
}

func handleTeacherReview(ctx context.Context, req *mcp.CallToolRequest, params *TeacherReviewParams) (*mcp.CallToolResult, any, error) {
	userID, _ := getUserFromContext(ctx)

	switch params.Action {
	case "create":
		if userID == 0 {
			return nil, nil, fmt.Errorf("用户未认证")
		}
		if params.TeacherName == "" || params.Campus == "" || params.CourseName == "" || params.Content == "" || params.Attitude == 0 {
			return nil, nil, fmt.Errorf("请提供完整的评价信息")
		}
		err := _mcpSvc.reviewService.CreateReview(userID, &request.CreateReviewRequest{
			TeacherName: params.TeacherName,
			Campus:      params.Campus,
			CourseName:  params.CourseName,
			Content:     params.Content,
			Attitude:    models.TeacherAttitude(params.Attitude),
		})
		if err != nil {
			return nil, nil, fmt.Errorf("创建评价失败: %w", err)
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: `{"message": "评价提交成功，等待审核"}`}},
		}, nil, nil

	case "get":
		if params.TeacherName == "" {
			return nil, nil, fmt.Errorf("请提供教师姓名")
		}
		page := params.Page
		if page <= 0 {
			page = 1
		}
		size := params.Size
		if size <= 0 {
			size = 10
		}
		reviews, total, err := _mcpSvc.reviewService.GetReviewsByTeacher(params.TeacherName, page, size)
		if err != nil {
			return nil, nil, fmt.Errorf("获取评价失败: %w", err)
		}
		result := map[string]any{
			"data":  reviews,
			"total": total,
			"page":  page,
			"size":  size,
		}
		data, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil

	default:
		return nil, nil, fmt.Errorf("不支持的操作: %s", params.Action)
	}
}

func handleGetCourseTable(ctx context.Context, req *mcp.CallToolRequest, params *GetCourseTableParams) (*mcp.CallToolResult, any, error) {
	userID, _ := getUserFromContext(ctx)
	if userID == 0 {
		return nil, nil, fmt.Errorf("用户未认证")
	}

	result, err := _mcpSvc.courseTableService.GetUserCourseTable(userID, params.Semester)
	if err != nil {
		return nil, nil, fmt.Errorf("获取课程表失败: %w", err)
	}

	data, _ := json.Marshal(result)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}

func handleEditCourseCell(ctx context.Context, req *mcp.CallToolRequest, params *EditCourseCellParams) (*mcp.CallToolResult, any, error) {
	userID, _ := getUserFromContext(ctx)
	if userID == 0 {
		return nil, nil, fmt.Errorf("用户未认证")
	}

	valueBytes, err := json.Marshal(params.Value)
	if err != nil {
		return nil, nil, fmt.Errorf("无效的格子数据: %w", err)
	}

	if err := _mcpSvc.courseTableService.EditUserCourseCell(userID, params.Semester, params.Index, datatypes.JSON(valueBytes)); err != nil {
		return nil, nil, fmt.Errorf("编辑课程表失败: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: `{"message": "编辑成功"}`}},
	}, nil, nil
}

func handleQueryFailRate(ctx context.Context, req *mcp.CallToolRequest, params *QueryFailRateParams) (*mcp.CallToolResult, any, error) {
	if params.Keyword == "" {
		// Random fail rate
		list, err := _mcpSvc.failRateService.Rand(10)
		if err != nil {
			return nil, nil, fmt.Errorf("查询挂科率失败: %w", err)
		}
		data, _ := json.Marshal(map[string]any{"data": list})
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	}

	page := params.Page
	if page <= 0 {
		page = 1
	}
	size := params.Size
	if size <= 0 {
		size = 10
	}

	list, total, err := _mcpSvc.failRateService.Search(params.Keyword, page, size)
	if err != nil {
		return nil, nil, fmt.Errorf("查询挂科率失败: %w", err)
	}

	result := map[string]any{
		"data":  list,
		"total": total,
		"page":  page,
		"size":  size,
	}
	data, _ := json.Marshal(result)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}

func handleCountdown(ctx context.Context, req *mcp.CallToolRequest, params *CountdownParams) (*mcp.CallToolResult, any, error) {
	userID, userRole := getUserFromContext(ctx)
	if userID == 0 {
		return nil, nil, fmt.Errorf("用户未认证")
	}

	switch params.Action {
	case "create":
		if params.Title == "" || params.TargetDate == "" {
			return nil, nil, fmt.Errorf("请提供标题和目标日期")
		}
		result, err := _mcpSvc.countdownService.CreateCountdown(userID, &request.CreateCountdownRequest{
			Title:       params.Title,
			Description: params.Description,
			TargetDate:  params.TargetDate,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("创建倒数日失败: %w", err)
		}
		data, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil

	case "list":
		result, err := _mcpSvc.countdownService.GetCountdowns(userID, userRole)
		if err != nil {
			return nil, nil, fmt.Errorf("获取倒数日列表失败: %w", err)
		}
		data, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil

	case "get":
		if params.ID == 0 {
			return nil, nil, fmt.Errorf("请提供倒数日ID")
		}
		result, err := _mcpSvc.countdownService.GetCountdownByID(params.ID, userID)
		if err != nil {
			return nil, nil, fmt.Errorf("获取倒数日详情失败: %w", err)
		}
		data, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil

	case "update":
		if params.ID == 0 {
			return nil, nil, fmt.Errorf("请提供倒数日ID")
		}
		result, err := _mcpSvc.countdownService.UpdateCountdown(params.ID, userID, &request.UpdateCountdownRequest{
			Title:       params.Title,
			Description: params.Description,
			TargetDate:  params.TargetDate,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("更新倒数日失败: %w", err)
		}
		data, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil

	case "delete":
		if params.ID == 0 {
			return nil, nil, fmt.Errorf("请提供倒数日ID")
		}
		if err := _mcpSvc.countdownService.DeleteCountdown(params.ID, userID); err != nil {
			return nil, nil, fmt.Errorf("删除倒数日失败: %w", err)
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: `{"message": "删除成功"}`}},
		}, nil, nil

	default:
		return nil, nil, fmt.Errorf("不支持的操作: %s", params.Action)
	}
}

func handleStudyTask(ctx context.Context, req *mcp.CallToolRequest, params *StudyTaskParams) (*mcp.CallToolResult, any, error) {
	userID, _ := getUserFromContext(ctx)
	if userID == 0 {
		return nil, nil, fmt.Errorf("用户未认证")
	}

	switch params.Action {
	case "create":
		if params.Title == "" {
			return nil, nil, fmt.Errorf("请提供任务标题")
		}
		priority := uint8(2) // default medium
		if params.Priority != nil {
			priority = *params.Priority
		}
		result, err := _mcpSvc.studyTaskService.CreateStudyTask(userID, &request.CreateStudyTaskRequest{
			Title:       params.Title,
			Description: params.Description,
			DueDate:     params.DueDate,
			Priority:    priority,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("创建学习任务失败: %w", err)
		}
		data, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil

	case "list":
		page := params.Page
		if page <= 0 {
			page = 1
		}
		size := params.Size
		if size <= 0 {
			size = 20
		}
		result, err := _mcpSvc.studyTaskService.GetStudyTasks(userID, &request.GetStudyTasksRequest{
			Page:     page,
			Size:     size,
			Status:   params.Status,
			Priority: params.Priority,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("获取学习任务列表失败: %w", err)
		}
		data, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil

	case "get":
		if params.ID == 0 {
			return nil, nil, fmt.Errorf("请提供任务ID")
		}
		result, err := _mcpSvc.studyTaskService.GetStudyTaskByID(params.ID, userID)
		if err != nil {
			return nil, nil, fmt.Errorf("获取学习任务详情失败: %w", err)
		}
		data, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil

	case "update":
		if params.ID == 0 {
			return nil, nil, fmt.Errorf("请提供任务ID")
		}
		result, err := _mcpSvc.studyTaskService.UpdateStudyTask(params.ID, userID, &request.UpdateStudyTaskRequest{
			Title:       params.Title,
			Description: params.Description,
			DueDate:     params.DueDate,
			Priority:    params.Priority,
			Status:      params.Status,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("更新学习任务失败: %w", err)
		}
		data, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil

	case "delete":
		if params.ID == 0 {
			return nil, nil, fmt.Errorf("请提供任务ID")
		}
		if err := _mcpSvc.studyTaskService.DeleteStudyTask(params.ID, userID); err != nil {
			return nil, nil, fmt.Errorf("删除学习任务失败: %w", err)
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: `{"message": "删除成功"}`}},
		}, nil, nil

	case "stats":
		result, err := _mcpSvc.studyTaskService.GetStudyTaskStats(userID)
		if err != nil {
			return nil, nil, fmt.Errorf("获取学习任务统计失败: %w", err)
		}
		data, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil

	default:
		return nil, nil, fmt.Errorf("不支持的操作: %s", params.Action)
	}
}
