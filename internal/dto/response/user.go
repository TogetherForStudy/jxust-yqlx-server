package response

import (
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
)

// WechatLoginResponse 微信登录响应
type WechatLoginResponse struct {
	Token    string      `json:"token"`
	UserInfo models.User `json:"user_info"`
}

// WechatSession 微信session信息
type WechatSession struct {
	OpenID     string `json:"openid"`
	UnionID    string `json:"unionid"`
	SessionKey string `json:"session_key"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

// UserProfileResponse 用户资料响应（不包含 openid/unionid）
type UserProfileResponse struct {
	ID        uint              `json:"id"`
	Nickname  string            `json:"nickname"`
	Avatar    string            `json:"avatar"`
	Phone     string            `json:"phone"`
	StudentID string            `json:"student_id"`
	RealName  string            `json:"real_name"`
	College   string            `json:"college"`
	Major     string            `json:"major"`
	ClassID   string            `json:"class_id"`
	Role      models.UserRole   `json:"role"`
	Status    models.UserStatus `json:"status"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}
