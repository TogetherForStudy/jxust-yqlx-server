package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/config"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/utils"

	"gorm.io/gorm"
)

type AuthService struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewAuthService(db *gorm.DB, cfg *config.Config) *AuthService {
	return &AuthService{
		db:  db,
		cfg: cfg,
	}
}

// WechatLogin 微信小程序登录
func (s *AuthService) WechatLogin(code string) (*response.WechatLoginResponse, error) {
	// 调用微信API获取openid
	session, err := s.getWechatSession(code)
	if err != nil {
		return nil, err
	}

	if session.ErrCode != 0 {
		return nil, fmt.Errorf("微信登录失败: %s", session.ErrMsg)
	}

	// 查找或创建用户
	var user models.User
	err = s.db.Where("open_id = ?", session.OpenID).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 创建新用户
			user = models.User{
				OpenID:    session.OpenID,
				UnionID:   session.UnionID,
				Role:      models.UserRoleNormal,
				Status:    models.UserStatusNormal,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := s.db.Create(&user).Error; err != nil {
				return nil, fmt.Errorf("创建用户失败: %w", err)
			}
		} else {
			return nil, fmt.Errorf("查询用户失败: %w", err)
		}
	}

	// 检查用户状态
	if user.Status == models.UserStatusDisabled {
		return nil, fmt.Errorf("用户账号已被禁用")
	}

	// 生成JWT token
	token, err := utils.GenerateJWT(user.ID, user.OpenID, uint8(user.Role), s.cfg.JWTSecret)
	if err != nil {
		return nil, fmt.Errorf("生成token失败: %w", err)
	}

	return &response.WechatLoginResponse{
		Token:    token,
		UserInfo: user,
	}, nil
}

// getWechatSession 获取微信session信息
func (s *AuthService) getWechatSession(code string) (*response.WechatSession, error) {
	url := fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		s.cfg.WechatAppID, s.cfg.WechatAppSecret, code)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("请求微信API失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var session response.WechatSession
	if err := json.Unmarshal(body, &session); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &session, nil
}

// GetUserByID 根据ID获取用户信息
func (s *AuthService) GetUserByID(userID uint) (*models.User, error) {
	var user models.User
	err := s.db.First(&user, userID).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUserProfile 更新用户资料
func (s *AuthService) UpdateUserProfile(userID uint, profile *models.User) error {
	return s.db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]any{
		"nickname":   profile.Nickname,
		"avatar":     profile.Avatar,
		"phone":      profile.Phone,
		"student_id": profile.StudentID,
		"real_name":  profile.RealName,
		"college":    profile.College,
		"major":      profile.Major,
		"class_id":   profile.ClassID,
		"updated_at": time.Now(),
	}).Error
}

// MockWechatLoginRequest 模拟微信登录请求
type MockWechatLoginRequest struct {
	TestUser string `json:"test_user" binding:"required"` // 测试用户类型: normal, admin, new_user
}

// MockWechatLogin 模拟微信小程序登录 - 仅用于测试
func (s *AuthService) MockWechatLogin(testUser string) (*response.WechatLoginResponse, error) {
	// 根据测试用户类型生成不同的模拟数据
	var mockOpenID, mockUnionID, nickname, avatar string
	var role models.UserRole

	switch testUser {
	case "admin":
		mockOpenID = "mock_admin_openid_123456"
		mockUnionID = "mock_admin_unionid_123456"
		nickname = "测试管理员"
		avatar = "https://thirdwx.qlogo.cn/mmopen/vi_32/admin_avatar.png"
		role = models.UserRoleAdmin
	case "normal":
		mockOpenID = "mock_normal_openid_789012"
		mockUnionID = "mock_normal_unionid_789012"
		nickname = "测试用户"
		avatar = "https://thirdwx.qlogo.cn/mmopen/vi_32/normal_avatar.png"
		role = models.UserRoleNormal
	case "new_user":
		mockOpenID = "mock_new_openid_345678"
		mockUnionID = "mock_new_unionid_345678"
		nickname = "新用户"
		avatar = "https://thirdwx.qlogo.cn/mmopen/vi_32/new_avatar.png"
		role = models.UserRoleNormal
	default:
		return nil, fmt.Errorf("不支持的测试用户类型: %s", testUser)
	}

	// 查找或创建用户
	var user models.User
	err := s.db.Where("open_id = ?", mockOpenID).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 创建新用户
			user = models.User{
				OpenID:    mockOpenID,
				UnionID:   mockUnionID,
				Nickname:  nickname,
				Avatar:    avatar,
				StudentID: fmt.Sprintf("2023%06d", time.Now().UnixNano()%1000000),
				RealName:  nickname,
				College:   "计算机学院",
				Major:     "软件工程",
				ClassID:   "2023级1班",
				Role:      role,
				Status:    models.UserStatusNormal,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := s.db.Create(&user).Error; err != nil {
				return nil, fmt.Errorf("创建测试用户失败: %w", err)
			}
		} else {
			return nil, fmt.Errorf("查询用户失败: %w", err)
		}
	}

	// 检查用户状态
	if user.Status == models.UserStatusDisabled {
		return nil, fmt.Errorf("用户账号已被禁用")
	}

	// 生成JWT token
	token, err := utils.GenerateJWT(user.ID, user.OpenID, uint8(user.Role), s.cfg.JWTSecret)
	if err != nil {
		return nil, fmt.Errorf("生成token失败: %w", err)
	}

	return &response.WechatLoginResponse{
		Token:    token,
		UserInfo: user,
	}, nil
}
