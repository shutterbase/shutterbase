package s3

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Client struct {
	Options          *S3ClientOptions
	Client           *minio.Client
	DownloadUrlCache *expirable.LRU[string, string]
}

type S3ClientOptions struct {
	Endpoint  string
	Port      int
	SSL       bool
	Bucket    string
	AccessKey string
	SecretKey string
}

func NewClient(options *S3ClientOptions) (*S3Client, error) {
	endpoint := fmt.Sprintf("%s:%d", options.Endpoint, options.Port)
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(options.AccessKey, options.SecretKey, ""),
		Secure: options.SSL,
	})
	if err != nil {
		return nil, err
	}

	return &S3Client{
		Options:          options,
		Client:           client,
		DownloadUrlCache: expirable.NewLRU[string, string](5000, nil, time.Minute*4),
	}, nil
}

func (s *S3Client) GetSignedUploadUrl(ctx context.Context, objectName string) (string, error) {
	url, err := s.Client.PresignedPutObject(ctx, s.Options.Bucket, objectName, time.Duration(4)*time.Minute)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

func (s *S3Client) GetSignedDownloadUrl(ctx context.Context, objectName string) (string, error) {
	cachedUrl, ok := s.DownloadUrlCache.Get(objectName)
	if ok {
		return cachedUrl, nil
	}
	url, err := s.Client.PresignedGetObject(ctx, s.Options.Bucket, objectName, time.Duration(4)*time.Hour, nil)
	if err != nil {
		return "", err
	}
	s.DownloadUrlCache.Add(objectName, url.String())
	return url.String(), nil
}

func (s *S3Client) DeleteImages(ctx context.Context, storageId string) error {
	objectsCh := s.Client.ListObjects(ctx, s.Options.Bucket, minio.ListObjectsOptions{
		Prefix: storageId,
	})
	for object := range objectsCh {
		if object.Err != nil {
			return object.Err
		}
		err := s.Delete(ctx, object.Key)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *S3Client) Delete(ctx context.Context, objectKey string) error {
	err := s.Client.RemoveObject(ctx, s.Options.Bucket, objectKey, minio.RemoveObjectOptions{})
	if err != nil {
		return err
	}
	return nil
}
