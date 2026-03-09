package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/config"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/apperr"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/cache"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/utils"

	json "github.com/bytedance/sonic"
	rediscache "github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var backofficeRoleTags = []string{constant.RoleTagAdmin, constant.RoleTagOperator}

type AuthService struct {
	db    *gorm.DB
	cfg   *config.Config
	rbac  *RBACService
	cache cache.Cache
}

type authSessionRecord struct {
	SID     string
	Session utils.AuthSession
}

func NewAuthService(db *gorm.DB, cfg *config.Config, rbac *RBACService, ca cache.Cache) *AuthService {
	return &AuthService{
		db:    db,
		cfg:   cfg,
		rbac:  rbac,
		cache: ca,
	}
}

// AdminLogin 后台手机号密码登录
func (s *AuthService) AdminLogin(ctx context.Context, phone, password, userAgent string) (*response.WechatLoginResponse, error) {
	phone = normalizePhone(phone)

	user, err := s.findAdminUserByPhone(ctx, phone)
	if err != nil {
		if appErr, ok := apperr.As(err); ok && appErr.Code == constant.AuthAdminLoginFailed {
			logger.WarnCtx(ctx, map[string]any{
				"action":  "auth_admin_login_failed",
				"message": "admin login rejected",
				"phone":   phone,
			})
			return nil, err
		}
		return nil, err
	}

	if user.Password == "" || !utils.CheckPassword(password, user.Password) {
		logger.WarnCtx(ctx, map[string]any{
			"action":  "auth_admin_login_failed",
			"message": "admin login password mismatch",
			"phone":   phone,
			"user_id": user.ID,
		})
		return nil, apperr.New(constant.AuthAdminLoginFailed)
	}

	return s.completeLoginWithAction(ctx, user, userAgent, "auth_admin_login_success")
}

// WechatLogin 微信小程序登录
func (s *AuthService) WechatLogin(ctx context.Context, code, userAgent string) (*response.WechatLoginResponse, error) {
	session, err := s.getWechatSession(ctx, code)
	if err != nil {
		logger.WarnCtx(ctx, map[string]any{
			"action":  "auth_login_failed",
			"message": "wechat session request failed",
			"error":   err.Error(),
		})
		return nil, err
	}

	if session.ErrCode != 0 {
		err := apperr.Wrap(constant.AuthWechatLoginFailed, fmt.Errorf("wechat errcode=%d errmsg=%s", session.ErrCode, session.ErrMsg))
		logger.WarnCtx(ctx, map[string]any{
			"action":  "auth_login_failed",
			"message": "wechat session returned error",
			"error":   err.Error(),
		})
		return nil, err
	}

	var user models.User
	err = s.db.WithContext(ctx).Where("open_id = ?", session.OpenID).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			user = models.User{
				OpenID:    session.OpenID,
				UnionID:   session.UnionID,
				Status:    models.UserStatusNormal,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := s.db.WithContext(ctx).Create(&user).Error; err != nil {
				logger.ErrorCtx(ctx, map[string]any{
					"action":  "auth_login_failed",
					"message": "failed to create user during wechat login",
					"error":   err.Error(),
				})
				return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("创建用户失败: %w", err))
			}
		} else {
			logger.ErrorCtx(ctx, map[string]any{
				"action":  "auth_login_failed",
				"message": "failed to query user during wechat login",
				"error":   err.Error(),
			})
			return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询用户失败: %w", err))
		}
	}

	return s.completeLogin(ctx, &user, userAgent)
}

func (s *AuthService) completeLogin(ctx context.Context, user *models.User, userAgent string) (*response.WechatLoginResponse, error) {
	return s.completeLoginWithAction(ctx, user, userAgent, "auth_login_success")
}

func (s *AuthService) completeLoginWithAction(ctx context.Context, user *models.User, userAgent, action string) (*response.WechatLoginResponse, error) {
	if err := s.ensureDefaultRole(ctx, user.ID); err != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action":  action,
			"message": "failed to sync default role",
			"user_id": user.ID,
			"error":   err.Error(),
		})
		return nil, err
	}
	if err := s.ensureUserLoginAllowed(ctx, *user); err != nil {
		logger.WarnCtx(ctx, map[string]any{
			"action":  action,
			"message": "login rejected by user state",
			"user_id": user.ID,
			"error":   err.Error(),
		})
		return nil, err
	}
	return s.issueTokenPair(ctx, user, userAgent, action)
}

func (s *AuthService) ensureDefaultRole(ctx context.Context, userID uint) error {
	if s.rbac == nil {
		return nil
	}
	if err := s.rbac.EnsureUserHasRoleByTag(ctx, userID, constant.RoleTagUserBasic); err != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("同步用户角色失败: %w", err))
	}
	return nil
}

func (s *AuthService) ensureUserLoginAllowed(ctx context.Context, user models.User) error {
	if user.Status == models.UserStatusDisabled {
		return apperr.New(constant.AuthAccountDisabled)
	}

	blockInfo, err := s.getBlockInfo(ctx, user.ID)
	if err != nil {
		return apperr.Wrap(constant.CommonInternal, err)
	}
	if blockInfo == nil {
		return nil
	}
	return blockedError(blockInfo)
}

func blockedError(blockInfo *utils.AuthBlockInfo) error {
	switch blockInfo.Type {
	case constant.AuthBlockTypeKick:
		return apperr.New(constant.AuthAccountKicked)
	case constant.AuthBlockTypeTempBan:
		return apperr.New(constant.AuthAccountTempBanned)
	default:
		return apperr.New(constant.AuthAccountDisabled)
	}
}

func (s *AuthService) issueTokenPair(ctx context.Context, user *models.User, userAgent, action string) (*response.WechatLoginResponse, error) {
	if err := s.requireAuthCache(); err != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action":  action,
			"message": "auth cache is unavailable",
			"user_id": user.ID,
			"error":   err.Error(),
		})
		return nil, err
	}

	role := s.mapRoleTagToLegacyRole(ctx, user.ID)
	if user.Role != role {
		if err := s.db.WithContext(ctx).Model(user).Update("role", role).Error; err != nil {
			return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("更新用户角色失败: %w", err))
		}
		user.Role = role
	}

	sid := utils.NewSessionID()
	accessToken, accessClaims, err := utils.GenerateAccessToken(user.ID, s.cfg.JWTSecret, role, s.accessTokenTTL(), sid)
	if err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("生成 AccessToken 失败: %w", err))
	}
	refreshToken, refreshClaims, err := utils.GenerateRefreshToken(user.ID, s.refreshTokenSecret(), s.refreshTokenTTL(), sid)
	if err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("生成 RefreshToken 失败: %w", err))
	}

	deviceInfo := utils.ParseDeviceInfo(userAgent)
	session := utils.AuthSession{
		UserID:        user.ID,
		RefreshJTI:    refreshClaims.JTI,
		DeviceType:    deviceInfo.DeviceType,
		ClientType:    deviceInfo.ClientType,
		IssuedAt:      accessClaims.IssuedAt.Unix(),
		LastRefreshAt: accessClaims.IssuedAt.Unix(),
		ExpiresAt:     refreshClaims.ExpiresAt.Unix(),
	}
	if err := s.storeSession(ctx, sid, session); err != nil {
		return nil, err
	}

	userProfile, err := s.buildUserProfile(ctx, *user)
	if err != nil {
		return nil, err
	}

	logger.InfoCtx(ctx, map[string]any{
		"action":      action,
		"message":     "issued auth token pair",
		"user_id":     user.ID,
		"sid":         sid,
		"jti":         accessClaims.JTI,
		"refresh_jti": refreshClaims.JTI,
		"device_type": deviceInfo.DeviceType,
		"client_type": deviceInfo.ClientType,
	})

	return &response.WechatLoginResponse{
		Token:                accessToken,
		RefreshToken:         refreshToken,
		AccessTokenExpiresAt: accessClaims.ExpiresAt.Unix(),
		UserInfo:             userProfile,
	}, nil
}

func (s *AuthService) storeSession(ctx context.Context, sid string, session utils.AuthSession) error {
	sessionPayload, err := json.Marshal(session)
	if err != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("序列化会话失败: %w", err))
	}

	ttl := s.refreshTokenTTL()
	if err := s.cache.Set(ctx, fmt.Sprintf(constant.AuthSessionKeyFormat, sid), string(sessionPayload), &ttl); err != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("保存会话失败: %w", err))
	}
	if err := s.cache.ZAdd(ctx, s.userSessionsIndexKey(session.UserID), float64(session.ExpiresAt), sid); err != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("保存用户会话索引失败: %w", err))
	}
	if err := s.cache.Expire(ctx, s.userSessionsIndexKey(session.UserID), s.refreshTokenTTL()); err != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("更新用户会话索引过期时间失败: %w", err))
	}
	if err := s.cleanupUserSessionIndex(ctx, session.UserID); err != nil {
		logger.WarnCtx(ctx, map[string]any{
			"action":  "auth_session_index_cleanup_failed",
			"message": "failed to prune stale session references after storing session",
			"user_id": session.UserID,
			"sid":     sid,
			"error":   err.Error(),
		})
	}
	return nil
}

func (s *AuthService) getSession(ctx context.Context, sid string) (*utils.AuthSession, error) {
	payload, err := s.cache.Get(ctx, fmt.Sprintf(constant.AuthSessionKeyFormat, sid))
	if err != nil {
		if isCacheMiss(err) {
			return nil, nil
		}
		return nil, apperr.Wrap(constant.CommonInternal, err)
	}

	var session utils.AuthSession
	if err := json.Unmarshal([]byte(payload), &session); err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("解析会话失败: %w", err))
	}
	return &session, nil
}

func (s *AuthService) loadUserSessions(ctx context.Context, userID uint) ([]authSessionRecord, error) {
	if s.cache == nil {
		return nil, nil
	}

	userSessionsKey := s.userSessionsIndexKey(userID)
	now := time.Now().UTC().Unix()
	if _, err := s.cache.ZRemRangeByScore(ctx, userSessionsKey, math.Inf(-1), float64(now)); err != nil && !isCacheMiss(err) {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("清理过期用户会话索引失败: %w", err))
	}

	sids, err := s.cache.ZRangeByScore(ctx, userSessionsKey, float64(now+1), math.Inf(1))
	if err != nil && !isCacheMiss(err) {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("读取用户会话列表失败: %w", err))
	}

	records := make([]authSessionRecord, 0, len(sids))
	staleSIDs := make([]interface{}, 0)
	for _, sid := range sids {
		session, err := s.getSession(ctx, sid)
		if err != nil {
			return nil, err
		}
		if session == nil {
			staleSIDs = append(staleSIDs, sid)
			continue
		}
		records = append(records, authSessionRecord{
			SID:     sid,
			Session: *session,
		})
	}

	if len(staleSIDs) > 0 {
		if _, err := s.cache.ZRem(ctx, userSessionsKey, staleSIDs...); err != nil && !isCacheMiss(err) {
			return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("清理失效用户会话索引失败: %w", err))
		}
	}
	return records, nil
}

func (s *AuthService) cleanupUserSessionIndex(ctx context.Context, userID uint) error {
	_, err := s.loadUserSessions(ctx, userID)
	return err
}

func (s *AuthService) userSessionsIndexKey(userID uint) string {
	return fmt.Sprintf(constant.AuthUserSessionsKeyFormat, userID)
}

func (s *AuthService) getBlockInfo(ctx context.Context, userID uint) (*utils.AuthBlockInfo, error) {
	if s.cache == nil {
		return nil, nil
	}

	payload, err := s.cache.Get(ctx, fmt.Sprintf(constant.AuthBlockedKeyFormat, userID))
	if err != nil {
		if isCacheMiss(err) {
			return nil, nil
		}
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("读取封禁状态失败: %w", err))
	}

	var blockInfo utils.AuthBlockInfo
	if err := json.Unmarshal([]byte(payload), &blockInfo); err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("解析封禁状态失败: %w", err))
	}
	return &blockInfo, nil
}

func (s *AuthService) setBlockInfo(ctx context.Context, userID uint, blockInfo utils.AuthBlockInfo, ttl time.Duration) error {
	payload, err := json.Marshal(blockInfo)
	if err != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("序列化封禁状态失败: %w", err))
	}
	if err := s.cache.Set(ctx, fmt.Sprintf(constant.AuthBlockedKeyFormat, userID), string(payload), &ttl); err != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("写入封禁状态失败: %w", err))
	}
	return nil
}

func (s *AuthService) clearBlockInfo(ctx context.Context, userID uint) error {
	if s.cache == nil {
		return nil
	}
	if err := s.cache.Delete(ctx, fmt.Sprintf(constant.AuthBlockedKeyFormat, userID)); err != nil {
		if isCacheMiss(err) {
			return nil
		}
		return apperr.Wrap(constant.CommonInternal, err)
	}
	return nil
}

func (s *AuthService) revokeCurrentSession(ctx context.Context, sid string) error {
	ttl := s.accessTokenTTL()
	if err := s.cache.Set(ctx, fmt.Sprintf(constant.AuthRevokedSessionKeyFormat, sid), "1", &ttl); err != nil {
		return apperr.Wrap(constant.CommonInternal, err)
	}
	return nil
}

func (s *AuthService) revokeAllAccessTokens(ctx context.Context, userID uint, at time.Time) error {
	ttl := s.accessTokenTTL()
	if err := s.cache.Set(ctx, fmt.Sprintf(constant.AuthRevokedBeforeKeyFormat, userID), strconv.FormatInt(at.Unix(), 10), &ttl); err != nil {
		return apperr.Wrap(constant.CommonInternal, err)
	}
	return nil
}

func (s *AuthService) deleteSession(ctx context.Context, userID uint, sid string) error {
	if err := s.cache.Delete(ctx, fmt.Sprintf(constant.AuthSessionKeyFormat, sid)); err != nil && !isCacheMiss(err) {
		return apperr.Wrap(constant.CommonInternal, err)
	}
	if _, err := s.cache.ZRem(ctx, s.userSessionsIndexKey(userID), sid); err != nil && !isCacheMiss(err) {
		return apperr.Wrap(constant.CommonInternal, err)
	}
	return nil
}

func (s *AuthService) revokeAllSessions(ctx context.Context, userID uint) (int, error) {
	if err := s.requireAuthCache(); err != nil {
		return 0, err
	}

	records, err := s.loadUserSessions(ctx, userID)
	if err != nil {
		return 0, err
	}

	deleted := 0
	for _, record := range records {
		if err := s.cache.Delete(ctx, fmt.Sprintf(constant.AuthSessionKeyFormat, record.SID)); err != nil && !isCacheMiss(err) {
			return deleted, apperr.Wrap(constant.CommonInternal, fmt.Errorf("删除用户会话失败: %w", err))
		}
		deleted++
	}
	if err := s.cache.Delete(ctx, s.userSessionsIndexKey(userID)); err != nil && !isCacheMiss(err) {
		return deleted, apperr.Wrap(constant.CommonInternal, fmt.Errorf("删除用户会话索引失败: %w", err))
	}

	now := time.Now().UTC()
	if err := s.revokeAllAccessTokens(ctx, userID, now); err != nil {
		return deleted, apperr.Wrap(constant.CommonInternal, fmt.Errorf("写入 access token 撤销标记失败: %w", err))
	}
	return deleted, nil
}

func (s *AuthService) accessTokenTTL() time.Duration {
	if s.cfg.AccessTokenTTL > 0 {
		return s.cfg.AccessTokenTTL
	}
	return constant.DefaultAccessTokenTTL
}

func (s *AuthService) refreshTokenTTL() time.Duration {
	if s.cfg.RefreshTokenTTL > 0 {
		return s.cfg.RefreshTokenTTL
	}
	return constant.DefaultRefreshTokenTTL
}

func (s *AuthService) refreshTokenSecret() string {
	if s.cfg.RefreshTokenSecret != "" {
		return s.cfg.RefreshTokenSecret
	}
	return s.cfg.JWTSecret
}

func (s *AuthService) requireAuthCache() error {
	if s.cache == nil {
		return apperr.New(constant.AuthCacheUnavailable)
	}
	return nil
}

func isCacheMiss(err error) bool {
	return errors.Is(err, rediscache.Nil)
}

// RefreshToken 刷新 AccessToken 和 RefreshToken。
func (s *AuthService) RefreshToken(ctx context.Context, refreshTokenString, userAgent string) (*response.WechatLoginResponse, error) {
	if err := s.requireAuthCache(); err != nil {
		return nil, err
	}

	claims, err := utils.ParseToken(refreshTokenString, s.refreshTokenSecret())
	if err != nil {
		logger.WarnCtx(ctx, map[string]any{
			"action":  "auth_refresh_failed",
			"message": "invalid refresh token",
			"error":   err.Error(),
		})
		return nil, apperr.New(constant.AuthRefreshTokenInvalid)
	}
	if claims.TokenType != constant.AuthTokenTypeRefresh {
		return nil, apperr.New(constant.AuthRefreshTokenTypeInvalid)
	}

	session, err := s.getSession(ctx, claims.SID)
	if err != nil {
		return nil, err
	}
	if session == nil || session.UserID != claims.UserID {
		return nil, apperr.New(constant.AuthRefreshTokenSessionNotFound)
	}
	if session.RefreshJTI != claims.JTI {
		logger.WarnCtx(ctx, map[string]any{
			"action":  "auth_refresh_failed",
			"message": "refresh token jti mismatch",
			"user_id": claims.UserID,
			"sid":     claims.SID,
			"jti":     claims.JTI,
		})
		return nil, apperr.New(constant.AuthRefreshTokenExpired)
	}

	user, err := s.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureUserLoginAllowed(ctx, *user); err != nil {
		return nil, err
	}

	currentDevice := utils.ParseDeviceInfo(userAgent)
	if session.DeviceType != currentDevice.DeviceType || session.ClientType != currentDevice.ClientType {
		logger.WarnCtx(ctx, map[string]any{
			"action":              "auth_device_mismatch",
			"message":             "device summary changed during refresh",
			"user_id":             claims.UserID,
			"sid":                 claims.SID,
			"stored_device_type":  session.DeviceType,
			"stored_client_type":  session.ClientType,
			"current_device_type": currentDevice.DeviceType,
			"current_client_type": currentDevice.ClientType,
		})
	}

	role := s.mapRoleTagToLegacyRole(ctx, user.ID)
	if user.Role != role {
		if err := s.db.WithContext(ctx).Model(user).Update("role", role).Error; err != nil {
			return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("更新用户角色失败: %w", err))
		}
		user.Role = role
	}

	accessToken, accessClaims, err := utils.GenerateAccessToken(user.ID, s.cfg.JWTSecret, role, s.accessTokenTTL(), claims.SID)
	if err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("生成 AccessToken 失败: %w", err))
	}
	newRefreshToken, refreshClaims, err := utils.GenerateRefreshToken(user.ID, s.refreshTokenSecret(), s.refreshTokenTTL(), claims.SID)
	if err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("生成 RefreshToken 失败: %w", err))
	}

	session.RefreshJTI = refreshClaims.JTI
	session.LastRefreshAt = time.Now().UTC().Unix()
	session.ExpiresAt = refreshClaims.ExpiresAt.Unix()
	if err := s.storeSession(ctx, claims.SID, *session); err != nil {
		return nil, err
	}

	userProfile, err := s.buildUserProfile(ctx, *user)
	if err != nil {
		return nil, err
	}

	logger.InfoCtx(ctx, map[string]any{
		"action":      "auth_refresh_success",
		"message":     "rotated refresh token",
		"user_id":     user.ID,
		"sid":         claims.SID,
		"jti":         accessClaims.JTI,
		"refresh_jti": refreshClaims.JTI,
		"device_type": session.DeviceType,
		"client_type": session.ClientType,
	})

	return &response.WechatLoginResponse{
		Token:                accessToken,
		RefreshToken:         newRefreshToken,
		AccessTokenExpiresAt: accessClaims.ExpiresAt.Unix(),
		UserInfo:             userProfile,
	}, nil
}

// getWechatSession 获取微信session信息
func (s *AuthService) getWechatSession(ctx context.Context, code string) (*response.WechatSession, error) {
	url := fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		s.cfg.WechatAppID, s.cfg.WechatAppSecret, code)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("创建请求失败: %w", err))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("请求微信API失败: %w", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("读取响应失败: %w", err))
	}

	var session response.WechatSession
	if err := json.Unmarshal(body, &session); err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("解析响应失败: %w", err))
	}

	return &session, nil
}

// GetUserByID 根据ID获取用户信息
func (s *AuthService) GetUserByID(ctx context.Context, userID uint) (*models.User, error) {
	var user models.User
	err := s.db.WithContext(ctx).First(&user, userID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.New(constant.CommonUserNotFound)
		}
		return nil, apperr.Wrap(constant.CommonInternal, err)
	}
	return &user, nil
}

func (s *AuthService) buildUserProfile(ctx context.Context, user models.User) (response.UserProfileResponse, error) {
	var roleTags []string
	if s.rbac != nil {
		if snap, err := s.rbac.GetUserPermissionSnapshot(ctx, user.ID); err == nil {
			roleTags = snap.RoleTags
		}
	}

	return response.UserProfileResponse{
		ID:        user.ID,
		Nickname:  user.Nickname,
		Avatar:    user.Avatar,
		Phone:     user.Phone,
		StudentID: user.StudentID,
		RealName:  user.RealName,
		College:   user.College,
		Major:     user.Major,
		ClassID:   user.ClassID,
		Role:      user.Role,
		RoleTags:  roleTags,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

// UpdateUserProfile 更新用户资料
func (s *AuthService) UpdateUserProfile(ctx context.Context, userID uint, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}

	if phoneRaw, ok := updates["phone"]; ok {
		phone, ok := phoneRaw.(string)
		if !ok {
			return apperr.New(constant.CommonBadRequest)
		}
		normalizedPhone := normalizePhone(phone)
		updates["phone"] = normalizedPhone

		isBackoffice, err := s.isBackofficeUser(ctx, userID, 0)
		if err != nil {
			return err
		}
		if isBackoffice {
			if normalizedPhone == "" {
				return apperr.New(constant.AuthAdminPhoneRequired)
			}
			if err := s.ensureAdminPhoneAvailable(ctx, normalizedPhone, userID); err != nil {
				return err
			}
		}
	}

	updates["updated_at"] = time.Now()
	if err := s.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return apperr.Wrap(constant.CommonInternal, err)
	}
	return nil
}

// SetAdminLoginCredentials 设置后台登录凭据
func (s *AuthService) SetAdminLoginCredentials(ctx context.Context, operatorUserID, targetUserID uint, phone, password string) error {
	phone = normalizePhone(phone)
	if phone == "" {
		return apperr.New(constant.AuthAdminPhoneRequired)
	}
	if err := validateAdminPassword(password); err != nil {
		return err
	}

	targetUser, err := s.GetUserByID(ctx, targetUserID)
	if err != nil {
		return err
	}

	isBackoffice, err := s.isBackofficeUser(ctx, targetUserID, targetUser.Role)
	if err != nil {
		return err
	}
	if !isBackoffice {
		return apperr.New(constant.AuthAdminTargetRoleInvalid)
	}

	if err := s.ensureAdminPhoneAvailable(ctx, phone, targetUserID); err != nil {
		return err
	}

	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("生成后台密码哈希失败: %w", err))
	}

	updates := map[string]any{
		"phone":      phone,
		"password":   passwordHash,
		"updated_at": time.Now(),
	}
	if err := s.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", targetUserID).Updates(updates).Error; err != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("更新后台登录凭据失败: %w", err))
	}

	logger.InfoCtx(ctx, map[string]any{
		"action":           "auth_admin_credentials_updated",
		"message":          "updated admin login credentials",
		"operator_user_id": operatorUserID,
		"target_user_id":   targetUserID,
	})

	return nil
}

// MockWechatLogin 模拟微信小程序登录 - 仅用于测试
func (s *AuthService) MockWechatLogin(ctx context.Context, testUser, userAgent string) (*response.WechatLoginResponse, error) {
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
		return nil, apperr.New(constant.AuthUnsupportedTestUserType)
	}

	var user models.User
	err := s.db.WithContext(ctx).Where("open_id = ?", mockOpenID).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
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
				return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("创建测试用户失败: %w", err))
			}
		} else {
			return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询用户失败: %w", err))
		}
	}

	if s.rbac != nil {
		if err := s.rbac.EnsureUserHasRoleByTag(ctx, user.ID, constant.RoleTagUserBasic); err != nil {
			return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("同步测试用户基础角色失败: %w", err))
		}

		switch testUser {
		case "active":
			if err := s.rbac.EnsureUserHasRoleByTag(ctx, user.ID, constant.RoleTagUserActive); err != nil {
				return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("同步测试用户 active 角色失败: %w", err))
			}
		case "verified":
			if err := s.rbac.EnsureUserHasRoleByTag(ctx, user.ID, constant.RoleTagUserVerified); err != nil {
				return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("同步测试用户 verified 角色失败: %w", err))
			}
		case "operator":
			if err := s.rbac.EnsureUserHasRoleByTag(ctx, user.ID, constant.RoleTagOperator); err != nil {
				return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("同步测试用户 operator 角色失败: %w", err))
			}
		case "admin":
			if err := s.rbac.EnsureUserHasRoleByTag(ctx, user.ID, constant.RoleTagAdmin); err != nil {
				return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("同步测试用户 admin 角色失败: %w", err))
			}
		}
	}

	return s.completeLogin(ctx, &user, userAgent)
}

func (s *AuthService) Logout(ctx context.Context, userID uint, sid string) error {
	if err := s.requireAuthCache(); err != nil {
		return err
	}

	if sid == "" {
		return apperr.New(constant.AuthMissingSessionInfo)
	}

	if err := s.deleteSession(ctx, userID, sid); err != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("删除会话失败: %w", err))
	}
	if err := s.revokeCurrentSession(ctx, sid); err != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("写入会话撤销标记失败: %w", err))
	}

	logger.InfoCtx(ctx, map[string]any{
		"action":  "auth_logout",
		"message": "logout current session",
		"user_id": userID,
		"sid":     sid,
	})
	return nil
}

func (s *AuthService) LogoutAll(ctx context.Context, userID uint) (int, error) {
	deleted, err := s.revokeAllSessions(ctx, userID)
	if err != nil {
		return deleted, err
	}

	logger.InfoCtx(ctx, map[string]any{
		"action":                "auth_logout_all",
		"message":               "logout all sessions",
		"user_id":               userID,
		"deleted_session_count": deleted,
	})
	return deleted, nil
}

func (s *AuthService) KickUser(ctx context.Context, operatorUserID, targetUserID uint) (int, error) {
	if _, err := s.GetUserByID(ctx, targetUserID); err != nil {
		return 0, err
	}

	deleted, err := s.revokeAllSessions(ctx, targetUserID)
	if err != nil {
		return deleted, err
	}

	expiresAt := time.Now().UTC().Add(s.accessTokenTTL())
	blockInfo := utils.AuthBlockInfo{
		Type:           constant.AuthBlockTypeKick,
		OperatorUserID: operatorUserID,
		ExpiresAt:      expiresAt.Unix(),
	}
	if err := s.setBlockInfo(ctx, targetUserID, blockInfo, s.accessTokenTTL()); err != nil {
		return deleted, err
	}

	logger.InfoCtx(ctx, map[string]any{
		"action":                "auth_admin_kick",
		"message":               "kicked user offline",
		"operator_user_id":      operatorUserID,
		"target_user_id":        targetUserID,
		"deleted_session_count": deleted,
		"block_type":            blockInfo.Type,
		"expires_at":            blockInfo.ExpiresAt,
	})
	return deleted, nil
}

func (s *AuthService) BanUser(ctx context.Context, operatorUserID, targetUserID uint, durationSeconds int64, reason string) (int, error) {
	user, err := s.GetUserByID(ctx, targetUserID)
	if err != nil {
		return 0, err
	}

	deleted, err := s.revokeAllSessions(ctx, targetUserID)
	if err != nil {
		return deleted, err
	}

	blockType := constant.AuthBlockTypePermanent
	blockTTL := s.accessTokenTTL()
	if durationSeconds > 0 {
		blockType = constant.AuthBlockTypeTempBan
		blockTTL = time.Duration(durationSeconds) * time.Second
	} else if user.Status != models.UserStatusDisabled {
		if err := s.db.WithContext(ctx).Model(user).Update("status", models.UserStatusDisabled).Error; err != nil {
			return deleted, apperr.Wrap(constant.CommonInternal, fmt.Errorf("更新用户状态失败: %w", err))
		}
		user.Status = models.UserStatusDisabled
	}

	expiresAt := time.Now().UTC().Add(blockTTL)
	blockInfo := utils.AuthBlockInfo{
		Type:           blockType,
		Reason:         reason,
		OperatorUserID: operatorUserID,
		ExpiresAt:      expiresAt.Unix(),
	}
	if err := s.setBlockInfo(ctx, targetUserID, blockInfo, blockTTL); err != nil {
		return deleted, err
	}

	logger.InfoCtx(ctx, map[string]any{
		"action":                "auth_admin_ban",
		"message":               "user ban applied",
		"operator_user_id":      operatorUserID,
		"target_user_id":        targetUserID,
		"deleted_session_count": deleted,
		"block_type":            blockInfo.Type,
		"duration_seconds":      durationSeconds,
		"expires_at":            blockInfo.ExpiresAt,
		"reason":                reason,
	})
	return deleted, nil
}

func (s *AuthService) UnbanUser(ctx context.Context, operatorUserID, targetUserID uint) error {
	user, err := s.GetUserByID(ctx, targetUserID)
	if err != nil {
		return err
	}

	if err := s.clearBlockInfo(ctx, targetUserID); err != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("清除封禁状态失败: %w", err))
	}
	if user.Status == models.UserStatusDisabled {
		if err := s.db.WithContext(ctx).Model(user).Update("status", models.UserStatusNormal).Error; err != nil {
			return apperr.Wrap(constant.CommonInternal, fmt.Errorf("恢复用户状态失败: %w", err))
		}
	}

	logger.InfoCtx(ctx, map[string]any{
		"action":           "auth_admin_unban",
		"message":          "user unbanned",
		"operator_user_id": operatorUserID,
		"target_user_id":   targetUserID,
	})
	return nil
}

func (s *AuthService) GetUserAuthDetail(ctx context.Context, userID uint) (*response.UserAuthDetailResponse, error) {
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	profile, err := s.buildUserProfile(ctx, *user)
	if err != nil {
		return nil, err
	}

	result := &response.UserAuthDetailResponse{
		UserInfo: profile,
		Devices:  make([]response.AuthSessionSummary, 0),
	}

	blockInfo, err := s.getBlockInfo(ctx, userID)
	if err != nil {
		return nil, err
	}
	if blockInfo != nil {
		result.BlockType = blockInfo.Type
		result.BlockReason = blockInfo.Reason
		result.BlockExpiresAt = blockInfo.ExpiresAt
	}

	if s.cache == nil {
		return result, nil
	}

	records, err := s.loadUserSessions(ctx, userID)
	if err != nil {
		return nil, err
	}

	for _, record := range records {
		result.Devices = append(result.Devices, response.AuthSessionSummary{
			SID:           record.SID,
			DeviceType:    record.Session.DeviceType,
			ClientType:    record.Session.ClientType,
			IssuedAt:      record.Session.IssuedAt,
			LastRefreshAt: record.Session.LastRefreshAt,
			ExpiresAt:     record.Session.ExpiresAt,
		})
	}

	sort.Slice(result.Devices, func(i, j int) bool {
		return result.Devices[i].LastRefreshAt > result.Devices[j].LastRefreshAt
	})
	result.SessionCount = len(result.Devices)
	return result, nil
}

// mapRoleTagToLegacyRole 将RBAC角色标签映射到旧的role字段（向前兼容）
// 返回：1=普通用户，2=管理员，3=运营
func (s *AuthService) mapRoleTagToLegacyRole(ctx context.Context, userID uint) int8 {
	if s.rbac == nil {
		return 1
	}

	snap, err := s.rbac.GetUserPermissionSnapshot(ctx, userID)
	if err != nil {
		return 1
	}

	hasAdmin := false
	hasOperator := false
	for _, tag := range snap.RoleTags {
		if tag == constant.RoleTagAdmin {
			hasAdmin = true
			break
		}
		if tag == constant.RoleTagOperator {
			hasOperator = true
		}
	}

	if hasAdmin {
		return 2
	}
	if hasOperator {
		return 3
	}
	return 1
}

func (s *AuthService) findAdminUserByPhone(ctx context.Context, phone string) (*models.User, error) {
	if phone == "" {
		return nil, apperr.New(constant.AuthAdminLoginFailed)
	}

	var userIDs []uint
	if err := s.db.WithContext(ctx).
		Table("users").
		Distinct("users.id").
		Joins("JOIN user_roles ur ON ur.user_id = users.id").
		Joins("JOIN roles ON roles.id = ur.role_id").
		Where("users.phone = ?", phone).
		Where("roles.role_tag IN ?", backofficeRoleTags).
		Pluck("users.id", &userIDs).Error; err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("按手机号查询后台用户失败: %w", err))
	}

	if len(userIDs) != 1 {
		return nil, apperr.New(constant.AuthAdminLoginFailed)
	}

	user, err := s.GetUserByID(ctx, userIDs[0])
	if err != nil {
		if appErr, ok := apperr.As(err); ok && appErr.Code == constant.CommonUserNotFound {
			return nil, apperr.New(constant.AuthAdminLoginFailed)
		}
		return nil, err
	}
	return user, nil
}

func (s *AuthService) ensureAdminPhoneAvailable(ctx context.Context, phone string, excludeUserID uint) error {
	if phone == "" {
		return apperr.New(constant.AuthAdminPhoneRequired)
	}

	var userIDs []uint
	query := s.db.WithContext(ctx).
		Table("users").
		Distinct("users.id").
		Joins("JOIN user_roles ur ON ur.user_id = users.id").
		Joins("JOIN roles ON roles.id = ur.role_id").
		Where("users.phone = ?", phone).
		Where("roles.role_tag IN ?", backofficeRoleTags)
	if excludeUserID != 0 {
		query = query.Where("users.id <> ?", excludeUserID)
	}
	if err := query.Pluck("users.id", &userIDs).Error; err != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("校验后台手机号冲突失败: %w", err))
	}
	if len(userIDs) > 0 {
		return apperr.New(constant.AuthAdminPhoneConflict)
	}
	return nil
}

func (s *AuthService) isBackofficeUser(ctx context.Context, userID uint, legacyRole int8) (bool, error) {
	if s.rbac == nil {
		return legacyRole == 2 || legacyRole == 3, nil
	}

	snap, err := s.rbac.GetUserPermissionSnapshot(ctx, userID)
	if err != nil {
		return false, apperr.Wrap(constant.CommonInternal, fmt.Errorf("获取后台角色快照失败: %w", err))
	}

	for _, roleTag := range snap.RoleTags {
		if roleTag == constant.RoleTagAdmin || roleTag == constant.RoleTagOperator {
			return true, nil
		}
	}

	return false, nil
}

func normalizePhone(phone string) string {
	return strings.TrimSpace(phone)
}

func validateAdminPassword(password string) error {
	if len(password) < 8 {
		return apperr.New(constant.AuthAdminPasswordInvalid)
	}

	hasLetter := false
	hasDigit := false
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}

	if !hasLetter || !hasDigit {
		return apperr.New(constant.AuthAdminPasswordInvalid)
	}
	return nil
}
