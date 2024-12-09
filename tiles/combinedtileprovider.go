package tiles

import (
	"fmt"
	"image"
	"sync"
)

type CombinedTileProvider struct {
	primary    TileProvider
	fallback   TileProvider
	loading    map[string]bool
	loadingMu  sync.RWMutex
	onLoadFunc func()
	cache      map[string]image.Image
	cacheMu    sync.RWMutex
}

func NewCombinedTileProvider(primary, fallback TileProvider) *CombinedTileProvider {
	return &CombinedTileProvider{
		primary:  primary,
		fallback: fallback,
		loading:  make(map[string]bool),
		cache:    make(map[string]image.Image),
	}
}

func (p *CombinedTileProvider) SetOnLoadCallback(callback func()) {
	p.onLoadFunc = callback
}

func (p *CombinedTileProvider) GetTile(tile Tile) (image.Image, error) {
	key := GetTileKey(tile)

	// Check if we already have the primary tile cached
	p.cacheMu.RLock()
	if cachedImg, exists := p.cache[key]; exists {
		p.cacheMu.RUnlock()
		return cachedImg, nil
	}
	p.cacheMu.RUnlock()

	// Try to get primary tile first
	primaryImg, err := p.primary.GetTile(tile)
	if err == nil {
		// Cache the successfully loaded primary tile
		p.cacheMu.Lock()
		p.cache[key] = primaryImg
		p.cacheMu.Unlock()
		return primaryImg, nil
	}

	// If primary fails, get fallback tile
	fallbackImg, err := p.fallback.GetTile(tile)
	if err != nil {
		return nil, fmt.Errorf("both primary and fallback providers failed: %v", err)
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
			if img, err := p.primary.GetTile(tile); err == nil {
				// Cache the successfully loaded primary tile
				p.cacheMu.Lock()
				p.cache[key] = img
				p.cacheMu.Unlock()

				// Notify that a new tile is available
				if p.onLoadFunc != nil {
					p.onLoadFunc()
				}
			}

			// Clear loading status
			p.loadingMu.Lock()
			delete(p.loading, key)
			p.loadingMu.Unlock()
		}()
	}

	// Return the fallback tile if we don't have the primary yet
	return fallbackImg, nil
}
