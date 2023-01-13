package cache

import (
	"errors"
	"time"

	"github.com/guyfedwards/nom/internal/rss"
)

// key is feedurl
type CacheContent = map[string]rss.RSS

type CacheInterface interface {
	Write(key string, content rss.RSS) error
	Read(key string) (rss.RSS, error)
}

type Cache struct {
	expiry  time.Duration
	path    string
	content CacheContent
}

var ErrCacheMiss = errors.New("cache miss")

// h * m * s * ms * μs * ns
// 24hrs
const DefaultExpiry time.Duration = 24 * 60 * 60 * 1000 * 1000 * 1000

func NewFileCache(path string, expiry time.Duration) CacheInterface {
	return &FileCache{
		expiry: expiry,
		path:   path,
	}
}

func NewMemoryCache() CacheInterface {
	return &MemoryCache{}
}
