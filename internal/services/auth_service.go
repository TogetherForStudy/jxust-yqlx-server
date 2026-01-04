package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/config"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/utils"

	json "github.com/bytedance/sonic"
	"gorm.io/gorm"
)

type AuthService struct {
	db   *gorm.DB
	cfg  *config.Config
	rbac *RBACService // 仅用于 EnsureUserHasRoleByTag
}

func NewAuthService(db *gorm.DB, cfg *config.Config, rbac *RBACService) *AuthService {
	return &AuthService{
		db:   db,
		cfg:  cfg,
		rbac: rbac,
	}
}

// WechatLogin 微信小程序登录
func (s *AuthService) WechatLogin(ctx context.Context, code string) (*response.WechatLoginResponse, error) {
	// 调用微信API获取openid
	session, err := s.getWechatSession(ctx, code)
	if err != nil {
		return nil, err
	}

	if session.ErrCode != 0 {
		return nil, fmt.Errorf("微信登录失败: %s", session.ErrMsg)
	}

	// 查找或创建用户
	var user models.User
	err = s.db.WithContext(ctx).Where("open_id = ?", session.OpenID).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 创建新用户
			user = models.User{
				OpenID:    session.OpenID,
				UnionID:   session.UnionID,
				Status:    models.UserStatusNormal,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := s.db.WithContext(ctx).Create(&user).Error; err != nil {
				return nil, fmt.Errorf("创建用户失败: %w", err)
			}
		} else {
			return nil, fmt.Errorf("查询用户失败: %w", err)
		}
	}

	// 确保默认角色
	if s.rbac != nil {
		if err := s.rbac.EnsureUserHasRoleByTag(ctx, user.ID, models.RoleTagUserBasic); err != nil {
			return nil, fmt.Errorf("同步用户角色失败: %w", err)
		}
	}

	// 检查用户状态
	if user.Status == models.UserStatusDisabled {
		return nil, fmt.Errorf("用户账号已被禁用")
	}

	// 获取用户角色并映射到旧的role字段（向前兼容）
	role := s.mapRoleTagToLegacyRole(ctx, user.ID)

	// 更新User模型中的role字段
	if user.Role != role {
		if err := s.db.WithContext(ctx).Model(&user).Update("role", role).Error; err != nil {
			return nil, fmt.Errorf("更新用户角色失败: %w", err)
		}
		user.Role = role
	}

	// 生成JWT token（带角色信息）
	token, err := utils.GenerateJWT(user.ID, s.cfg.JWTSecret, role)
	if err != nil {
		return nil, fmt.Errorf("生成 Token 失败: %w", err)
	}

	// 获取角色标签（RBAC新逻辑）
	var roleTags []string
	if s.rbac != nil {
		if snap, err := s.rbac.GetUserPermissionSnapshot(ctx, user.ID); err == nil {
			roleTags = snap.RoleTags
		}
	}

	// 转换为 UserProfileResponse
	userProfile := response.UserProfileResponse{
		ID:        user.ID,
		Nickname:  user.Nickname,
		Avatar:    user.Avatar,
		Phone:     user.Phone,
		StudentID: user.StudentID,
		RealName:  user.RealName,
		College:   user.College,
		Major:     user.Major,
		ClassID:   user.ClassID,
		Role:      user.Role, // 向前兼容字段
		RoleTags:  roleTags,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	return &response.WechatLoginResponse{
		Token:    token,
		UserInfo: userProfile,
	}, nil
}

// getWechatSession 获取微信session信息
func (s *AuthService) getWechatSession(ctx context.Context, code string) (*response.WechatSession, error) {
	url := fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		s.cfg.WechatAppID, s.cfg.WechatAppSecret, code)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
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
func (s *AuthService) GetUserByID(ctx context.Context, userID uint) (*models.User, error) {
	var user models.User
	err := s.db.WithContext(ctx).First(&user, userID).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUserProfile 更新用户资料
func (s *AuthService) UpdateUserProfile(ctx context.Context, userID uint, profile *models.User) error {
	return s.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Updates(map[string]any{
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

// MockWechatLogin 模拟微信小程序登录 - 仅用于测试
func (s *AuthService) MockWechatLogin(ctx context.Context, testUser string) (*response.WechatLoginResponse, error) {
	// 根据测试用户类型生成不同的模拟数据
	var mockOpenID, mockUnionID, nickname, avatar string

	switch testUser {
	case "admin":
		mockOpenID = "mock_admin_openid_123456"
		mockUnionID = "mock_admin_unionid_123456"
		nickname = "测试管理员"
		avatar = "https://thirdwx.qlogo.cn/mmopen/vi_32/admin_avatar.png"
	case "basic":
		mockOpenID = "mock_basic_openid_789012"
		mockUnionID = "mock_basic_unionid_789012"
		nickname = "测试基本用户"
		avatar = "https://thirdwx.qlogo.cn/mmopen/vi_32/normal_avatar.png"
	case "active":
		mockOpenID = "mock_active_openid_345678"
		mockUnionID = "mock_active_unionid_345678"
		nickname = "测试活跃用户"
		avatar = "https://thirdwx.qlogo.cn/mmopen/vi_32/active_avatar.png"
	case "verified":
		mockOpenID = "mock_verified_openid_123456"
		mockUnionID = "mock_verified_unionid_123456"
		nickname = "测试认证用户"
		avatar = "https://thirdwx.qlogo.cn/mmopen/vi_32/verified_avatar.png"
	case "operator":
		mockOpenID = "mock_operator_openid_123456"
		mockUnionID = "mock_operator_unionid_123456"
		nickname = "测试运营"
		avatar = "https://thirdwx.qlogo.cn/mmopen/vi_32/operator_avatar.png"
	default:
		return nil, fmt.Errorf("不支持的测试用户类型: %s", testUser)
	}

	// 查找或创建用户
	var user models.User
	err := s.db.WithContext(ctx).Where("open_id = ?", mockOpenID).First(&user).Error
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
				Status:    models.UserStatusNormal,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := s.db.WithContext(ctx).Create(&user).Error; err != nil {
				return nil, fmt.Errorf("创建测试用户失败: %w", err)
			}
		} else {
			return nil, fmt.Errorf("查询用户失败: %w", err)
		}
	}

	// 同步角色绑定，所有用户都分配 UserBasic，admin 不需多角色，active/verified/operator 再额外分配具体角色
	if s.rbac != nil {
		// 必须绑定 UserBasic 角色
		if err := s.rbac.EnsureUserHasRoleByTag(ctx, user.ID, models.RoleTagUserBasic); err != nil {
			return nil, fmt.Errorf("同步测试用户基础角色失败: %w", err)
		}

		switch testUser {
		case "active":
			if err := s.rbac.EnsureUserHasRoleByTag(ctx, user.ID, models.RoleTagUserActive); err != nil {
				return nil, fmt.Errorf("同步测试用户 active 角色失败: %w", err)
			}
		case "verified":
			if err := s.rbac.EnsureUserHasRoleByTag(ctx, user.ID, models.RoleTagUserVerified); err != nil {
				return nil, fmt.Errorf("同步测试用户 verified 角色失败: %w", err)
			}
		case "operator":
			if err := s.rbac.EnsureUserHasRoleByTag(ctx, user.ID, models.RoleTagOperator); err != nil {
				return nil, fmt.Errorf("同步测试用户 operator 角色失败: %w", err)
			}
		}
	}

	// 检查用户状态
	if user.Status == models.UserStatusDisabled {
		return nil, fmt.Errorf("用户账号已被禁用")
	}

	// 获取用户角色并映射到旧的role字段（向前兼容）
	role := s.mapRoleTagToLegacyRole(ctx, user.ID)

	// 更新User模型中的role字段
	if user.Role != role {
		if err := s.db.WithContext(ctx).Model(&user).Update("role", role).Error; err != nil {
			return nil, fmt.Errorf("更新用户角色失败: %w", err)
		}
		user.Role = role
	}

	// 生成JWT token（带角色信息）
	token, err := utils.GenerateJWT(user.ID, s.cfg.JWTSecret, role)
	if err != nil {
		return nil, fmt.Errorf("生成token失败: %w", err)
	}

	// 获取角色标签（RBAC新逻辑）
	var roleTags []string
	if s.rbac != nil {
		if snap, err := s.rbac.GetUserPermissionSnapshot(ctx, user.ID); err == nil {
			roleTags = snap.RoleTags
		}
	}

	// 转换为 UserProfileResponse
	userProfile := response.UserProfileResponse{
		ID:        user.ID,
		Nickname:  user.Nickname,
		Avatar:    user.Avatar,
		Phone:     user.Phone,
		StudentID: user.StudentID,
		RealName:  user.RealName,
		College:   user.College,
		Major:     user.Major,
		ClassID:   user.ClassID,
		Role:      user.Role, // 向前兼容字段
		RoleTags:  roleTags,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	return &response.WechatLoginResponse{
		Token:    token,
		UserInfo: userProfile,
	}, nil
}

// mapRoleTagToLegacyRole 将RBAC角色标签映射到旧的role字段（向前兼容）
// 返回：1=普通用户，2=管理员，3=运营
func (s *AuthService) mapRoleTagToLegacyRole(ctx context.Context, userID uint) int8 {
	if s.rbac == nil {
		return 1 // 默认普通用户
	}

	snap, err := s.rbac.GetUserPermissionSnapshot(ctx, userID)
	if err != nil {
		return 1 // 默认普通用户
	}

	// 检查角色优先级：admin > operator > user
	hasAdmin := false
	hasOperator := false

	for _, tag := range snap.RoleTags {
		if tag == models.RoleTagAdmin {
			hasAdmin = true
			break // admin优先级最高，找到就返回
		}
		if tag == models.RoleTagOperator {
			hasOperator = true
		}
	}

	if hasAdmin {
		return 2 // 管理员
	}
	if hasOperator {
		return 3 // 运营
	}

	return 1 // 普通用户
}
