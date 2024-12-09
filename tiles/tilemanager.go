package tiles

import (
	"context"
	"fmt"
	"image"
	_ "image/png"

	"gioui.org/op/paint"
	"github.com/olablt/gio-tiles/tiles/worker"
)

type TileProvider interface {
	GetTile(tile Tile) (image.Image, error)
}

type TileManager struct {
	cache    Cache
	provider TileProvider
	onLoad   func()
	pool     *worker.Pool
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewTileManager(provider TileProvider, cacheType CacheType) *TileManager {
	ctx, cancel := context.WithCancel(context.Background())

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
		pool:     worker.NewPool(4),
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (tm *TileManager) GetCache() Cache {
	return tm.cache
}

func (tm *TileManager) SetOnLoadCallback(callback func()) {
	tm.onLoad = callback
	if provider, ok := tm.provider.(*CombinedTileProvider); ok {
		provider.SetOnLoadCallback(callback)
	}
}

// getTileKey returns a unique string key for a tile
func GetTileKey(tile Tile) string {
	return fmt.Sprintf("%d/%d/%d", tile.Zoom, tile.X, tile.Y)
}

func (tm *TileManager) GetTile(tile Tile) (image.Image, error) {
    key := GetTileKey(tile)

    // First check if we already have the OSM tile cached
    if cached, exists := tm.cache.Get(key); exists {
        switch tm.cache.GetType() {
        case CacheImage:
            if img, ok := cached.(image.Image); ok {
                return img, nil
            }
        case CacheImageOp:
            if imgOp, ok := cached.(paint.ImageOp); ok {
                // We have the OSM tile cached
                return tm.provider.GetTile(tile)
            }
        }
    }

    // Start async loading of OSM tile if not already loading
    tm.pool.Submit(worker.Task{
        Ctx: tm.ctx,
        Work: func() error {
            img, err := tm.provider.GetTile(tile)
            if err != nil {
                return err
            }

            switch tm.cache.GetType() {
            case CacheImage:
                tm.cache.Set(key, img)
            case CacheImageOp:
                tm.cache.Set(key, paint.NewImageOp(img))
            }

            if tm.onLoad != nil {
                tm.onLoad()
            }
            return nil
        },
        Priority: tile.Zoom,
    })

    // Return local tile immediately while OSM loads
    if localProvider, ok := tm.provider.(*CombinedTileProvider); ok {
        return localProvider.fallback.GetTile(tile)
    }

    // Fallback if not using CombinedTileProvider
    return tm.provider.GetTile(tile)
}
