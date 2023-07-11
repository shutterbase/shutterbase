package storage

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"

	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"
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

	data, ok := fileCache.Get(id.String())
	if ok {
		log.Debug().Str("id", id.String()).Msg("file cache hit")
		return &data, nil
	}
	log.Debug().Str("id", id.String()).Msg("lru cache miss")

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

	buf, err := ioutil.ReadAll(object)
	if err != nil {
		log.Error().Err(err).Msg("failed to read object from s3")
		return nil, err
	}

	go cacheFile(id, &buf)

	return &buf, nil
}

func cacheFile(id uuid.UUID, data *[]byte) {
	megabyteSize := float64(len(*data)) / (1024 * 1024)
	log.Debug().Str("id", id.String()).Msgf("caching file with %.2fMB", megabyteSize)
	fileCache.Add(id.String(), *data)
}

func PutFile(ctx context.Context, id uuid.UUID, data []byte) error {
	megabyteSize := float64(len(data)) / (1024 * 1024)
	log.Debug().Str("id", id.String()).Msgf("putting file with %.2fMB to s3", megabyteSize)
	reader := bytes.NewReader(data)
	_, err := s3Client.PutObject(ctx, S3_BUCKET, id.String(), reader, int64(len(data)), minio.PutObjectOptions{})
	if err != nil {
		log.Error().Err(err).Msg("failed to put object to s3")
		return err
	}

	go cacheFile(id, &data)
	return nil
}
