package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/cache/v9"
	lru "github.com/hashicorp/golang-lru"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/mxcd/go-config/config"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type MemoryCache struct {
	Cache           *lru.Cache
	ExpireableCache *expirable.LRU[string, interface{}]
	Name            string
	LocalSize       int
	Backed          bool
	TTL             time.Duration
}

type MemoryCacheConfig struct {
	LocalSize int
	Backed    bool
	TTL       int
}

var memoryCaches = map[string]*MemoryCache{}
var redisCache *cache.Cache

func InitMemoryCaches() {
	ctx := context.Background()
	REDIS_HOST := config.Get().String("REDIS_HOST")
	REDIS_PORT := config.Get().Int("REDIS_PORT")
	REDIS_PASSWORD := config.Get().String("REDIS_PASSWORD")

	redisUrl := fmt.Sprintf("%s:%d", REDIS_HOST, REDIS_PORT)

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisUrl,
		Password: REDIS_PASSWORD,
		DB:       0,
	})

	status := rdb.Ping(ctx)
	if status.Err() != nil {
		log.Fatal().Err(status.Err()).Msg("Failed to connect to redis")
	}
	redisCache = cache.New(&cache.Options{
		Redis: rdb,
	})

	DefineCaches()
}

func DefineCaches() {
	defineCache("timeOffsetCache", &MemoryCacheConfig{
		LocalSize: 100,
		Backed:    false,
		TTL:       30,
	})
	defineCache("projectTagCache", &MemoryCacheConfig{
		LocalSize: 100,
		Backed:    false,
		TTL:       0,
	})
	defineCache("photographerTagCache", &MemoryCacheConfig{
		LocalSize: 100,
		Backed:    false,
		TTL:       0,
	})
	defineCache("dateTagCache", &MemoryCacheConfig{
		LocalSize: 100,
		Backed:    false,
		TTL:       0,
	})
	defineCache("weekdayTagCache", &MemoryCacheConfig{
		LocalSize: 100,
		Backed:    false,
		TTL:       0,
	})

	defineCache("scaledImageCache", &MemoryCacheConfig{
		LocalSize: 50,
		Backed:    true,
		TTL:       3600 * 24 * 7,
	})
	defineCache("scaledThumbnailImageCache", &MemoryCacheConfig{
		LocalSize: 250,
		Backed:    true,
		TTL:       3600 * 24 * 7,
	})
}

func defineCache(name string, config *MemoryCacheConfig) {
	memoryCache := &MemoryCache{
		LocalSize: config.LocalSize,
		Backed:    config.Backed,
		TTL:       time.Second * time.Duration(config.TTL),
	}
	if config.TTL > 0 {
		memoryCache.ExpireableCache = expirable.NewLRU[string, interface{}](config.LocalSize, nil, memoryCache.TTL)
	} else {
		memoryCache.Cache, _ = lru.New(config.LocalSize)
	}
	memoryCaches[name] = memoryCache
}

func getCache(name string) *MemoryCache {
	return memoryCaches[name]
}

func GetCacheItem[T any](ctx context.Context, name string, key string, value *T) bool {
	memoryCache := getCache(name)
	if memoryCache == nil {
		log.Fatal().Msgf("Cache %s not defined", name)
	}

	var ok bool
	var rawItem interface{}
	if memoryCache.ExpireableCache != nil {
		rawItem, ok = memoryCache.ExpireableCache.Get(key)
	} else {
		rawItem, ok = memoryCache.Cache.Get(key)
	}

	if ok {
		var item *T = rawItem.(*T)
		if item == nil {
			return false
		}
		*value = *item
		log.Trace().Msgf("LOCAL cache hit for %s:%s", name, key)
		return true
	}

	if memoryCache.Backed {
		err := redisCache.Get(ctx, key, &value)
		if err != nil {
			log.Trace().Msgf("REDIS cache miss for %s", key)
			return false
		}
		log.Trace().Msgf("REDIS cache hit for %s", key)
		if memoryCache.ExpireableCache != nil {
			memoryCache.ExpireableCache.Add(key, value)
		} else {
			memoryCache.Cache.Add(key, value)
		}
		return true
	}
	return false
}

func SetCacheItem[T any](ctx context.Context, name string, key string, value *T) error {
	memoryCache := getCache(name)
	if memoryCache == nil {
		log.Fatal().Msgf("Cache %s not defined", name)
	}
	log.Trace().Msgf("Setting cache item %s", key)

	if memoryCache.ExpireableCache != nil {
		memoryCache.ExpireableCache.Add(key, value)
	} else {
		memoryCache.Cache.Add(key, value)
	}

	if memoryCache.Backed {
		log.Trace().Msgf("Setting redis cache item %s", key)
		err := redisCache.Set(&cache.Item{
			Ctx:   ctx,
			Key:   key,
			Value: value,
			TTL:   memoryCache.TTL,
		})
		if err != nil {
			log.Err(err).Msgf("Error setting redis cache item %s", key)
			return err
		}
	}
	return nil
}

func GetImageCacheKey(cacheName string, id string, size uint) string {
	return fmt.Sprintf("%s:%d:%s", cacheName, size, id)
}

func GetCacheKey(cacheName string, id string) string {
	return fmt.Sprintf("%s:%s", cacheName, id)
}
