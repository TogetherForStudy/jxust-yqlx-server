package handlers

import (
	"net/http"
	"strconv"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
	"github.com/gin-gonic/gin"
)

type HeroHandler struct {
	service *services.HeroService
}

func NewHeroHandler(service *services.HeroService) *HeroHandler {
	return &HeroHandler{service: service}
}

// Create 创建 Hero
func (h *HeroHandler) Create(c *gin.Context) {
	var req request.CreateHeroRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	m, err := h.service.Create(req.Name, req.Sort, req.IsShow)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	helper.SuccessResponse(c, m)
}

// Update 更新 Hero
func (h *HeroHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id64 == 0 {
		helper.ValidateResponse(c, "无效的ID")
		return
	}
	var req request.UpdateHeroRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}
	if err := h.service.Update(uint(id64), req.Name, req.Sort, req.IsShow); err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	helper.SuccessResponse(c, "更新成功")
}

// Delete 物理删除
func (h *HeroHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id64 == 0 {
		helper.ValidateResponse(c, "无效的ID")
		return
	}
	if err := h.service.Delete(uint(id64)); err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	helper.SuccessResponse(c, "删除成功")
}

// Get 获取单个
func (h *HeroHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id64 == 0 {
		helper.ValidateResponse(c, "无效的ID")
		return
	}
	m, err := h.service.Get(uint(id64))
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	helper.SuccessResponse(c, m)
}

// ListAll 获取全部（按 sort 升序）
func (h *HeroHandler) ListAll(c *gin.Context) {
	items, err := h.service.ListAll()
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "查询失败")
		return
	}
	helper.SuccessResponse(c, items)
}
