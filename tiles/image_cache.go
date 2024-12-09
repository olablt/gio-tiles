package tiles

import (
    "image"
    "sync"
)

type ImageCache struct {
    cache map[string]image.Image
    mu    sync.RWMutex
}

func NewImageCache() *ImageCache {
    return &ImageCache{
        cache: make(map[string]image.Image),
    }
}

func (c *ImageCache) Get(key string) (interface{}, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    val, ok := c.cache[key]
    return val, ok
}

func (c *ImageCache) Set(key string, value interface{}) {
    if img, ok := value.(image.Image); ok {
        c.mu.Lock()
        c.cache[key] = img
        c.mu.Unlock()
    }
}

func (c *ImageCache) Clear() {
    c.mu.Lock()
    c.cache = make(map[string]image.Image)
    c.mu.Unlock()
}

func (c *ImageCache) GetType() CacheType {
    return CacheImage
}
