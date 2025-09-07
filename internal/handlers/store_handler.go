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

func (h *StoreHandler) DeleteFile(c *gin.Context) {
	resourceID := c.Param("resource_id")
	if err := h.s3Service.DeleteObject(c.Request.Context(), resourceID); err != nil {
		logger.Errorf("file delete error: %+v, RequestID: %s", err, helper.GetRequestID(c))
		helper.ErrorResponse(c, http.StatusInternalServerError, "failed to delete file")
		return
	}
	helper.SuccessResponse(c, gin.H{"message": "file deleted successfully"})
}

func (h *StoreHandler) ListFiles(c *gin.Context) {
	files, err := h.s3Service.ListObjects(c.Request.Context())
	if err != nil {
		logger.Errorf("file list error: %+v, RequestID: %s", err, helper.GetRequestID(c))
		helper.ErrorResponse(c, http.StatusInternalServerError, "failed to list files")
		return
	}
	helper.SuccessResponse(c, files)
}

func (h *StoreHandler) ListExpiredFiles(c *gin.Context) {
	files, err := h.s3Service.ListExpiredObjects(c.Request.Context())
	if err != nil {
		logger.Errorf("expired file list error: %+v, RequestID: %s", err, helper.GetRequestID(c))
		helper.ErrorResponse(c, http.StatusInternalServerError, "failed to list expired files")
		return
	}
	helper.SuccessResponse(c, files)
}

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
