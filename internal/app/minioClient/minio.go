package minio

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const ConstructionsBucket = "constructions"

func NewMinioClient(endpoint, accessKey, secretKey string, useSSL bool) (*minio.Client, error) {
	return minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
}

func InitMinio() (*minio.Client, error) {
	host := os.Getenv("MINIO_HOST")
	port := os.Getenv("MINIO_PORT")
	user := os.Getenv("MINIO_USER")
	pass := os.Getenv("MINIO_PASS")
	mc, err := NewMinioClient(host+":"+port, user, pass, false)
	if err != nil {
		return nil, err
	}
	return mc, nil
}

func EnsureBucket(ctx context.Context, mc *minio.Client, bucket string) error {
	exists, err := mc.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if !exists {
		return mc.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
	}
	return nil
}

func GenerateObjectName(originalName string) string {
	ext := filepath.Ext(originalName)
	return fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
}

func UploadFile(ctx context.Context, mc *minio.Client, bucket string, file *multipart.FileHeader) (string, error) {
	f, err := file.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()

	contentType := file.Header.Get("Content-Type")
	objectName := GenerateObjectName(file.Filename)

	_, err = mc.PutObject(ctx, bucket, objectName, f, file.Size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}
	return objectName, nil
}

func UploadFromReader(ctx context.Context, mc *minio.Client, bucket, objectName string, r io.Reader, size int64, contentType string) error {
	_, err := mc.PutObject(ctx, bucket, objectName, r, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func DeleteObject(ctx context.Context, mc *minio.Client, bucket, objectName string) error {
	return mc.RemoveObject(ctx, bucket, objectName, minio.RemoveObjectOptions{})
}
