package tiles

import (
	"context"
	"fmt"
	"image"
	_ "image/png"
	"log"
	"net/http"
	"sync"
	"time"
)

type OSMTileProvider struct {
	progressMutex sync.Mutex
	progress      map[string]int
	client        *http.Client
}

func NewOSMTileProvider() *OSMTileProvider {
	return &OSMTileProvider{
		progressMutex: sync.Mutex{},
		progress:      map[string]int{},
		client:        &http.Client{},
	}
}

func (p *OSMTileProvider) GetTile(tile Tile) (image.Image, error) {
	url := p.GetTileURL(tile)

	p.progressMutex.Lock()
	if _, downloadInProgress := p.progress[url]; downloadInProgress {
		p.progressMutex.Unlock()
		log.Printf("OSM: Requested tile with existing download in progress for: %s", url)
		return nil, fmt.Errorf("OSM: Requested tile with existing download in progress for: %s", url)
	}

	log.Printf("OSM: Requesting tile z=%d x=%d y=%d from %s", tile.Zoom, tile.X, tile.Y, url)
	p.progress[url] = 0
	p.progressMutex.Unlock()

	// Create request with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer func() {
		p.progressMutex.Lock()
		delete(p.progress, url)
		p.progressMutex.Unlock()
		cancel()
	}()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Printf("Error creating request for tile %v: %v", tile, err)
		return nil, err
	}

	// Add browser-like headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/115.0")
	req.Header.Set("Accept", "image/webp,*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", "https://www.openstreetmap.org/")

	resp, err := p.client.Do(req)
	if err != nil {
		log.Printf("Error fetching tile %v: %v", tile, err)
		return nil, err
	}
	defer resp.Body.Close()

	// log.Printf("OSM tile response status: %s", resp.Status)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		log.Printf("Error decoding tile image %v: %v", tile, err)
		return nil, err
	}

	log.Printf("OSM: Successfully loaded tile z=%d x=%d y=%d", tile.Zoom, tile.X, tile.Y)
	return img, nil
}

// GetTileURL returns the URL for downloading the map tile
func (p *OSMTileProvider) GetTileURL(tile Tile) string {
	return fmt.Sprintf("https://tile.openstreetmap.org/%d/%d/%d.png",
		tile.Zoom, tile.X, tile.Y)
}
