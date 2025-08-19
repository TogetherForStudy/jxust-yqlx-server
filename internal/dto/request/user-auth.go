package request

// WechatLoginRequest 微信登录请求
type WechatLoginRequest struct {
	Code string `json:"code" binding:"required"`
}

// UpdateProfileRequest 更新用户资料请求
type UpdateProfileRequest struct {
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar"`
	Phone     string `json:"phone"`
	StudentID string `json:"student_id"`
	RealName  string `json:"real_name"`
	College   string `json:"college"`
	Major     string `json:"major"`
	ClassID   string `json:"class_id"`
}

// MockWechatLoginRequest 模拟微信登录请求
type MockWechatLoginRequest struct {
	TestUser string `json:"test_user" binding:"required"` // 测试用户类型: normal, admin, new_user
}
