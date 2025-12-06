package handlers

import (
	"net/http"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
	"github.com/gin-gonic/gin"
)

type ConfigHandler struct {
	service *services.ConfigService
}

func NewConfigHandler(service *services.ConfigService) *ConfigHandler {
	return &ConfigHandler{service: service}
}

// GetByKey 按key返回配置项
func (h *ConfigHandler) GetByKey(c *gin.Context) {
	key := c.Param("key")
	m, err := h.service.GetByKey(c, key)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	resp := response.ConfigResponse{
		Key:       m.Key,
		Value:     m.Value,
		ValueType: m.ValueType,
	}
	helper.SuccessResponse(c, resp)
}

// Create 管理员创建配置项
func (h *ConfigHandler) Create(c *gin.Context) {
	var req request.CreateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}
	m, err := h.service.Create(c, req.Key, req.Value, req.ValueType, req.Description)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	helper.SuccessResponse(c, m)
}

// Update 管理员按key更新配置项
func (h *ConfigHandler) Update(c *gin.Context) {
	key := c.Param("key")
	var req request.UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}
	if err := h.service.Update(c, key, req.Value, req.ValueType, req.Description); err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	helper.SuccessResponse(c, "更新成功")
}

// Delete 管理员按key删除（软删除）
func (h *ConfigHandler) Delete(c *gin.Context) {
	key := c.Param("key")
	if err := h.service.Delete(c, key); err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	helper.SuccessResponse(c, "删除成功")
}

// SearchConfigs 搜索配置项，空query返回全部（支持分页）
func (h *ConfigHandler) SearchConfigs(c *gin.Context) {
	var req request.SearchConfigRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 10
	}

	items, total, err := h.service.SearchConfigs(c, req.Query, req.Page, req.Size)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "搜索失败")
		return
	}
	helper.PageSuccessResponse(c, items, total, req.Page, req.Size)
}
