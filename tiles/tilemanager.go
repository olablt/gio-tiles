package tiles

import (
	"fmt"
	"image"
	_ "image/png"

	"gioui.org/op/paint"
)

type TileProvider interface {
	GetTile(tile Tile) (image.Image, error)
}

type TileManager struct {
	cache    Cache
	provider TileProvider
	onLoad   func()
}

func NewTileManager(provider TileProvider, cacheType CacheType) *TileManager {
	var cache Cache
	switch cacheType {
	case CacheImageOp:
		cache = NewImageOpCache()
	default:
		cache = NewImageCache()
	}

	return &TileManager{
		cache:    cache,
		provider: provider,
	}
}

func (tm *TileManager) GetCache() Cache {
	return tm.cache
}

func (tm *TileManager) SetOnLoadCallback(callback func()) {
	tm.onLoad = callback
}

// getTileKey returns a unique string key for a tile
func GetTileKey(tile Tile) string {
	return fmt.Sprintf("%d/%d/%d", tile.Zoom, tile.X, tile.Y)
}

func (tm *TileManager) GetTile(tile Tile) (image.Image, error) {
	key := GetTileKey(tile)

	// Check cache first
	if cached, exists := tm.cache.Get(key); exists {
		switch tm.cache.GetType() {
		case CacheImage:
			if img, ok := cached.(image.Image); ok {
				return img, nil
			}
		case CacheImageOp:
			// For ImageOp cache we need to return the original image
			if _, ok := cached.(paint.ImageOp); ok {
				// Get fresh image from provider since we can't extract it from ImageOp
				return tm.provider.GetTile(tile)
			}
		}
	}

	// If not in cache, load from provider
	img, err := tm.provider.GetTile(tile)
	if err != nil {
		return nil, err
	}

	// Cache according to type
	switch tm.cache.GetType() {
	case CacheImage:
		tm.cache.Set(key, img)
	case CacheImageOp:
		tm.cache.Set(key, paint.NewImageOp(img))
	}

	if tm.onLoad != nil {
		tm.onLoad()
	}
	return img, nil
}
