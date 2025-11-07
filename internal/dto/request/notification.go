package request

// CreateNotificationRequest 创建通知请求
type CreateNotificationRequest struct {
	Title      string `json:"title" binding:"required,max=200"`         // 通知标题
	Content    string `json:"content" binding:"required"`               // 详细内容
	Categories []int  `json:"categories" binding:"required,min=1,dive"` // 分类ID数组
}

// UpdateNotificationRequest 更新通知请求
type UpdateNotificationRequest struct {
	Title      *string `json:"title" binding:"omitempty,max=200"` // 通知标题
	Content    *string `json:"content"`                           // 详细内容
	Categories []int   `json:"categories" binding:"dive"`         // 分类ID数组
}

// GetNotificationsRequest 获取通知列表请求
type GetNotificationsRequest struct {
	Page       int    `form:"page" binding:"min=1"`         // 页码
	Size       int    `form:"size" binding:"min=1,max=100"` // 每页数量
	Categories []int  `form:"categories"`                   // 分类过滤
	Status     *uint8 `form:"status" binding:"omitempty"`   // 状态过滤：1=草稿，2=待审核，3=已发布，4=已删除
	Keyword    string `form:"keyword"`                      // 关键词搜索
}

// ConvertToScheduleRequest 转换为日程请求
type ConvertToScheduleRequest struct {
	Title       string                      `json:"title" binding:"required"`           // 总日程名称
	Description string                      `json:"description"`                        // 日程描述
	TimeSlots   []ConvertToScheduleTimeSlot `json:"time_slots" binding:"required,dive"` // 时间段列表
}

// ConvertToScheduleTimeSlot 日程时间段请求
type ConvertToScheduleTimeSlot struct {
	Name      string `json:"name" binding:"required"`       // 时间段名称
	StartDate string `json:"start_date" binding:"required"` // 开始日期 YYYY-MM-DD
	EndDate   string `json:"end_date"`                      // 结束日期 YYYY-MM-DD
	StartTime string `json:"start_time"`                    // 开始时间 HH:MM
	EndTime   string `json:"end_time"`                      // 结束时间 HH:MM
	IsAllDay  bool   `json:"is_all_day"`                    // 是否全天
}

// ApproveNotificationRequest 审核通知请求
type ApproveNotificationRequest struct {
	Status NotificationApprovalStatusRequest `json:"status" binding:"required,oneof=1 2"` // 审核状态：1=同意，2=拒绝
	Note   string                            `json:"note" binding:"max=500"`              // 审核备注
}

type NotificationApprovalStatusRequest int8

const (
	NotificationApprovalStatusRequestApproved NotificationApprovalStatusRequest = 1 // 同意
	NotificationApprovalStatusRequestRejected NotificationApprovalStatusRequest = 2 // 拒绝
)

// CreateCategoryRequest 创建分类请求
type CreateCategoryRequest struct {
	Name     string `json:"name" binding:"required,max=20"` // 分类名称
	Sort     int    `json:"sort"`                           // 排序值
	IsActive bool   `json:"is_active"`                      // 是否启用
}

// UpdateCategoryRequest 更新分类请求
type UpdateCategoryRequest struct {
	Name     *string `json:"name" binding:"omitempty,max=20"` // 分类名称
	Sort     *int    `json:"sort"`                            // 排序值
	IsActive *bool   `json:"is_active"`                       // 是否启用
}
