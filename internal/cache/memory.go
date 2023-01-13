package cache

import "github.com/guyfedwards/nom/internal/rss"

type MemoryCache struct {
	content CacheContent
}

func (c *MemoryCache) Write(key string, content rss.RSS) error {
	if c.content == nil {
		c.content = make(CacheContent)
	}
	c.content[key] = content
	return nil
}

func (c *MemoryCache) Read(key string) (rss.RSS, error) {
	if _, ok := c.content[key]; !ok {
		return rss.RSS{}, ErrCacheMiss
	}

	return c.content[key], nil
}
