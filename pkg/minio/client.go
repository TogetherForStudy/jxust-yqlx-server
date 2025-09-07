package minio

import (
	"context"
	"errors"
	"io"
	"net/url"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/config"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type PutObjectOptions = minio.PutObjectOptions

type Client interface {
	PresignedGetObject(ctx context.Context, bucketName string, objectName string, expires time.Duration, reqParams url.Values) (u *url.URL, err error)
	CreateShareLink(ctx context.Context, bucketName, objectName string, time time.Duration) (string, error)

	ObjectExists(ctx context.Context, bucketName, objectName string) (bool, error)
	DeleteObject(ctx context.Context, bucketName, objectName string) error
	GetObject(ctx context.Context, bucketName, objectName string) (io.ReadCloser, error)
	StoreObject(ctx context.Context, data io.ReadCloser, bucketName, objectName string) (string, error)
	PutObject(ctx context.Context, bucketName string, objectName string, reader io.Reader, size int64, opts minio.PutObjectOptions) (info minio.UploadInfo, err error)
}
type client struct {
	*minio.Client
}

var (
	ErrFailedCreateMinioCli = errors.New("failed to create minio client")
)

func NewMinioClient(conf *config.MinIO) Client {
	endpoint := conf.MinIOEndpoint
	accessKeyID := conf.MinIOAccessKey
	secretAccessKey := conf.MinIOSecretKey
	useSSL := conf.MinIOUseSSL

	cli, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		logger.Errorf("Failed to create minio client: %v", err)
		return empty{err: errors.Join(err, ErrFailedCreateMinioCli)}
	}
	return &client{Client: cli}
}
