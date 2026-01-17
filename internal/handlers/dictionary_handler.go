package handlers

import (
	"net/http"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
	"github.com/gin-gonic/gin"
)

type DictionaryHandler struct {
	service *services.DictionaryService
}

func NewDictionaryHandler(service *services.DictionaryService) *DictionaryHandler {
	return &DictionaryHandler{service: service}
}

// GetRandomWord 随机获取一个词
func (h *DictionaryHandler) GetRandomWord(c *gin.Context) {
	word, err := h.service.GetRandomWord(c)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, "获取随机单词失败")
		return
	}

	helper.SuccessResponse(c, word)
}
