package repository

import (
	"log"

	lru "github.com/hashicorp/golang-lru"
)

var memoryCaches = map[string]*lru.Cache{}

func DefineCache(name string, size int) {
	cache, err := lru.New(size)
	if err != nil {
		panic(err)
	}
	memoryCaches[name] = cache
}

func getCache(name string) *lru.Cache {
	return memoryCaches[name]
}

func GetCacheItem(name string, key interface{}) (interface{}, bool) {
	cache := getCache(name)
	if cache == nil {
		return nil, false
	}
	return cache.Get(key)
}

func SetCacheItem(name string, key interface{}, value interface{}) {
	cache := getCache(name)
	if cache == nil {
		log.Fatalf("Cache %s not defined", name)
	}
	cache.Add(key, value)
}
