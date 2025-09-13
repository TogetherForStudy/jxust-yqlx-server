package handlers

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/services"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"
	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
)

type StoreHandler struct {
	s3Service services.S3ServiceInterface
}

func NewStoreHandler(s3Service services.S3ServiceInterface) *StoreHandler {
	return &StoreHandler{
		s3Service: s3Service,
	}
}

// UploadFile 上传文件
// @Summary 上传文件                                                                                             │
// @Description 管理员上传文件                                                                                   │
// @Tags 管理员                                                                                                  │
// @Accept multipart/form-data                                                                                   │
// @Produce json                                                                                                 │
// @Security ApiKeyAuth                                                                                          │
// @Param file formData file true "文件"                                                                         │
// @Param tags formData string false "标签 (json 格式)"                                                          │
// @Success 200 {object} response.UploadFileResponse                                                             │
// @Failure 400 {object} utils.Response                                                                          │
// @Failure 500 {object} utils.Response                                                                          │
// @Router /api/v0/store [post]
func (h *StoreHandler) UploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		logger.Errorf("file upload error at http read file: %+v, RequestID: %s", err, helper.GetRequestID(c))
		helper.ErrorResponse(c, http.StatusBadRequest, "file upload failed")
		return
	}

	src, err := file.Open()
	if err != nil {
		logger.Errorf("file upload error at open upload file: %+v, RequestID: %s", err, helper.GetRequestID(c))
		helper.ErrorResponse(c, http.StatusInternalServerError, "failed to open file")
		return
	}
	defer src.Close()

	tagsStr := c.PostForm("tags")
	var tags map[string]string
	if tagsStr != "" {
		if err := sonic.UnmarshalString(tagsStr, &tags); err != nil {
			logger.Errorf("file upload error at unmarshal tags: %+v, RequestID: %s", err, helper.GetRequestID(c))
			helper.ErrorResponse(c, http.StatusBadRequest, "invalid tags format")
			return
		}
	}

	resourceID, err := h.s3Service.AddObject(c.Request.Context(), src, file.Filename, file.Header.Get("Content-Type"), true, nil, tags)
	if err != nil {
		logger.Errorf("file upload error at minio add object: %+v, RequestID: %s", err, helper.GetRequestID(c))
		helper.ErrorResponse(c, http.StatusInternalServerError, "failed to store file")
		return
	}

	helper.SuccessResponse(c, response.UploadFileResponse{ResourceID: resourceID})
}

// DeleteFile 删除文件                                                                                           │
// @Summary 删除文件                                                                                             │
// @Description 管理员删除文件                                                                                   │
// @Tags 管理员                                                                                                  │
// @Accept json                                                                                                  │
// @Produce json                                                                                                 │
// @Security ApiKeyAuth                                                                                          │
// @Param resource_id path string true "资源ID"                                                                  │
// @Success 200 {object} utils.Response                                                                          │
// @Failure 500 {object} utils.Response                                                                          │
// @Router /api/v0/store/{resource_id} [delete]
func (h *StoreHandler) DeleteFile(c *gin.Context) {
	resourceID := c.Param("resource_id")
	if err := h.s3Service.DeleteObject(c.Request.Context(), resourceID); err != nil {
		logger.Errorf("file delete error: %+v, RequestID: %s", err, helper.GetRequestID(c))
		helper.ErrorResponse(c, http.StatusInternalServerError, "failed to delete file")
		return
	}
	helper.SuccessResponse(c, gin.H{"message": "file deleted successfully"})
}

// ListFiles 获取文件列表                                                                                        │
// @Summary 获取文件列表                                                                                         │
// @Description 管理员获取文件列表                                                                               │
// @Tags 管理员                                                                                                  │
// @Accept json                                                                                                  │
// @Produce json                                                                                                 │
// @Security ApiKeyAuth                                                                                          │
// @Success 200 {object} response.ListFilesResponse                                                              │
// @Failure 500 {object} utils.Response                                                                          │
// @Router /api/v0/store [get]
func (h *StoreHandler) ListFiles(c *gin.Context) {
	files, err := h.s3Service.ListObjects(c.Request.Context())
	if err != nil {
		logger.Errorf("file list error: %+v, RequestID: %s", err, helper.GetRequestID(c))
		helper.ErrorResponse(c, http.StatusInternalServerError, "failed to list files")
		return
	}
	helper.SuccessResponse(c, files)
}

// ListExpiredFiles 获取过期文件列表                                                                             │
// @Summary 获取过期文件列表                                                                                     │
// @Description 管理员获取过期文件列表                                                                           │
// @Tags 管理员                                                                                                  │
// @Accept json                                                                                                  │
// @Produce json                                                                                                 │
// @Security ApiKeyAuth                                                                                          │
// @Success 200 {object} response.ListFilesResponse                                                              │
// @Failure 500 {object} utils.Response                                                                          │
// @Router /api/v0/store/expired [get]
func (h *StoreHandler) ListExpiredFiles(c *gin.Context) {
	files, err := h.s3Service.ListExpiredObjects(c.Request.Context())
	if err != nil {
		logger.Errorf("expired file list error: %+v, RequestID: %s", err, helper.GetRequestID(c))
		helper.ErrorResponse(c, http.StatusInternalServerError, "failed to list expired files")
		return
	}
	helper.SuccessResponse(c, files)
}

// GetFileURL 获取文件URL
// @Summary 获取文件URL
// @Description 获取文件访问URL
// @Tags 存储
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param resource_id path string true "资源ID"
// @Param expires query int false "过期时间（分钟）" default(60)
// @Param download query bool false "是否为下载链接" default(false)
// @Success 200 {object} response.GetFileURLResponse
// @Failure 500 {object} utils.Response
// @Router /api/v0/store/{resource_id}/url [get]
func (h *StoreHandler) GetFileURL(c *gin.Context) {
	var req request.GetFileURLRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		logger.Errorf("get file url error at bind arg: %+v, RequestID: %s", err, helper.GetRequestID(c))
		helper.ValidateResponse(c, "invalid request parameters")
		return
	}

	resourceID := c.Param("resource_id")
	expires := time.Duration(req.Expires) * time.Minute
	if req.Expires == 0 {
		expires = constant.DefaultExpired
	}

	url, err := h.s3Service.ShareObject(c.Request.Context(), resourceID, &expires, req.Download)
	if err != nil {
		logger.Errorf("get file url error at minio share object: %+v, RequestID: %s", err, helper.GetRequestID(c))
		helper.ErrorResponse(c, http.StatusInternalServerError, "failed to get file url")
		return
	}

	helper.SuccessResponse(c, response.GetFileURLResponse{URL: url})
}

// GetFileStream 获取文件流
// @Summary 获取文件流
// @Description 获取文件流
// @Tags 存储
// @Accept json
// @Produce octet-stream
// @Security ApiKeyAuth
// @Param resource_id path string true "资源ID"
// @Success 200
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v0/store/{resource_id}/stream [get]
func (h *StoreHandler) GetFileStream(c *gin.Context) {
	resourceID := c.Param("resource_id")
	obj, s3Data, err := h.s3Service.GetObject(c.Request.Context(), resourceID)
	if err != nil {
		logger.Errorf("file stream error at minio get object: %+v, RequestID: %s", err, helper.GetRequestID(c))
		helper.ErrorResponse(c, http.StatusNotFound, "file not found")
		return
	}
	defer obj.Close()

	c.Header("Content-Disposition", "attachment; filename="+s3Data.FileName)
	c.Header("Content-Type", s3Data.MimeType)
	c.Header("Content-Length", strconv.FormatInt(s3Data.FileSize, 10))

	_, err = io.Copy(c.Writer, obj)
	if err != nil {
		logger.Errorf("file stream error at io copy: %+v, RequestID: %s", err, helper.GetRequestID(c))
		helper.ErrorResponse(c, http.StatusInternalServerError, "failed to stream file")
		return
	}
}
