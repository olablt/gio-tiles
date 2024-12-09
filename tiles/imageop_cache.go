package tiles

import (
    "gioui.org/op/paint"
    "sync"
)

type ImageOpCache struct {
    cache map[string]paint.ImageOp
    mu    sync.RWMutex
}

func NewImageOpCache() *ImageOpCache {
    return &ImageOpCache{
        cache: make(map[string]paint.ImageOp),
    }
}

func (c *ImageOpCache) Get(key string) (interface{}, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    val, ok := c.cache[key]
    return val, ok
}

func (c *ImageOpCache) Set(key string, value interface{}) {
    if imageOp, ok := value.(paint.ImageOp); ok {
        c.mu.Lock()
        c.cache[key] = imageOp
        c.mu.Unlock()
    }
}

func (c *ImageOpCache) Clear() {
    c.mu.Lock()
    c.cache = make(map[string]paint.ImageOp)
    c.mu.Unlock()
}

func (c *ImageOpCache) GetType() CacheType {
    return CacheImageOp
}
