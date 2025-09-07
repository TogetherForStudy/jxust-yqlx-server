package handlers

import (
	"net/http"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
	"github.com/gin-gonic/gin"
)

type OSSHandler struct {
	service *services.OSSService
}

func NewOSSHandler(service *services.OSSService) *OSSHandler {
	return &OSSHandler{service: service}
}

// GetToken 生成 CDN/OSS 访问签名
func (h *OSSHandler) GetToken(c *gin.Context) {
	var req request.OSSGetTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ValidateResponse(c, "参数验证失败")
		return
	}

	token, expireAt, signedURL, err := h.service.GenerateToken(req.URI, req.ExpireSeconds)
	if err != nil {
		helper.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	helper.SuccessResponse(c, response.OSSGetTokenResponse{
		TokenParam: token,
		ExpireAt:   expireAt,
		SignedURL:  signedURL,
	})
}
