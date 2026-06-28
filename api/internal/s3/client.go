package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ErrObjectTooLarge is returned by GetObject when the object exceeds maxBytes.
var ErrObjectTooLarge = errors.New("s3 object exceeds size cap")

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

// GetObject reads an object into memory, capped at maxBytes (S10: a huge object
// must not exhaust memory / the exiftool pool on /download). maxBytes <= 0 means
// uncapped. Returns ErrObjectTooLarge if the object exceeds the cap. ponytail:
// whole-object read into RAM (exiftool needs it on disk anyway); the cap is the
// guard, streaming is a later upgrade if originals ever dwarf the cap.
func (s *S3Client) GetObject(ctx context.Context, objectName string, maxBytes int64) ([]byte, error) {
	obj, err := s.Client.GetObject(ctx, s.Options.Bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer obj.Close()
	if maxBytes <= 0 {
		return io.ReadAll(obj)
	}
	// Read one extra byte to detect overflow without trusting Content-Length.
	data, err := io.ReadAll(io.LimitReader(obj, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, ErrObjectTooLarge
	}
	return data, nil
}

func (s *S3Client) DeleteImages(ctx context.Context, storageId string) error {
	// Objects live under "<id[:2]>/<id>[...].jpg" (see server.GetObjectIds), so
	// listing by the bare storageId matched nothing and orphaned every object on
	// delete. Mirror the stored key layout and recurse past the "/" delimiter.
	prefix := storageId
	if len(storageId) > 2 {
		prefix = storageId[:2] + "/" + storageId
	}
	objectsCh := s.Client.ListObjects(ctx, s.Options.Bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
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
