package response

// ConfigResponse 配置项响应
type ConfigResponse struct {
	ID          uint   `json:"id"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	ValueType   string `json:"value_type"`
	Description string `json:"description"`
}
