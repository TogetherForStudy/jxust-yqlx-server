package services

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/config"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/minio"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type S3Service struct {
	S3 minio.Client
	db *gorm.DB

	rootBucketName string
	host           string
	scheme         string
}

type S3ServiceInterface interface {
	AddObject(ctx context.Context, data io.ReadCloser,
		fileName string, mimeType string, isAdmin bool,
		customPath *string, tags map[string]string) (string, error)
	DeleteObject(ctx context.Context, resourceID string) error
	ShareObject(ctx context.Context, resourceID string, expires *time.Duration, download bool) (string, error)
	ListObjects(ctx context.Context) ([]models.S3Data, error)
	ListExpiredObjects(ctx context.Context) ([]models.S3Resource, error)
	GetObject(ctx context.Context, resourceID string) (io.ReadCloser, *models.S3Data, error)
}

func NewS3Service(db *gorm.DB, cfg *config.Config) S3ServiceInterface {
	return &S3Service{
		S3:             minio.NewMinioClient(&cfg.MinIO),
		db:             db,
		rootBucketName: cfg.MinIO.BucketName,
		host:           cfg.Host,
		scheme:         cfg.Scheme,
	}
}

// AddObject adds an object to S3 and returns the resource ID.
func (s *S3Service) AddObject(ctx context.Context, data io.ReadCloser,
	fileName string, mimeType string, isAdmin bool, customPath *string, tags map[string]string) (string, error) {
	resourceID := uuid.NewString()
	var objectKey string

	if customPath != nil && *customPath != "" {
		objectKey = fmt.Sprintf("%s/%s", *customPath, resourceID)
	} else if isAdmin {
		objectKey = fmt.Sprintf("admin/%s", resourceID)
	} else {
		objectKey = fmt.Sprintf("user/%s", resourceID)
	}

	info, err := s.S3.PutObject(ctx, s.rootBucketName, objectKey, data, -1, minio.PutObjectOptions{
		ContentType: mimeType,
		UserTags:    tags,
	})
	if err != nil {
		return "", err
	}

	if tags == nil {
		tags = make(map[string]string)
	}
	tag, err := sonic.MarshalString(tags)
	if err != nil {
		return "", err
	}

	s3Data := &models.S3Data{
		ResourceID: resourceID,
		Bucket:     s.rootBucketName,
		ObjectKey:  objectKey,
		FileSize:   info.Size,
		FileName:   fileName,
		MimeType:   mimeType,
		Tag:        &tag,
	}

	if err := s.db.WithContext(ctx).Create(s3Data).Error; err != nil {
		// If database insertion fails, delete the object from S3 to maintain consistency.
		_ = s.S3.DeleteObject(ctx, s.rootBucketName, objectKey)
		return "", err
	}

	return resourceID, nil
}

// DeleteObject deletes an object from S3 based on the resource ID.
func (s *S3Service) DeleteObject(ctx context.Context, resourceID string) error {
	var s3Data models.S3Data
	if err := s.db.WithContext(ctx).Where("resource_id = ?", resourceID).First(&s3Data).Error; err != nil {
		return err
	}

	if err := s.S3.DeleteObject(ctx, s3Data.Bucket, s3Data.ObjectKey); err != nil {
		return err
	}

	// Using transaction to delete S3Data and S3Resource
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.S3Data{}).
			Where("resource_id = ?", resourceID).
			Update("deleted_at", gorm.DeletedAt{Time: time.Now(), Valid: true}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.S3Resource{}).
			Where("resource_id = ?", resourceID).
			Update("expired_at", gorm.DeletedAt{Time: time.Now(), Valid: true}).Error; err != nil {
			return err
		}
		return nil
	})
}

// ShareObject generates a presigned URL for an object.
func (s *S3Service) ShareObject(ctx context.Context, resourceID string, expires *time.Duration, download bool) (string, error) {
	var s3Data models.S3Data
	if err := s.db.WithContext(ctx).Where("resource_id = ?", resourceID).First(&s3Data).Error; err != nil {
		return "", err
	}

	if expires == nil || *expires == 0 {
		expiredAt := constant.DefaultExpired
		expires = &expiredAt
	}

	reqParams := make(url.Values)
	if download {
		reqParams.Set("response-content-disposition", fmt.Sprintf("attachment; filename=\"%s\"", s3Data.FileName))
	}

	presignedURL, err := s.S3.PresignedGetObject(ctx, s3Data.Bucket, s3Data.ObjectKey, *expires, reqParams)
	if err != nil {
		return "", err
	}
	presignedURL.Host = s.host
	presignedURL.Scheme = s.scheme

	userId := ctx.Value("open_id").(string) // wechat user openid
	expiredAt := time.Now().Add(*expires)
	s3Resource := &models.S3Resource{
		ResourceID: resourceID,
		URL:        presignedURL.String(),
		UserID:     userId,
		ExpiredAt:  gorm.DeletedAt{Time: expiredAt, Valid: true},
	}

	if err := s.db.WithContext(ctx).Create(s3Resource).Error; err != nil {
		return "", err
	}

	return presignedURL.String(), nil
}

// ListObjects lists all objects in the S3Data table.
func (s *S3Service) ListObjects(ctx context.Context) ([]models.S3Data, error) {
	var s3Data []models.S3Data
	if err := s.db.WithContext(ctx).Find(&s3Data).Error; err != nil {
		return nil, err
	}
	return s3Data, nil
}

// ListExpiredObjects lists all expired objects in the S3Resource table.
func (s *S3Service) ListExpiredObjects(ctx context.Context) ([]models.S3Resource, error) {
	var s3Resources []models.S3Resource
	if err := s.db.WithContext(ctx).Unscoped().Where("expired_at < ?", time.Now()).Find(&s3Resources).Error; err != nil {
		return nil, err
	}
	return s3Resources, nil
}

// GetObject retrieves an object from S3 based on the resource ID.
func (s *S3Service) GetObject(ctx context.Context, resourceID string) (io.ReadCloser, *models.S3Data, error) {
	var s3Data models.S3Data
	if err := s.db.WithContext(ctx).Where("resource_id = ?", resourceID).First(&s3Data).Error; err != nil {
		return nil, nil, err
	}

	obj, err := s.S3.GetObject(ctx, s3Data.Bucket, s3Data.ObjectKey)
	if err != nil {
		return nil, nil, err
	}
	return obj, &s3Data, nil
}
