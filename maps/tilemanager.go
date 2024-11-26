package maps

import (
	"fmt"
	"image"
	_ "image/png"
	"sync"
)

type TileProvider interface {
	GetTile(tile Tile) (image.Image, error)
}

type TileManager struct {
	cache    map[string]image.Image
	mutex    sync.RWMutex
	provider TileProvider
}

func NewTileManager(provider TileProvider) *TileManager {
	return &TileManager{
		cache:    make(map[string]image.Image),
		provider: provider,
	}
}

func (tm *TileManager) GetTile(tile Tile) (image.Image, error) {
	key := fmt.Sprintf("%d/%d/%d", tile.Zoom, tile.X, tile.Y)
	// log.Println("GetTile", key)

	tm.mutex.RLock()
	if img, exists := tm.cache[key]; exists {
		tm.mutex.RUnlock()
		return img, nil
	}
	tm.mutex.RUnlock()

	img, err := tm.provider.GetTile(tile)
	if err != nil {
		return nil, err
	}

	tm.mutex.Lock()
	tm.cache[key] = img
	tm.mutex.Unlock()

	return img, nil
}
