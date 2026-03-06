package response

import (
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
)

// WechatLoginResponse 微信登录响应
type WechatLoginResponse struct {
	Token                string              `json:"token"` // access token
	RefreshToken         string              `json:"refresh_token"`
	AccessTokenExpiresAt int64               `json:"access_token_expires_at"`
	UserInfo             UserProfileResponse `json:"user_info"`
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
	Role      int8              `json:"role,omitempty"` // 向前兼容字段：1=普通用户，2=管理员，3=运营
	RoleTags  []string          `json:"role_tags,omitempty"`
	Status    models.UserStatus `json:"status"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type AuthSessionSummary struct {
	SID           string `json:"sid"`
	DeviceType    string `json:"device_type"`
	ClientType    string `json:"client_type"`
	IssuedAt      int64  `json:"issued_at"`
	LastRefreshAt int64  `json:"last_refresh_at"`
	ExpiresAt     int64  `json:"expires_at"`
}

type UserAuthDetailResponse struct {
	UserInfo       UserProfileResponse  `json:"user_info"`
	BlockType      string               `json:"block_type,omitempty"`
	BlockReason    string               `json:"block_reason,omitempty"`
	BlockExpiresAt int64                `json:"block_expires_at,omitempty"`
	SessionCount   int                  `json:"session_count"`
	Devices        []AuthSessionSummary `json:"devices"`
}
