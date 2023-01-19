package cache

import "github.com/guyfedwards/nom/internal/rss"

type MemoryCache struct {
	content CacheContent
}

// Write writes content to a file at the location specified in the cache
func (c *MemoryCache) Write(key string, content rss.RSS) error {
	if c.content == nil {
		c.content = make(CacheContent)
	}
	c.content[key] = content
	return nil
}

// Read reads from the cache, returning a ErrFileCacheMiss if nothing found or
// if the cache is older than the expiry
func (c *MemoryCache) Read(key string) (rss.RSS, error) {
	if _, ok := c.content[key]; !ok {
		return rss.RSS{}, ErrCacheMiss
	}

	return c.content[key], nil
}
