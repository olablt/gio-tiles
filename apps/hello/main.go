package main

import (
	"gio-maps/maps"
	"image"
	"log"
	"os"

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
	list         *widget.List
	size         image.Point
	visibleTiles []maps.Tile
}

func NewMapView() *MapView {
	return &MapView{
		tileManager: maps.NewTileManager(maps.NewLocalTileProvider()), // Use local provider
		center:      maps.LatLng{Lat: 51.507222, Lng: -0.1275},        // London
		zoom:        4,
		list: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
	}
}

func (mv *MapView) calculateVisibleTiles() {
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
			mv.visibleTiles = append(mv.visibleTiles, maps.Tile{
				X:    x,
				Y:    y,
				Zoom: mv.zoom,
			})
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
		mv.calculateVisibleTiles()
	}

	// Create operations stack
	ops := new(op.Ops)

	// Create a stack for all operations
	stack := op.Stack{}
	stack.Push(ops)
	defer stack.Pop()

	// Draw all visible tiles
	for _, tile := range mv.visibleTiles {
		img, err := mv.tileManager.GetTile(tile)
		if err != nil {
			log.Printf("Error loading tile %v: %v", tile, err)
			continue
		}

		// Calculate position for this tile relative to center
		centerTile := maps.LatLngToTile(mv.center, mv.zoom)
		offsetX := (tile.X - centerTile.X) * 256
		offsetY := (tile.Y - centerTile.Y) * 256

		// Create a transform stack for this tile
		tileStack := op.Stack{}
		tileStack.Push(ops)

		// Apply offset transform
		op.Offset(image.Point{X: offsetX, Y: offsetY}).Add(ops)

		// Draw the tile
		imageOp := paint.NewImageOp(img)
		imageOp.Add(ops)
		widget.Image{
			Src: imageOp,
			Fit: widget.Contain,
		}.Layout(gtx)

		tileStack.Pop()
	}

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
