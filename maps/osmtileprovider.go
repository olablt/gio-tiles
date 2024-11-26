package maps

import (
	"fmt"
	"image"
	_ "image/png"
	"net/http"
)

type OSMTileProvider struct{}

func NewOSMTileProvider() *OSMTileProvider {
	return &OSMTileProvider{}
}

func (p *OSMTileProvider) GetTile(tile Tile) (image.Image, error) {
	resp, err := http.Get(p.GetTileURL(tile))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	img, _, err := image.Decode(resp.Body)
	return img, err
}

// GetTileURL returns the URL for downloading the map tile
func (p *OSMTileProvider) GetTileURL(tile Tile) string {
	return fmt.Sprintf("https://tile.openstreetmap.org/%d/%d/%d.png",
		tile.Zoom, tile.X, tile.Y)
}
