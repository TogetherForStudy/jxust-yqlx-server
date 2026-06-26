package handlers

import (
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"

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
	if userID == 0 {
		helper.HandleErrCode(c, constant.AuthMissingUserContext)
		return
	}

	err := h.pomodoroService.IncrementPomodoroCount(c, userID)
	if err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessResponse(c, gin.H{"message": "番茄钟次数已增加"})
}

// GetCount 获取当前用户番茄钟次数
func (h *PomodoroHandler) GetCount(c *gin.Context) {
	userID := helper.GetUserID(c)
	if userID == 0 {
		helper.HandleErrCode(c, constant.AuthMissingUserContext)
		return
	}

	count, err := h.pomodoroService.GetPomodoroCount(c, userID)
	if err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessResponse(c, gin.H{"pomodoro_count": count})
}

// GetRanking 获取番茄钟排名
func (h *PomodoroHandler) GetRanking(c *gin.Context) {
	result, err := h.pomodoroService.GetPomodoroRanking(c)
	if err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessResponse(c, result)
}
