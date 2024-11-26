package main

import (
	"gio-maps/maps"
	"image"
	"io"
	"log"
	"math"
	"os"
	"strings"

	"gioui.org/f32"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/widget"
)

type MapView struct {
	tileManager  *maps.TileManager
	center       maps.LatLng
	zoom         int
	minZoom      int
	maxZoom      int
	list         *widget.List
	size         image.Point
	visibleTiles []maps.Tile
	drag         widget.Draggable
	lastDragPos  f32.Point
	released     bool
}

func NewMapView() *MapView {
	return &MapView{
		tileManager: maps.NewTileManager(maps.NewLocalTileProvider()), // Use local provider
		// tileManager: maps.NewTileManager(maps.NewOSMTileProvider()), // Use OSM provider
		center:  maps.LatLng{Lat: 51.507222, Lng: -0.1275}, // London
		zoom:    4,
		minZoom: 0,
		maxZoom: 19,
		list: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
	}
}

func (mv *MapView) constrainTile(tile maps.Tile) maps.Tile {
	maxTile := int(math.Pow(2, float64(tile.Zoom))) - 1
	tile.X = max(0, min(tile.X, maxTile))
	tile.Y = max(0, min(tile.Y, maxTile))
	return tile
}

func (mv *MapView) setZoom(newZoom int) {
	mv.zoom = max(mv.minZoom, min(newZoom, mv.maxZoom))
	mv.updateVisibleTiles()
}

func (mv *MapView) updateVisibleTiles() {
	// // Calculate bounding box in terms of tiles
	// topLeftTile := maps.LatLngToTile(mv.topLeft(), mv.zoom)
	// bottomRightTile := maps.LatLngToTile(mv.bottomRight(), mv.zoom)

	// mv.visibleTiles = make([]maps.Tile, 0)
	// for x := topLeftTile.X; x <= bottomRightTile.X; x++ {
	// 	for y := topLeftTile.Y; y <= bottomRightTile.Y; y++ {
	// 		mv.visibleTiles = append(mv.visibleTiles, maps.Tile{
	// 			X:    x,
	// 			Y:    y,
	// 			Zoom: mv.zoom,
	// 		})
	// 	}
	// }

	// // Asynchronously load tiles
	// for _, tile := range mv.visibleTiles {
	// 	go mv.tileManager.GetTile(tile)
	// }

	// Calculate center tile
	centerTile := maps.LatLngToTile(mv.center, mv.zoom)

	// Calculate how many tiles we need in each direction based on window size
	tilesX := (mv.size.X / 256) + 2 // Add buffer tiles
	tilesY := (mv.size.Y / 256) + 2

	startX := centerTile.X - tilesX/2
	startY := centerTile.Y - tilesY/2

	mv.visibleTiles = make([]maps.Tile, 0, tilesX*tilesY)

	for x := startX; x < startX+tilesX; x++ {
		for y := startY; y < startY+tilesY; y++ {
			tile := mv.constrainTile(maps.Tile{
				X:    x,
				Y:    y,
				Zoom: mv.zoom,
			})
			mv.visibleTiles = append(mv.visibleTiles, tile)
		}
	}

	// Start loading tiles asynchronously
	for _, tile := range mv.visibleTiles {
		go mv.tileManager.GetTile(tile)
	}
}

func (mv *MapView) Layout(gtx layout.Context) layout.Dimensions {
	// Update size if changed
	if mv.size != gtx.Constraints.Max {
		mv.size = gtx.Constraints.Max
		mv.updateVisibleTiles()
	}

	// Handle drag events
	if mv.drag.Dragging() {
		pos := mv.drag.Pos()
		if mv.released {
			mv.lastDragPos = pos
			mv.released = false
		}
		if pos != mv.lastDragPos {
			// Calculate the delta from last position
			deltaX := pos.X - mv.lastDragPos.X
			deltaY := pos.Y - mv.lastDragPos.Y

			// Convert screen movement to geographical coordinates
			metersPerPixel := 156543.03392 * math.Cos(mv.center.Lat*math.Pi/180) / math.Pow(2, float64(mv.zoom))
			latChange := float64(deltaY) * metersPerPixel / 111319.9
			lngChange := -float64(deltaX) * metersPerPixel / (111319.9 * math.Cos(mv.center.Lat*math.Pi/180))

			mv.center.Lat += latChange
			mv.center.Lng += lngChange
			mv.updateVisibleTiles()
			mv.lastDragPos = pos
		}
	} else {
		mv.released = true
	}

	ops := gtx.Ops

	// Draw all visible tiles
	for _, tile := range mv.visibleTiles {
		img, err := mv.tileManager.GetTile(tile)
		if err != nil {
			log.Printf("Error loading tile %v: %v", tile, err)
			continue
		}

		// Calculate position for this tile relative to center
		centerTile := maps.LatLngToTile(mv.center, mv.zoom)
		n := math.Pow(2, float64(mv.zoom))

		// Get precise pixel coordinates for center
		centerPxX := float64(centerTile.X)*256 + (mv.center.Lng+180.0)/360.0*n*256
		centerPxY := float64(centerTile.Y)*256 + (1.0-math.Log(math.Tan(mv.center.Lat*math.Pi/180.0)+(1/math.Cos(mv.center.Lat*math.Pi/180.0)))/math.Pi)/2.0*n*256

		// Calculate screen position
		screenCenterX := mv.size.X / 2
		screenCenterY := mv.size.Y / 2

		// Position relative to window center
		finalX := screenCenterX + int(float64(tile.X*256)-centerPxX)
		finalY := screenCenterY + int(float64(tile.Y*256)-centerPxY)

		// Create transform stack and apply offset
		transform := op.Offset(image.Point{X: finalX, Y: finalY}).Push(ops)

		// Draw the tile
		imageOp := paint.NewImageOp(img)
		imageOp.Add(ops)
		paint.PaintOp{}.Add(ops)

		transform.Pop()
	}

	w := func(gtx layout.Context) layout.Dimensions {
		// sz := image.Pt(10, 10) // drag area
		sz := gtx.Constraints.Max
		return layout.Dimensions{Size: sz}
	}
	mv.drag.Layout(gtx, w, w)
	// drag must respond with an Offer event when requested.
	// Use the drag method for this.
	if m, ok := mv.drag.Update(gtx); ok {
		mv.drag.Offer(gtx, m, io.NopCloser(strings.NewReader("hello world")))
	}
	// mv.drag.Layout(gtx, func(gtx layout.Context, index int) layout.Dimensions {
	// 	return layout.Dimensions{Size: image.Point{X: 256, Y: 256}}
	// })

	return layout.Dimensions{Size: mv.size}
}

func main() {
	go func() {
		w := new(app.Window)

		mv := NewMapView()

		var ops op.Ops
		for {
			switch e := w.Event().(type) {
			case app.DestroyEvent:
				os.Exit(0)
			case app.FrameEvent:
				gtx := app.NewContext(&ops, e)
				mv.Layout(gtx)
				e.Frame(gtx.Ops)
			}
		}
	}()
	app.Main()
}
