package handlers

import (
	"encoding/json"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/utils"
	"github.com/gin-gonic/gin"
)

type GPABackupHandler struct {
	service *services.GPABackupService
}

func NewGPABackupHandler(service *services.GPABackupService) *GPABackupHandler {
	return &GPABackupHandler{service: service}
}

func (h *GPABackupHandler) CreateBackup(c *gin.Context) {
	userID := helper.GetUserID(c)
	if userID == 0 {
		helper.HandleErrCode(c, constant.AuthMissingUserContext)
		return
	}

	var req request.CreateGPABackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	now := utils.FormatDateTime(utils.GetLocalTime())
	req.Data["create"] = now
	req.Data["update"] = now

	normalizedBody, err := json.Marshal(req.Data)
	if err != nil {
		helper.HandleError(c, err)
		return
	}

	result, err := h.service.CreateBackup(c.Request.Context(), userID, req.Title, normalizedBody)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessResponse(c, result)
}

func (h *GPABackupHandler) ListBackups(c *gin.Context) {
	userID := helper.GetUserID(c)
	if userID == 0 {
		helper.HandleErrCode(c, constant.AuthMissingUserContext)
		return
	}

	result, err := h.service.ListBackups(c.Request.Context(), userID)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessResponse(c, result)
}

func (h *GPABackupHandler) GetBackupByID(c *gin.Context) {
	userID := helper.GetUserID(c)
	if userID == 0 {
		helper.HandleErrCode(c, constant.AuthMissingUserContext)
		return
	}

	var req struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	result, err := h.service.GetBackupByID(c.Request.Context(), userID, req.ID)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessResponse(c, result)
}

func (h *GPABackupHandler) DeleteBackup(c *gin.Context) {
	userID := helper.GetUserID(c)
	if userID == 0 {
		helper.HandleErrCode(c, constant.AuthMissingUserContext)
		return
	}

	var req struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&req); err != nil {
		helper.HandleErrCode(c, constant.CommonBadRequest)
		return
	}

	if err := h.service.DeleteBackup(c.Request.Context(), userID, req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessResponse(c, gin.H{"message": "删除成功"})
}
