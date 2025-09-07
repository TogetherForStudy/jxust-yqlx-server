package minio

import (
	"context"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
)

type empty struct {
	err error
}

func (e empty) PutObject(ctx context.Context, bucketName string, objectName string, reader io.Reader, size int64, opts minio.PutObjectOptions) (info minio.UploadInfo, err error) {
	return minio.UploadInfo{}, e.err
}

func (e empty) PresignedGetObject(context.Context, string, string, time.Duration, url.Values) (u *url.URL, err error) {
	return nil, e.err
}

func (e empty) CreateShareLink(context.Context, string, string, time.Duration) (string, error) {
	return "", e.err
}

func (e empty) ObjectExists(context.Context, string, string) (bool, error) {
	return false, e.err
}

func (e empty) DeleteObject(context.Context, string, string) error {
	return e.err
}

func (e empty) GetObject(context.Context, string, string) (io.ReadCloser, error) {
	return nil, e.err
}

func (e empty) StoreObject(context.Context, io.ReadCloser, string, string) (string, error) {
	return "", e.err
}
