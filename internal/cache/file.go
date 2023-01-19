package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/guyfedwards/nom/internal/rss"
)

var DefaultPath = filepath.Join(os.TempDir(), "nom")

type FileCache struct {
	expiry  time.Duration
	path    string
	content CacheContent
}

// Write writes content to a file at the location specified in the cache
func (c *FileCache) Write(key string, content rss.RSS) error {
	err := createCacheIfNotExists(c.path)
	if err != nil {
		return fmt.Errorf("createcache: %w", err)
	}

	data, err := os.ReadFile(filepath.Join(c.path, "cache.json"))
	if err != nil {
		return fmt.Errorf("cache Write: %w", err)
	}

	var cc CacheContent

	err = json.Unmarshal(data, &cc)
	if err != nil {
		return fmt.Errorf("cache Write json unmarshal: %w", err)
	}

	cc[key] = content

	str, err := json.Marshal(cc)
	if err != nil {
		return fmt.Errorf("cache write marshal json: %w", err)
	}

	err = os.WriteFile(filepath.Join(c.path, "cache.json"), str, 0655)
	if err != nil {
		return fmt.Errorf("cache Write: %w", err)
	}

	return nil
}

// Read reads from the cache, returning a ErrFileCacheMiss if nothing found or
// if the cache is older than the expiry
func (c *FileCache) Read(key string) (rss.RSS, error) {
	err := createCacheIfNotExists(c.path)
	if err != nil {
		return rss.RSS{}, fmt.Errorf("cache read: %w", err)
	}

	data, err := os.ReadFile(filepath.Join(c.path, "cache.json"))
	if err != nil {
		return rss.RSS{}, fmt.Errorf("cache read file: %w", err)
	}

	var cc CacheContent

	err = json.Unmarshal(data, &cc)
	if err != nil {
		return rss.RSS{}, fmt.Errorf("cache read unmarshal: %w", err)
	}

	if _, ok := cc[key]; !ok {
		return rss.RSS{}, ErrCacheMiss
	}

	return cc[key], nil
}

func createCacheIfNotExists(path string) error {
	cachePath := filepath.Join(path, "cache.json")
	info, _ := os.Stat(cachePath)
	if info != nil {
		return nil
	}

	fmt.Println("No existing cache found, creating")

	err := os.MkdirAll(path, 0755)
	if err != nil {
		return fmt.Errorf("createDirIfNotExists: %w", err)
	}

	var cc = make(CacheContent)

	str, err := json.Marshal(cc)
	if err != nil {
		return fmt.Errorf("createDirIfNotExists: %w", err)
	}

	err = os.WriteFile(cachePath, []byte(str), 0655)
	if err != nil {
		return fmt.Errorf("createDirIfNotExists: %w", err)
	}

	return nil
}
