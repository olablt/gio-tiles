package main

import (
	"image"
	"log"
	"math"
	"os"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"github.com/olablt/gio-tiles/tiles"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
)

const (
	initialLatitude  = 51.507222 // London
	initialLongitude = -0.1275
	tileSize         = 256
)

type MapView struct {
	tileManager    *tiles.TileManager
	center         tiles.LatLng
	zoom           float64 // Changed to float64 for smooth zooming
	targetZoom     int     // The nearest integer zoom level for tile loading
	minZoom        int
	maxZoom        int
	list           *widget.List
	size           image.Point
	visibleTiles   []tiles.Tile
	metersPerPixel float64
	//
	clickPos    f32.Point
	dragging    bool
	lastDragPos f32.Point
	released    bool
	refresh     chan struct{}
}

func (mv *MapView) Update(gtx layout.Context) {
	tag := mv

	// process events
	dragDelta := f32.Point{}
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target:  tag,
			Kinds:   pointer.Scroll | pointer.Drag | pointer.Press | pointer.Release | pointer.Cancel,
			ScrollY: pointer.ScrollRange{Min: -10, Max: 10},
		})
		if !ok {
			break
		}

		if x, ok := ev.(pointer.Event); ok {
			// log
			// log.Println("pointer.Event", x)
			switch x.Kind {
			case pointer.Press:
				mv.clickPos = x.Position
				mv.dragging = true
			case pointer.Scroll:
				// Get mouse position relative to screen center
				screenCenterX := float64(mv.size.X >> 1)
				screenCenterY := float64(mv.size.Y >> 1)
				mouseOffsetX := float64(x.Position.X) - screenCenterX
				mouseOffsetY := float64(x.Position.Y) - screenCenterY

				// Convert screen coordinates to world coordinates at current zoom
				worldX, worldY := tiles.CalculateWorldCoordinates(mv.center, mv.zoom)
				mouseWorldX := worldX + mouseOffsetX
				mouseWorldY := worldY + mouseOffsetY

				// Update zoom level smoothly
				zoomDelta := float64(x.Scroll.Y) * -0.125 // Smaller increment for smoother zoom
				newZoom := mv.zoom + zoomDelta
				newZoom = math.Max(float64(mv.minZoom), math.Min(newZoom, float64(mv.maxZoom)))

				// If zoom changed, adjust center to keep mouse position fixed
				if newZoom != mv.zoom {
					// Calculate the new world coordinates after zoom
					zoomFactor := math.Pow(2, newZoom-mv.zoom)
					mv.zoom = newZoom
					mv.targetZoom = int(math.Round(mv.zoom))
					newWorldX := mouseWorldX * zoomFactor
					newWorldY := mouseWorldY * zoomFactor

					// Calculate where the new center should be
					newWorldCenterX := newWorldX - mouseOffsetX
					newWorldCenterY := newWorldY - mouseOffsetY

					// Convert back to geographical coordinates
					mv.center = tiles.WorldToLatLng(newWorldCenterX, newWorldCenterY, mv.zoom)

					mv.updateVisibleTiles()
				}
			case pointer.Drag:
				dragDelta = x.Position.Sub(mv.clickPos)
				log.Println("pointer.Drag", dragDelta)
			case pointer.Release:
				fallthrough
			case pointer.Cancel:
				mv.dragging = false
				mv.released = true
			}
		}
	}

	if mv.dragging {
		if mv.released {
			mv.lastDragPos = dragDelta
			mv.released = false
		}
		if dragDelta != mv.lastDragPos {
			// Calculate the delta from last position
			deltaX := dragDelta.X - mv.lastDragPos.X
			deltaY := dragDelta.Y - mv.lastDragPos.Y

			// Convert screen movement to geographical coordinates using cached metersPerPixel
			latChange := float64(deltaY) * mv.metersPerPixel / 111319.9
			lngChange := -float64(deltaX) * mv.metersPerPixel / (111319.9 * math.Cos(mv.center.Lat*math.Pi/180))

			mv.center.Lat += latChange
			mv.center.Lng += lngChange
			mv.updateVisibleTiles()
			mv.lastDragPos = dragDelta
		}
	}

	// Update size if changed
	if mv.size != gtx.Constraints.Max {
		mv.size = gtx.Constraints.Max
		mv.updateVisibleTiles()
	}
}

func (mv *MapView) Layout(gtx layout.Context) layout.Dimensions {
	mv.Update(gtx)

	// Calculate scale factor for current fractional zoom
	baseScale := math.Pow(2, mv.zoom-float64(mv.targetZoom))

	tag := mv

	// Confine the area of interest to a gtx Max
	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
	// mv.drag.Add(gtx.Ops)
	// Declare `tag` as being one of the targets.
	event.Op(gtx.Ops, tag)

	// Draw all visible tiles
	for _, tile := range mv.visibleTiles {
		var imageOp paint.ImageOp
		key := tiles.GetTileKey(tile)

		// Try to get from cache first
		if cached, ok := mv.tileManager.GetCache().Get(key); ok {
			if imgOp, ok := cached.(paint.ImageOp); ok {
				imageOp = imgOp
			}
		} else {
			// If not in cache, load and cache it
			img, err := mv.tileManager.GetTile(tile)
			if err != nil {
				log.Printf("Error loading tile %v: %v", tile, err)
				continue
			}
			imageOp = paint.NewImageOp(img)
			mv.tileManager.GetCache().Set(key, imageOp)
		}

		// Calculate positions
		centerWorldPx, centerWorldPy := tiles.CalculateWorldCoordinates(mv.center, mv.zoom)
		screenCenterX := mv.size.X >> 1
		screenCenterY := mv.size.Y >> 1
		tileWorldPx := float64(tile.X * tiles.TileSize)
		tileWorldPy := float64(tile.Y * tiles.TileSize)
		finalX := screenCenterX + int(tileWorldPx-centerWorldPx)
		finalY := screenCenterY + int(tileWorldPy-centerWorldPy)

		// Draw only if tile is visible
		if finalX+tiles.TileSize >= 0 && finalX <= mv.size.X &&
			finalY+tiles.TileSize >= 0 && finalY <= mv.size.Y {
			transformStack := op.Offset(image.Point{X: finalX, Y: finalY}).Push(gtx.Ops)
			scaleStack := op.Scale(f32.Point{X: float32(baseScale), Y: float32(baseScale)}).Push(gtx.Ops)
			imageOp.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			scaleStack.Pop()
			transformStack.Pop()
		}
	}

	return layout.Dimensions{Size: mv.size}
}

func NewMapView(refresh chan struct{}) *MapView {
	tm := tiles.NewTileManager(
		tiles.NewCombinedTileProvider(
			tiles.NewOSMTileProvider(),
			tiles.NewLocalTileProvider(),
		),
		tiles.CacheImageOp,
	)
	tm.SetOnLoadCallback(func() {
		// Non-blocking send to refresh channel
		select {
		case refresh <- struct{}{}:
		default:
		}
	})

	return &MapView{
		tileManager: tm,
		center:      tiles.LatLng{Lat: initialLatitude, Lng: initialLongitude}, // London
		zoom:        4.0,
		targetZoom:  4,
		minZoom:     0,
		maxZoom:     19,
		list: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
	}
}

func (mv *MapView) updateVisibleTiles() {
	mv.metersPerPixel = tiles.CalculateMetersPerPixel(mv.center.Lat, mv.targetZoom)
	mv.visibleTiles = tiles.CalculateVisibleTiles(mv.center, mv.targetZoom, mv.size)

	// Start loading tiles asynchronously
	for _, tile := range mv.visibleTiles {
		go mv.tileManager.GetTile(tile)
	}
}

func main() {
	refresh := make(chan struct{}, 1)
	mv := NewMapView(refresh)
	go func() {
		w := new(app.Window)

		var ops op.Ops
		go func() {
			for range refresh {
				w.Invalidate()
			}
		}()
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
