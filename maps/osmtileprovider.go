package maps

import (
	"fmt"
	"image"
	_ "image/png"
	"log"
	"net/http"
)

type OSMTileProvider struct{}

func NewOSMTileProvider() *OSMTileProvider {
	return &OSMTileProvider{}
}

func (p *OSMTileProvider) GetTile(tile Tile) (image.Image, error) {
	url := p.GetTileURL(tile)
	log.Printf("Requesting OSM tile: %s", url)

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error fetching tile %v: %v", tile, err)
		return nil, err
	}
	defer resp.Body.Close()

	log.Printf("OSM tile response status: %s", resp.Status)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		log.Printf("Error decoding tile image %v: %v", tile, err)
		return nil, err
	}
	
	log.Printf("Successfully loaded OSM tile: %v", tile)
	return img, nil
}

// GetTileURL returns the URL for downloading the map tile
func (p *OSMTileProvider) GetTileURL(tile Tile) string {
	return fmt.Sprintf("https://tile.openstreetmap.org/%d/%d/%d.png",
		tile.Zoom, tile.X, tile.Y)
}
