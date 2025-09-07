package services

import (
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/config"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/minio"

	"gorm.io/gorm"
)

type S3Service struct {
	S3 minio.Client
	db *gorm.DB
}

func NewS3Service(db *gorm.DB, cfg *config.Config) *S3Service {
	return &S3Service{
		S3: minio.NewMinioClient(&cfg.MinIO),
		db: db,
	}
}
