package handlers

import (
	"net/http"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"

	"github.com/gin-gonic/gin"
)

type PomodoroHandler struct {
	pomodoroService *services.PomodoroService
}

func NewPomodoroHandler(pomodoroService *services.PomodoroService) *PomodoroHandler {
	return &PomodoroHandler{
		pomodoroService: pomodoroService,
	}
}

// Increment 增加番茄钟次数
func (h *PomodoroHandler) Increment(c *gin.Context) {
	userID := helper.GetUserID(c)

	err := h.pomodoroService.IncrementPomodoroCount(c, userID)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "番茄钟次数已增加"})
}

// GetRanking 获取番茄钟排名
func (h *PomodoroHandler) GetRanking(c *gin.Context) {
	result, err := h.pomodoroService.GetPomodoroRanking(c)
	if err != nil {
		helper.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	helper.SuccessResponse(c, result)
}
