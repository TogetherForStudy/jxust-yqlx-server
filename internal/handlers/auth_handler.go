package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
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
	var req services.WechatLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	result, err := h.authService.WechatLogin(req.Code)
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
// @Param body body services.MockWechatLoginRequest true "模拟登录请求"
// @Success 200 {object} utils.Response{data=services.WechatLoginResponse}
// @Failure 400 {object} utils.Response
// @Router /api/auth/mock-wechat-login [post]
func (h *AuthHandler) MockWechatLogin(c *gin.Context) {
	var req services.MockWechatLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	result, err := h.authService.MockWechatLogin(req.TestUser)
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
	userID, exists := c.Get("user_id")
	if !exists {
		helper.ErrorResponse(c, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	user, err := h.authService.GetUserByID(userID.(uint))
	if err != nil {
		helper.ErrorResponse(c, http.StatusNotFound, "用户不存在")
		return
	}

	helper.SuccessResponse(c, user)
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
	userID, exists := c.Get("user_id")
	if !exists {
		helper.ErrorResponse(c, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	profile := &models.User{
		Nickname:  req.Nickname,
		Avatar:    req.Avatar,
		Phone:     req.Phone,
		StudentID: req.StudentID,
		RealName:  req.RealName,
		College:   req.College,
		Major:     req.Major,
		ClassID:   req.ClassID,
	}

	if err := h.authService.UpdateUserProfile(userID.(uint), profile); err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "更新失败")
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "更新成功"})
}
