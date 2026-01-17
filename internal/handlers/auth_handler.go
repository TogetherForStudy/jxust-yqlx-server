package handlers

import (
	"net/http"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
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
// @Summary 微信小程序登录
// @Description 通过微信授权码登录获取JWT token
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body services.WechatLoginRequest true "登录请求"
// @Success 200 {object} utils.Response{data=services.WechatLoginResponse}
// @Failure 400 {object} utils.Response
// @Router /api/auth/wechat-login [post]
func (h *AuthHandler) WechatLogin(c *gin.Context) {
	var req request.WechatLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	result, err := h.authService.WechatLogin(c, req.Code)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// MockWechatLogin 模拟微信小程序登录 - 仅用于测试
// @Summary 模拟微信小程序登录
// @Description 模拟微信登录返回信息，用于测试其他接口。支持的测试用户类型：normal(普通用户), admin(管理员), new_user(新用户)
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body request.MockWechatLoginRequest true "模拟登录请求"
// @Success 200 {object} utils.Response{data=services.WechatLoginResponse}
// @Failure 400 {object} utils.Response
// @Router /api/auth/mock-wechat-login [post]
func (h *AuthHandler) MockWechatLogin(c *gin.Context) {
	var req request.MockWechatLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	result, err := h.authService.MockWechatLogin(c, req.TestUser)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}

// GetProfile 获取用户资料
// @Summary 获取当前用户资料
// @Description 获取当前登录用户的详细信息
// @Tags 用户
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} utils.Response{data=models.User}
// @Failure 401 {object} utils.Response
// @Router /api/user/profile [get]
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, exists := helper.GetUserID(c)
	if !exists {
		helper.ErrorResponse(c, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	user, err := h.authService.GetUserByID(c, userID.(uint))
	if err != nil {
		helper.ErrorResponse(c, http.StatusNotFound, "用户不存在")
		return
	}

	// 获取角色标签（RBAC新逻辑）
	var roleTags []string
	if h.rbacService != nil {
		if snap, err := h.rbacService.GetUserPermissionSnapshot(c, user.ID); err != nil {
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
		Role:      user.Role, // 向前兼容字段
		RoleTags:  roleTags,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	helper.SuccessResponse(c, resp)
}

// UpdateProfile 更新用户资料
// @Summary 更新用户资料
// @Description 更新当前登录用户的资料信息
// @Tags 用户
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param body body UpdateProfileRequest true "用户资料"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Router /api/user/profile [put]
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID, exists := helper.GetUserID(c)
	if !exists {
		helper.ErrorResponse(c, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	var req request.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	// 构建更新 map，只包含前端传递的字段（指针不为 nil 的字段）
	// 这样可以区分"未传递"和"空值"：
	// - 指针为 nil：字段未传递，不更新
	// - 指针不为 nil：字段已传递，更新（即使值为空字符串）
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

	if err := h.authService.UpdateUserProfile(c, userID.(uint), updates); err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "更新失败")
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "更新成功"})
}
