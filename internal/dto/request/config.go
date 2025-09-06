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
