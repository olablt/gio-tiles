package maps

import (
    "fmt"
    "image"
    _ "image/png"
    "net/http"
    "sync"
)

type TileManager struct {
    cache map[string]image.Image
    mutex sync.RWMutex
}

func NewTileManager() *TileManager {
    return &TileManager{
        cache: make(map[string]image.Image),
    }
}

func (tm *TileManager) GetTile(tile Tile) (image.Image, error) {
    key := fmt.Sprintf("%d/%d/%d", tile.Zoom, tile.X, tile.Y)
    
    tm.mutex.RLock()
    if img, exists := tm.cache[key]; exists {
        tm.mutex.RUnlock()
        return img, nil
    }
    tm.mutex.RUnlock()

    // Download tile
    resp, err := http.Get(GetTileURL(tile))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    img, _, err := image.Decode(resp.Body)
    if err != nil {
        return nil, err
    }

    tm.mutex.Lock()
    tm.cache[key] = img
    tm.mutex.Unlock()

    return img, nil
}
