package request

// CreateGPABackupRequest 创建绩点备份请求
type CreateGPABackupRequest struct {
	Title string         `json:"title" binding:"required,max=200"` // 备份标题
	Data  map[string]any `json:"data" binding:"required"`          // 绩点备份原始数据
}
