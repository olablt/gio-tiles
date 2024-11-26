package maps

import (
	"math"
)

// Tile represents a map tile coordinates
type Tile struct {
	X, Y, Zoom int
}

// LatLng represents a geographical point
type LatLng struct {
	Lat, Lng float64
}

// LatLngToTile converts geographical coordinates to tile coordinates
func LatLngToTile(ll LatLng, zoom int) Tile {
	lat_rad := ll.Lat * math.Pi / 180
	n := math.Pow(2, float64(zoom))
	x := int((ll.Lng + 180.0) / 360.0 * n)
	y := int((1.0 - math.Log(math.Tan(lat_rad)+(1/math.Cos(lat_rad)))/math.Pi) / 2.0 * n)
	return Tile{X: x, Y: y, Zoom: zoom}
}

// TileToLatLng converts tile coordinates to geographical coordinates (returns center of tile)
func TileToLatLng(tile Tile) LatLng {
	n := math.Pow(2, float64(tile.Zoom))
	lon_deg := float64(tile.X)/n*360.0 - 180.0
	lat_rad := math.Atan(math.Sinh(math.Pi * (1 - 2*float64(tile.Y)/n)))
	lat_deg := lat_rad * 180.0 / math.Pi
	return LatLng{Lat: lat_deg, Lng: lon_deg}
}
