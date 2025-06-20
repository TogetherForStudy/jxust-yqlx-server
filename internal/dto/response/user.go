package response

import (
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
