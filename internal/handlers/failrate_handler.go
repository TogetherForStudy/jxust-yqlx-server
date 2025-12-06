package handlers

import (
	"net/http"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
	"github.com/gin-gonic/gin"
)

type FailRateHandler struct {
	service *services.FailRateService
}

func NewFailRateHandler(service *services.FailRateService) *FailRateHandler {
	return &FailRateHandler{service: service}
}

// SearchFailRate 搜索接口：按课程名关键词分页，默认 failrate 降序
func (h *FailRateHandler) SearchFailRate(c *gin.Context) {
	var req request.SearchFailRateRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 10
	}

	list, total, err := h.service.Search(c, req.Keyword, req.Page, req.Size)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "查询失败")
		return
	}

	items := make([]response.FailRateItem, 0, len(list))
	for _, it := range list {
		items = append(items, response.FailRateItem{
			ID:         it.ID,
			CourseName: it.CourseName,
			Department: it.Department,
			Semester:   it.Semester,
			FailRate:   it.FailRate,
		})
	}

	helper.SuccessResponse(c, response.FailRateListResponse{
		List:  items,
		Total: total,
		Page:  req.Page,
		Size:  req.Size,
	})
}

// RandFailRate 随机返回10条
func (h *FailRateHandler) RandFailRate(c *gin.Context) {
	list, err := h.service.Rand(c, 10)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "查询失败")
		return
	}

	items := make([]response.FailRateItem, 0, len(list))
	for _, it := range list {
		items = append(items, response.FailRateItem{
			ID:         it.ID,
			CourseName: it.CourseName,
			Department: it.Department,
			Semester:   it.Semester,
			FailRate:   it.FailRate,
		})
	}

	helper.SuccessResponse(c, response.FailRateListResponse{List: items})
}
