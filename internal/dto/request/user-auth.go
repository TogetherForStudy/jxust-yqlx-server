package request

// WechatLoginRequest 微信登录请求
type WechatLoginRequest struct {
	Code string `json:"code" binding:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type BanUserRequest struct {
	DurationSeconds int64  `json:"duration_seconds" binding:"omitempty,min=0"`
	Reason          string `json:"reason"`
}

// UpdateProfileRequest 更新用户资料请求
type UpdateProfileRequest struct {
	Nickname  *string `json:"nickname"`
	Avatar    *string `json:"avatar"`
	Phone     *string `json:"phone"`
	StudentID *string `json:"student_id"`
	RealName  *string `json:"real_name"`
	College   *string `json:"college"`
	Major     *string `json:"major"`
	ClassID   *string `json:"class_id"`
}

// MockWechatLoginRequest 模拟微信登录请求
type MockWechatLoginRequest struct {
	TestUser string `json:"test_user" binding:"required"` // 测试用户类型: basic, active, verified, operator, admin
}
