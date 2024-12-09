package tiles

import (
	"fmt"
	"image"
	_ "image/png"
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
func getTileKey(tile Tile) string {
	return fmt.Sprintf("%d/%d/%d", tile.Zoom, tile.X, tile.Y)
}

func (tm *TileManager) GetTile(tile Tile) (image.Image, error) {
	key := getTileKey(tile)
	// log.Println("GetTile", key)

	if cached, exists := tm.cache.Get(key); exists {
		if img, ok := cached.(image.Image); ok {
			return img, nil
		}
	}

	img, err := tm.provider.GetTile(tile)
	if err != nil {
		return nil, err
	}

	tm.cache.Set(key, img)

	if tm.onLoad != nil {
		tm.onLoad()
	}
	return img, nil
}
