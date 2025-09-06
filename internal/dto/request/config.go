package request

// CreateConfigRequest 创建配置项请求
type CreateConfigRequest struct {
	Key         string `json:"key" binding:"required"`
	Value       string `json:"value" binding:"required"`
	ValueType   string `json:"value_type" binding:"oneof=string number boolean json"`
	Description string `json:"description"`
}

// UpdateConfigRequest 更新配置项请求
type UpdateConfigRequest struct {
	Value       string `json:"value" binding:"required"`
	ValueType   string `json:"value_type" binding:"oneof=string number boolean json"`
	Description string `json:"description"`
}

// SearchConfigRequest 搜索配置项请求
type SearchConfigRequest struct {
	Query string `form:"q" json:"q"`                  // 搜索关键词
	Page  int    `form:"page,default=1" json:"page"`  // 页码
	Size  int    `form:"size,default=10" json:"size"` // 每页数量
}
