package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"
	"github.com/shutterbase/shutterbase/internal/tracing"
)

var S3_BUCKET string
var s3Client *minio.Client
var fileCache *lru.Cache[string, []byte]

func Init() error {
	S3_HOST := config.Get().String("S3_HOST")
	S3_PORT := config.Get().Int("S3_PORT")
	S3_SSL := config.Get().Bool("S3_SSL")
	S3_ACCESS_KEY := config.Get().String("S3_ACCESS_KEY")
	S3_SECRET_KEY := config.Get().String("S3_SECRET_KEY")
	S3_BUCKET = config.Get().String("S3_BUCKET")

	endpoint := fmt.Sprintf("%s:%d", S3_HOST, S3_PORT)

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(S3_ACCESS_KEY, S3_SECRET_KEY, ""),
		Secure: S3_SSL,
	})

	if err != nil {
		return err
	}

	s3Client = client

	LRU_CACHE_SIZE := config.Get().Int("LRU_CACHE_SIZE")
	cache, err := lru.New[string, []byte](LRU_CACHE_SIZE)
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating file cache")
	}
	fileCache = cache

	return nil
}

func GetFile(ctx context.Context, id uuid.UUID) (*[]byte, error) {
	ctx, tracer := tracing.GetTracer().Start(ctx, "download_file")
	defer tracer.End()
	data, ok := fileCache.Get(id.String())
	if ok {
		log.Debug().Str("id", id.String()).Msg("file cache hit")
		return &data, nil
	}
	log.Debug().Str("id", id.String()).Msg("lru cache miss")

	startTime := time.Now()
	object, err := s3Client.GetObject(ctx, S3_BUCKET, id.String(), minio.GetObjectOptions{})
	if err != nil {
		log.Error().Err(err).Msg("failed to get object from s3")
		return nil, err
	}
	defer object.Close()

	file, err := object.Stat()
	if err != nil {
		log.Error().Err(err).Msg("failed to get object stats from s3")
		return nil, err
	}
	log.Debug().Str("id", id.String()).Msgf("file size: %d", file.Size)

	buf, err := io.ReadAll(object)
	if err != nil {
		log.Error().Err(err).Msg("failed to read object from s3")
		return nil, err
	}

	log.Debug().Str("id", id.String()).Msgf("downloaded file with %.2fMB in %.2fs", float64(len(buf))/(1024*1024), time.Since(startTime).Seconds())

	go cacheFile(id, &buf)

	return &buf, nil
}

func cacheFile(id uuid.UUID, data *[]byte) {
	megabyteSize := float64(len(*data)) / (1024 * 1024)
	log.Debug().Str("id", id.String()).Msgf("caching file with %.2fMB", megabyteSize)
	fileCache.Add(id.String(), *data)
}

func PutFile(ctx context.Context, id uuid.UUID, data []byte) error {
	ctx, tracer := tracing.GetTracer().Start(ctx, "upload_file")
	defer tracer.End()
	log.Debug().Str("id", id.String()).Msgf("putting file with %s to s3", getHumanReadableSize(int64(len(data))))
	reader := bytes.NewReader(data)
	_, err := s3Client.PutObject(ctx, S3_BUCKET, id.String(), reader, int64(len(data)), minio.PutObjectOptions{})
	if err != nil {
		log.Error().Err(err).Msg("failed to put object to s3")
		return err
	}

	go cacheFile(id, &data)
	return nil
}

func getHumanReadableSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.2f KiB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MiB", float64(size)/(1024*1024))
	} else if size < 1024*1024*1024*1024 {
		return fmt.Sprintf("%.2f GiB", float64(size)/(1024*1024*1024))
	} else {
		return fmt.Sprintf("%.2f TiB", float64(size)/(1024*1024*1024*1024))
	}
}
