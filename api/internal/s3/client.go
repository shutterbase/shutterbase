package s3

import (
	"context"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Client struct {
	Options *S3ClientOptions
	Client  *minio.Client
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
		Options: options,
		Client:  client,
	}, nil
}

func (s *S3Client) GetSignedUploadUrl(ctx context.Context, objectName string) (string, error) {
	url, err := s.Client.PresignedPutObject(ctx, s.Options.Bucket, objectName, time.Duration(10)*time.Minute)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}
