package tiles

import (
	"image"
	"sync"
)

type CombinedTileProvider struct {
	primary    TileProvider
	fallback   TileProvider
	loading    map[string]bool
	loadingMu  sync.RWMutex
	onLoadFunc func()
}

func NewCombinedTileProvider(primary, fallback TileProvider) *CombinedTileProvider {
	return &CombinedTileProvider{
		primary:  primary,
		fallback: fallback,
		loading:  make(map[string]bool),
	}
}

func (p *CombinedTileProvider) SetOnLoadCallback(callback func()) {
	p.onLoadFunc = callback
}

func (p *CombinedTileProvider) GetTile(tile Tile) (image.Image, error) {
	key := GetTileKey(tile)

	// First try to get the fallback tile
	fallbackImg, err := p.fallback.GetTile(tile)
	if err != nil {
		return nil, err
	}

	// Check if we're already loading this tile
	p.loadingMu.RLock()
	isLoading := p.loading[key]
	p.loadingMu.RUnlock()

	if !isLoading {
		// Start loading the primary tile in a goroutine
		p.loadingMu.Lock()
		p.loading[key] = true
		p.loadingMu.Unlock()

		go func() {
			// Load primary tile asynchronously
			if _, err := p.primary.GetTile(tile); err == nil && p.onLoadFunc != nil {
				p.onLoadFunc() // Notify that a new tile is available
			}

			// Clear loading status
			p.loadingMu.Lock()
			delete(p.loading, key)
			p.loadingMu.Unlock()
		}()
	}

	// Return the fallback tile immediately
	return fallbackImg, nil
}
