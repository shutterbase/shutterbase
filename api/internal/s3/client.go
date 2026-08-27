package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
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
	if s == nil {
		// tests run without an S3 client; the serializer logs and skips
		return "", fmt.Errorf("no s3 client configured")
	}
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

// GetObjectIds maps each requested thumbnail size to its S3 object key for a
// storageId. Layout (SPEC §4.3): "XX/<storageId>.jpg" for size 0 ("original"),
// "XX/<storageId>-<size>.jpg" otherwise, where XX = first two chars of storageId.
// Size 0 is always included as the original alongside the passed sizes.
func GetObjectIds(storageId string, sizes []int) map[int]string {
	prefix := storageId
	if len(storageId) > 2 {
		prefix = storageId[:2]
	}
	keys := map[int]string{0: fmt.Sprintf("%s/%s.jpg", prefix, storageId)}
	for _, size := range sizes {
		keys[size] = fmt.Sprintf("%s/%s-%d.jpg", prefix, storageId, size)
	}
	return keys
}

// PresignGet mints an uncached presigned GET with the given expiry, optionally
// forcing a download filename via response-content-disposition. Callers that
// want memoization (the public gallery's short-lived preview URLs) keep their
// own cache keyed by (object, expiry).
func (s *S3Client) PresignGet(ctx context.Context, objectName string, expiry time.Duration, downloadName string) (string, error) {
	params := make(map[string][]string)
	if downloadName != "" {
		params["response-content-disposition"] = []string{fmt.Sprintf("attachment; filename=%q", downloadName)}
	}
	url, err := s.Client.PresignedGetObject(ctx, s.Options.Bucket, objectName, expiry, params)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

// GetObjectToFile streams an object into path, capped at maxBytes (<= 0 is
// uncapped), without holding it in memory. Returns ErrObjectTooLarge past the
// cap; the partial file is removed on any error.
func (s *S3Client) GetObjectToFile(ctx context.Context, objectName string, maxBytes int64, path string) (int64, error) {
	obj, err := s.Client.GetObject(ctx, s.Options.Bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return 0, err
	}
	defer obj.Close()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return 0, err
	}
	var r io.Reader = obj
	if maxBytes > 0 {
		r = io.LimitReader(obj, maxBytes+1)
	}
	n, err := io.Copy(f, r)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && maxBytes > 0 && n > maxBytes {
		err = ErrObjectTooLarge
	}
	if err != nil {
		_ = os.Remove(path)
		return 0, err
	}
	return n, nil
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
