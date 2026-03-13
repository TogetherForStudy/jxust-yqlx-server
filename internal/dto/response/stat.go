package response

// SystemOnlineStatResponse 系统在线统计响应
type SystemOnlineStatResponse struct {
	OnlineCount int64 `json:"online_count"` // 在线人数
}

// ProjectOnlineStatResponse 项目在线统计响应
type ProjectOnlineStatResponse struct {
	ProjectID   uint  `json:"project_id"`   // 项目ID
	OnlineCount int64 `json:"online_count"` // 在线人数
}

// AllProjectsOnlineStatItem 单个项目在线统计项
type AllProjectsOnlineStatItem struct {
	ProjectID   uint   `json:"project_id"`   // 项目ID
	ProjectName string `json:"project_name"` // 项目名称
	OnlineCount int64  `json:"online_count"` // 在线人数
}

// AllProjectsOnlineStatResponse 所有启用项目在线统计响应
type AllProjectsOnlineStatResponse struct {
	Projects []AllProjectsOnlineStatItem `json:"projects"` // 各项目在线统计
}

type AdminUserCountStatResponse struct {
	UserID uint  `json:"user_id"`
	Count  int64 `json:"count"`
}
