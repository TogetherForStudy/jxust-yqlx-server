package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"gorm.io/datatypes"
)

// MCPHandler handles MCP protocol requests for LLM tool calling
type MCPHandler struct {
	mcpServer  *server.MCPServer
	httpServer *server.StreamableHTTPServer
}

// mcpToolHandlers holds all services and provides tool handler methods
type mcpToolHandlers struct {
	heroService         *services.HeroService
	notificationService *services.NotificationService
	authService         *services.AuthService
	reviewService       *services.ReviewService
	courseTableService  *services.CourseTableService
	failRateService     *services.FailRateService
	countdownService    *services.CountdownService
	studyTaskService    *services.StudyTaskService
}

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
	// Create tool handlers with services
	th := &mcpToolHandlers{
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
	mcpServer := server.NewMCPServer("gojxust-mcp-server", "0.1.0")

	// Register all tools
	th.registerTools(mcpServer)

	// Create streamable HTTP handler
	httpServer := server.NewStreamableHTTPServer(mcpServer)

	return &MCPHandler{
		mcpServer:  mcpServer,
		httpServer: httpServer,
	}
}

// Handle processes MCP requests via Gin context
func (h *MCPHandler) Handle(c *gin.Context) {
	// Get user info from Gin context (set by AuthMiddleware)
	userID := helper.GetUserID(c)
	if userID == 0 {
		helper.ErrorResponse(c, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	// Inject user info into request context for MCP tool handlers
	ctx := context.WithValue(c.Request.Context(), mcpContextKey(constant.MCPUserIDKey), userID)

	h.httpServer.ServeHTTP(c.Writer, c.Request.WithContext(ctx))
}

// Context keys for user info
type mcpContextKey string

func getUserFromContext(ctx context.Context) uint {
	userID, _ := ctx.Value(mcpContextKey(constant.MCPUserIDKey)).(uint)
	return userID
}

// registerTools registers all MCP tools
func (th *mcpToolHandlers) registerTools(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("listHeroes",
			mcp.WithDescription("列出英雄榜（列出所有为项目贡献代码、贡献学习资料的贡献者） - 获取所有显示的英雄名单"),
			mcp.WithInputSchema[ListHeroesParams](),
		),
		th.handleListHeroes,
	)

	s.AddTool(
		mcp.NewTool("notifications",
			mcp.WithDescription("通知管理 - 获取通知列表或获取单个通知详情"),
			mcp.WithInputSchema[NotificationsParams](),
		),
		th.handleNotifications,
	)

	s.AddTool(
		mcp.NewTool("userProfile",
			mcp.WithDescription("用户信息管理 - 获取或更新用户信息（昵称、真名、学院、专业、班级）"),
			mcp.WithInputSchema[UserProfileParams](),
		),
		th.handleUserProfile,
	)

	s.AddTool(
		mcp.NewTool("teacherReview",
			mcp.WithDescription("教师评价 - 创建教师评价或获取教师评价列表"),
			mcp.WithInputSchema[TeacherReviewParams](),
		),
		th.handleTeacherReview,
	)

	s.AddTool(
		mcp.NewTool("getCourseTable",
			mcp.WithDescription("获取用户课程表"),
			mcp.WithInputSchema[GetCourseTableParams](),
		),
		th.handleGetCourseTable,
	)

	s.AddTool(
		mcp.NewTool("editCourseCell",
			mcp.WithDescription("编辑个人课程表中的单个格子"),
			mcp.WithInputSchema[EditCourseCellParams](),
		),
		th.handleEditCourseCell,
	)

	s.AddTool(
		mcp.NewTool("queryFailRate",
			mcp.WithDescription("查询挂科率 - 如果不指定课程名则随机返回10条挂科率数据"),
			mcp.WithInputSchema[QueryFailRateParams](),
		),
		th.handleQueryFailRate,
	)

	s.AddTool(
		mcp.NewTool("countdown",
			mcp.WithDescription("倒数日管理 - 创建、获取、更新、删除倒数日"),
			mcp.WithInputSchema[CountdownParams](),
		),
		th.handleCountdown,
	)

	s.AddTool(
		mcp.NewTool("studyTask",
			mcp.WithDescription("学习清单管理 - 创建、获取、更新、删除、统计学习任务"),
			mcp.WithInputSchema[StudyTaskParams](),
		),
		th.handleStudyTask,
	)
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

// ============== Tool Handler Methods ==============

func (th *mcpToolHandlers) handleListHeroes(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	heroes, err := th.heroService.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取英雄榜失败: %w", err)
	}
	data, err := sonic.Marshal(heroes)
	if err != nil {
		return nil, fmt.Errorf("序列化英雄榜数据失败: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent(string(data))}}, nil
}

func (th *mcpToolHandlers) handleNotifications(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params NotificationsParams
	if err := req.BindArguments(&params); err != nil {
		return nil, fmt.Errorf("参数解析失败: %w", err)
	}

	switch params.Action {
	case "list":
		page, size := params.Page, params.Size
		if page <= 0 {
			page = 1
		}
		if size <= 0 {
			size = 20
		}
		result, err := th.notificationService.GetNotifications(ctx, &request.GetNotificationsRequest{Page: page, Size: size})
		if err != nil {
			return nil, fmt.Errorf("获取通知列表失败: %w", err)
		}
		data, _ := sonic.Marshal(result)
		return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent(string(data))}}, nil
	case "get":
		if params.ID == 0 {
			return nil, fmt.Errorf("请提供通知ID")
		}
		result, err := th.notificationService.GetNotificationByID(ctx, params.ID)
		if err != nil {
			return nil, fmt.Errorf("获取通知详情失败: %w", err)
		}
		data, _ := sonic.Marshal(result)
		return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent(string(data))}}, nil
	default:
		return nil, fmt.Errorf("不支持的操作: %s", params.Action)
	}
}

func (th *mcpToolHandlers) handleUserProfile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params UserProfileParams
	if err := req.BindArguments(&params); err != nil {
		return nil, fmt.Errorf("参数解析失败: %w", err)
	}

	userID := getUserFromContext(ctx)
	if userID == 0 {
		return nil, fmt.Errorf("用户未认证")
	}

	switch params.Action {
	case "get":
		user, err := th.authService.GetUserByID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("获取用户信息失败: %w", err)
		}
		result := map[string]any{"id": user.ID, "nickname": user.Nickname, "real_name": user.RealName, "college": user.College, "major": user.Major, "class_id": user.ClassID}
		data, _ := sonic.Marshal(result)
		return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent(string(data))}}, nil
	case "update":
		updates := make(map[string]any)
		if params.Nickname != "" {
			updates["nickname"] = params.Nickname
		}
		if params.RealName != "" {
			updates["real_name"] = params.RealName
		}
		if params.College != "" {
			updates["college"] = params.College
		}
		if params.Major != "" {
			updates["major"] = params.Major
		}
		if params.ClassID != "" {
			updates["class_id"] = params.ClassID
		}
		if err := th.authService.UpdateUserProfile(ctx, userID, updates); err != nil {
			return nil, fmt.Errorf("更新用户信息失败: %w", err)
		}
		return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent(`{"message": "更新成功"}`)}}, nil
	default:
		return nil, fmt.Errorf("不支持的操作: %s", params.Action)
	}
}

func (th *mcpToolHandlers) handleTeacherReview(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params TeacherReviewParams
	if err := req.BindArguments(&params); err != nil {
		return nil, fmt.Errorf("参数解析失败: %w", err)
	}

	userID := getUserFromContext(ctx)

	switch params.Action {
	case "create":
		if userID == 0 {
			return nil, fmt.Errorf("用户未认证")
		}
		if params.TeacherName == "" || params.Campus == "" || params.CourseName == "" || params.Content == "" || params.Attitude == 0 {
			return nil, fmt.Errorf("请提供完整的评价信息")
		}
		err := th.reviewService.CreateReview(ctx, userID, &request.CreateReviewRequest{
			TeacherName: params.TeacherName, Campus: params.Campus, CourseName: params.CourseName, Content: params.Content, Attitude: models.TeacherAttitude(params.Attitude),
		})
		if err != nil {
			return nil, fmt.Errorf("创建评价失败: %w", err)
		}
		return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent(`{"message": "评价提交成功，等待审核"}`)}}, nil
	case "get":
		if params.TeacherName == "" {
			return nil, fmt.Errorf("请提供教师姓名")
		}
		page, size := params.Page, params.Size
		if page <= 0 {
			page = 1
		}
		if size <= 0 {
			size = 10
		}
		reviews, total, err := th.reviewService.GetReviewsByTeacher(ctx, params.TeacherName, page, size)
		if err != nil {
			return nil, fmt.Errorf("获取评价失败: %w", err)
		}
		data, _ := sonic.Marshal(map[string]any{"data": reviews, "total": total, "page": page, "size": size})
		return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent(string(data))}}, nil
	default:
		return nil, fmt.Errorf("不支持的操作: %s", params.Action)
	}
}

func (th *mcpToolHandlers) handleGetCourseTable(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params GetCourseTableParams
	if err := req.BindArguments(&params); err != nil {
		return nil, fmt.Errorf("参数解析失败: %w", err)
	}

	userID := getUserFromContext(ctx)
	if userID == 0 {
		return nil, fmt.Errorf("用户未认证")
	}
	result, err := th.courseTableService.GetUserCourseTable(ctx, userID, params.Semester)
	if err != nil {
		return nil, fmt.Errorf("获取课程表失败: %w", err)
	}
	data, _ := sonic.Marshal(result)
	return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent(string(data))}}, nil
}

func (th *mcpToolHandlers) handleEditCourseCell(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params EditCourseCellParams
	if err := req.BindArguments(&params); err != nil {
		return nil, fmt.Errorf("参数解析失败: %w", err)
	}

	userID := getUserFromContext(ctx)
	if userID == 0 {
		return nil, fmt.Errorf("用户未认证")
	}
	valueBytes, err := sonic.Marshal(params.Value)
	if err != nil {
		return nil, fmt.Errorf("无效的格子数据: %w", err)
	}
	if err := th.courseTableService.EditUserCourseCell(ctx, userID, params.Semester, params.Index, datatypes.JSON(valueBytes)); err != nil {
		return nil, fmt.Errorf("编辑课程表失败: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent(`{"message": "编辑成功"}`)}}, nil
}

func (th *mcpToolHandlers) handleQueryFailRate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params QueryFailRateParams
	if err := req.BindArguments(&params); err != nil {
		return nil, fmt.Errorf("参数解析失败: %w", err)
	}

	if params.Keyword == "" {
		list, err := th.failRateService.Rand(ctx, 10)
		if err != nil {
			return nil, fmt.Errorf("查询挂科率失败: %w", err)
		}
		data, _ := sonic.Marshal(map[string]any{"data": list})
		return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent(string(data))}}, nil
	}
	page, size := params.Page, params.Size
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	list, total, err := th.failRateService.Search(ctx, params.Keyword, page, size)
	if err != nil {
		return nil, fmt.Errorf("查询挂科率失败: %w", err)
	}
	data, _ := sonic.Marshal(map[string]any{"data": list, "total": total, "page": page, "size": size})
	return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent(string(data))}}, nil
}

func (th *mcpToolHandlers) handleCountdown(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params CountdownParams
	if err := req.BindArguments(&params); err != nil {
		return nil, fmt.Errorf("参数解析失败: %w", err)
	}

	userID := getUserFromContext(ctx)
	if userID == 0 {
		return nil, fmt.Errorf("用户未认证")
	}

	switch params.Action {
	case "create":
		if params.Title == "" || params.TargetDate == "" {
			return nil, fmt.Errorf("请提供标题和目标日期")
		}
		result, err := th.countdownService.CreateCountdown(ctx, userID, &request.CreateCountdownRequest{Title: params.Title, Description: params.Description, TargetDate: params.TargetDate})
		if err != nil {
			return nil, fmt.Errorf("创建倒数日失败: %w", err)
		}
		data, _ := sonic.Marshal(result)
		return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent(string(data))}}, nil
	case "list":
		result, err := th.countdownService.GetCountdowns(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("获取倒数日列表失败: %w", err)
		}
		data, _ := sonic.Marshal(result)
		return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent(string(data))}}, nil
	case "get":
		if params.ID == 0 {
			return nil, fmt.Errorf("请提供倒数日ID")
		}
		result, err := th.countdownService.GetCountdownByID(ctx, params.ID, userID)
		if err != nil {
			return nil, fmt.Errorf("获取倒数日详情失败: %w", err)
		}
		data, _ := sonic.Marshal(result)
		return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent(string(data))}}, nil
	case "update":
		if params.ID == 0 {
			return nil, fmt.Errorf("请提供倒数日ID")
		}
		var titlePtr, descPtr, datePtr *string
		if params.Title != "" {
			titlePtr = &params.Title
		}
		if params.Description != "" {
			descPtr = &params.Description
		}
		if params.TargetDate != "" {
			datePtr = &params.TargetDate
		}
		result, err := th.countdownService.UpdateCountdown(ctx, params.ID, userID, &request.UpdateCountdownRequest{Title: titlePtr, Description: descPtr, TargetDate: datePtr})
		if err != nil {
			return nil, fmt.Errorf("更新倒数日失败: %w", err)
		}
		data, _ := sonic.Marshal(result)
		return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent(string(data))}}, nil
	case "delete":
		if params.ID == 0 {
			return nil, fmt.Errorf("请提供倒数日ID")
		}
		if err := th.countdownService.DeleteCountdown(ctx, params.ID, userID); err != nil {
			return nil, fmt.Errorf("删除倒数日失败: %w", err)
		}
		return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent(`{"message": "删除成功"}`)}}, nil
	default:
		return nil, fmt.Errorf("不支持的操作: %s", params.Action)
	}
}

func (th *mcpToolHandlers) handleStudyTask(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params StudyTaskParams
	if err := req.BindArguments(&params); err != nil {
		return nil, fmt.Errorf("参数解析失败: %w", err)
	}

	userID := getUserFromContext(ctx)
	if userID == 0 {
		return nil, fmt.Errorf("用户未认证")
	}

	switch params.Action {
	case "create":
		if params.Title == "" {
			return nil, fmt.Errorf("请提供任务标题")
		}
		priority := uint8(2)
		if params.Priority != nil {
			priority = *params.Priority
		}
		result, err := th.studyTaskService.CreateStudyTask(ctx, userID, &request.CreateStudyTaskRequest{Title: params.Title, Description: params.Description, DueDate: params.DueDate, Priority: priority})
		if err != nil {
			return nil, fmt.Errorf("创建学习任务失败: %w", err)
		}
		data, _ := sonic.Marshal(result)
		return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent(string(data))}}, nil
	case "list":
		page, size := params.Page, params.Size
		if page <= 0 {
			page = 1
		}
		if size <= 0 {
			size = 20
		}
		result, err := th.studyTaskService.GetStudyTasks(ctx, userID, &request.GetStudyTasksRequest{Page: page, Size: size, Status: params.Status, Priority: params.Priority})
		if err != nil {
			return nil, fmt.Errorf("获取学习任务列表失败: %w", err)
		}
		data, _ := sonic.Marshal(result)
		return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent(string(data))}}, nil
	case "get":
		if params.ID == 0 {
			return nil, fmt.Errorf("请提供任务ID")
		}
		result, err := th.studyTaskService.GetStudyTaskByID(ctx, params.ID, userID)
		if err != nil {
			return nil, fmt.Errorf("获取学习任务详情失败: %w", err)
		}
		data, _ := sonic.Marshal(result)
		return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent(string(data))}}, nil
	case "update":
		if params.ID == 0 {
			return nil, fmt.Errorf("请提供任务ID")
		}
		var titlePtr, descPtr, datePtr *string
		if params.Title != "" {
			titlePtr = &params.Title
		}
		if params.Description != "" {
			descPtr = &params.Description
		}
		if params.DueDate != "" {
			datePtr = &params.DueDate
		}
		result, err := th.studyTaskService.UpdateStudyTask(ctx, params.ID, userID, &request.UpdateStudyTaskRequest{Title: titlePtr, Description: descPtr, DueDate: datePtr, Priority: params.Priority, Status: params.Status})
		if err != nil {
			return nil, fmt.Errorf("更新学习任务失败: %w", err)
		}
		data, _ := sonic.Marshal(result)
		return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent(string(data))}}, nil
	case "delete":
		if params.ID == 0 {
			return nil, fmt.Errorf("请提供任务ID")
		}
		if err := th.studyTaskService.DeleteStudyTask(ctx, params.ID, userID); err != nil {
			return nil, fmt.Errorf("删除学习任务失败: %w", err)
		}
		return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent(`{"message": "删除成功"}`)}}, nil
	case "stats":
		result, err := th.studyTaskService.GetStudyTaskStats(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("获取学习任务统计失败: %w", err)
		}
		data, _ := sonic.Marshal(result)
		return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent(string(data))}}, nil
	default:
		return nil, fmt.Errorf("不支持的操作: %s", params.Action)
	}
}
