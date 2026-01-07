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
