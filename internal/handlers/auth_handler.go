package handlers

import (
	"errors"
	"io"
	"strconv"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/apperr"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *services.AuthService
	rbacService *services.RBACService
}

func NewAuthHandler(authService *services.AuthService, rbacService *services.RBACService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		rbacService: rbacService,
	}
}

// WechatLogin 微信小程序登录
func (h *AuthHandler) WechatLogin(c *gin.Context) {
	var req request.WechatLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.HandleError(c, apperr.Wrap(constant.CommonBadRequest, err))
		return
	}

	result, err := h.authService.WechatLogin(c.Request.Context(), req.Code, c.Request.UserAgent())
	if err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessResponse(c, result)
}

// MockWechatLogin 模拟微信小程序登录 - 仅用于测试
func (h *AuthHandler) MockWechatLogin(c *gin.Context) {
	var req request.MockWechatLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.HandleError(c, apperr.Wrap(constant.CommonBadRequest, err))
		return
	}

	result, err := h.authService.MockWechatLogin(c.Request.Context(), req.TestUser, c.Request.UserAgent())
	if err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessResponse(c, result)
}

// AdminLogin 管理界面手机号密码登录
func (h *AuthHandler) AdminLogin(c *gin.Context) {
	var req request.AdminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.HandleError(c, apperr.Wrap(constant.CommonBadRequest, err))
		return
	}

	result, err := h.authService.AdminLogin(c.Request.Context(), req.Phone, req.Password, c.Request.UserAgent())
	if err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessResponse(c, result)
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req request.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.HandleError(c, apperr.Wrap(constant.CommonBadRequest, err))
		return
	}

	result, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken, c.Request.UserAgent())
	if err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessResponse(c, result)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	userID := helper.GetUserID(c)
	if userID == 0 {
		helper.HandleErrCode(c, constant.AuthMissingUserContext)
		return
	}

	sid := helper.GetAuthSessionID(c)
	if err := h.authService.Logout(c.Request.Context(), userID, sid); err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "退出成功"})
}

func (h *AuthHandler) LogoutAll(c *gin.Context) {
	userID := helper.GetUserID(c)
	if userID == 0 {
		helper.HandleErrCode(c, constant.AuthMissingUserContext)
		return
	}

	deleted, err := h.authService.LogoutAll(c.Request.Context(), userID)
	if err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "已退出全部设备", "deleted_session_count": deleted})
}

// GetProfile 获取用户资料
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := helper.GetUserID(c)
	if userID == 0 {
		helper.HandleErrCode(c, constant.AuthMissingUserContext)
		return
	}

	user, err := h.authService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		helper.HandleError(c, err)
		return
	}

	var roleTags []string
	if h.rbacService != nil {
		if snap, err := h.rbacService.GetUserPermissionSnapshot(c.Request.Context(), user.ID); err != nil {
			logger.WarnGin(c, map[string]any{
				"action":         "get_user_roles",
				"message":        "获取用户角色失败",
				"target_user_id": user.ID,
				"error":          err.Error(),
			})
		} else {
			roleTags = snap.RoleTags
		}
	}

	resp := response.UserProfileResponse{
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
	}

	helper.SuccessResponse(c, resp)
}

// UpdateProfile 更新用户资料
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID := helper.GetUserID(c)
	if userID == 0 {
		helper.HandleErrCode(c, constant.AuthMissingUserContext)
		return
	}

	var req request.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.HandleError(c, apperr.Wrap(constant.CommonBadRequest, err))
		return
	}

	updates := make(map[string]any)
	if req.Nickname != nil {
		updates["nickname"] = *req.Nickname
	}
	if req.Avatar != nil {
		updates["avatar"] = *req.Avatar
	}
	if req.Phone != nil {
		updates["phone"] = *req.Phone
	}
	if req.StudentID != nil {
		updates["student_id"] = *req.StudentID
	}
	if req.RealName != nil {
		updates["real_name"] = *req.RealName
	}
	if req.College != nil {
		updates["college"] = *req.College
	}
	if req.Major != nil {
		updates["major"] = *req.Major
	}
	if req.ClassID != nil {
		updates["class_id"] = *req.ClassID
	}

	if err := h.authService.UpdateUserProfile(c.Request.Context(), userID, updates); err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "更新成功"})
}

func (h *AuthHandler) KickUser(c *gin.Context) {
	targetUserID, err := parsePathUserID(c)
	if err != nil {
		helper.HandleError(c, apperr.Wrap(constant.CommonBadRequest, err))
		return
	}

	operatorUserID := helper.GetUserID(c)
	deleted, err := h.authService.KickUser(c.Request.Context(), operatorUserID, targetUserID)
	if err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "用户已踢下线", "deleted_session_count": deleted})
}

func (h *AuthHandler) BanUser(c *gin.Context) {
	targetUserID, err := parsePathUserID(c)
	if err != nil {
		helper.HandleError(c, apperr.Wrap(constant.CommonBadRequest, err))
		return
	}

	var req request.BanUserRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		helper.HandleError(c, apperr.Wrap(constant.CommonBadRequest, err))
		return
	}

	operatorUserID := helper.GetUserID(c)
	deleted, err := h.authService.BanUser(c.Request.Context(), operatorUserID, targetUserID, req.DurationSeconds, req.Reason)
	if err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "封禁成功", "deleted_session_count": deleted})
}

func (h *AuthHandler) UnbanUser(c *gin.Context) {
	targetUserID, err := parsePathUserID(c)
	if err != nil {
		helper.HandleError(c, apperr.Wrap(constant.CommonBadRequest, err))
		return
	}

	operatorUserID := helper.GetUserID(c)
	if err := h.authService.UnbanUser(c.Request.Context(), operatorUserID, targetUserID); err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "解封成功"})
}

func (h *AuthHandler) GetUserDetail(c *gin.Context) {
	targetUserID, err := parsePathUserID(c)
	if err != nil {
		helper.HandleError(c, apperr.Wrap(constant.CommonBadRequest, err))
		return
	}

	result, err := h.authService.GetUserAuthDetail(c.Request.Context(), targetUserID)
	if err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessResponse(c, result)
}

func (h *AuthHandler) SetAdminLoginCredentials(c *gin.Context) {
	targetUserID, err := parsePathUserID(c)
	if err != nil {
		helper.HandleError(c, apperr.Wrap(constant.CommonBadRequest, err))
		return
	}

	operatorUserID := helper.GetUserID(c)
	if operatorUserID == 0 {
		helper.HandleErrCode(c, constant.AuthMissingUserContext)
		return
	}

	var req request.AdminLoginCredentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.HandleError(c, apperr.Wrap(constant.CommonBadRequest, err))
		return
	}

	if err := h.authService.SetAdminLoginCredentials(c.Request.Context(), operatorUserID, targetUserID, req.Phone, req.Password); err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "后台登录凭据更新成功"})
}

func parsePathUserID(c *gin.Context) (uint, error) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return 0, errors.New("无效的用户ID")
	}
	return uint(userID), nil
}
