package minio

import (
	"context"
	"errors"
	"io"
	"net/url"
	"os"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Client interface {
	PresignedGetObject(ctx context.Context, bucketName string, objectName string, expires time.Duration, reqParams url.Values) (u *url.URL, err error)
	CreateShareLink(ctx context.Context, bucketName, objectName string, time time.Duration) (string, error)

	ObjectExists(ctx context.Context, bucketName, objectName string) (bool, error)
	DeleteObject(ctx context.Context, bucketName, objectName string) error
	GetObject(ctx context.Context, bucketName, objectName string) (io.ReadCloser, error)
	StoreObject(ctx context.Context, data io.ReadCloser, bucketName, objectName string) (string, error)
}
type client struct {
	*minio.Client
}

var (
	ErrFailedCreateMinioCli = errors.New("failed to create minio client")
)

func NewMinioClient() Client {
	//todo:修改签名，从国际config加载
	endpoint := os.Getenv(constant.ENV_MINIO_ENDPOINT)
	accessKeyID := os.Getenv(constant.ENV_MINIO_ACCESS_KEY)
	secretAccessKey := os.Getenv(constant.ENV_MINIO_SECRET_KEY)
	useSSL := os.Getenv(constant.ENV_MINIO_USE_SSL) == "true"

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
