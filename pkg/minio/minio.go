package minio

import (
	"context"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
)

func (c *client) StoreObject(ctx context.Context, data io.ReadCloser, bucketName, objectName string) (string, error) {
	object, err := c.Client.PutObject(ctx, bucketName, objectName, data, -1, minio.PutObjectOptions{})
	if err != nil {
		return "", err
	}
	return object.Key, nil
}
func (c *client) GetObject(ctx context.Context, bucketName, objectName string) (io.ReadCloser, error) {
	object, err := c.Client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return object, nil
}
func (c *client) DeleteObject(ctx context.Context, bucketName, objectName string) error {
	return c.Client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
}
func (c *client) ObjectExists(ctx context.Context, bucketName, objectName string) (bool, error) {
	_, err := c.Client.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
func (c *client) CreateShareLink(ctx context.Context, bucketName, objectName string, time time.Duration) (string, error) {
	presignedURL, err := c.Client.PresignedGetObject(ctx, bucketName, objectName, time, nil)
	if err != nil {
		return "", err
	}
	return presignedURL.String(), nil
}
