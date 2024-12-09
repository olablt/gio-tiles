package mapview

import (
	"image"
	"log"
	"math"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
	"github.com/olablt/gio-tiles/tiles"
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
	prevZoom       int     // Previous integer zoom level for scaling old tiles
	minZoom        int
	maxZoom        int
	list           *widget.List
	size           image.Point
	visibleTiles   []tiles.Tile
	prevTiles      []tiles.Tile // Previous zoom level tiles
	metersPerPixel float64
	//
	clickPos       f32.Point
	dragging       bool
	lastDragPos    f32.Point
	released       bool
	refresh        chan struct{}
	currentCtx     context.Context
	cancelCurrent  context.CancelFunc
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
			switch x.Kind {
			case pointer.Press:
				mv.clickPos = x.Position
				mv.dragging = true
			case pointer.Scroll:
				log.Println("pointer.Scroll", x.Scroll.Y)
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
				zoomDelta := float64(x.Scroll.Y) * -0.015 // Smaller increment for smoother zoom
				newZoom := mv.zoom + zoomDelta
				newZoom = math.Max(float64(mv.minZoom), math.Min(newZoom, float64(mv.maxZoom)))

				// If zoom changed, adjust center to keep mouse position fixed
				if newZoom != mv.zoom {
					log.Println("newZoom", newZoom)
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
				x.Scroll = f32.Point{}

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

			// Adjust delta based on current zoom scale
			scale := math.Pow(2, mv.zoom-float64(mv.targetZoom))
			adjustedDeltaX := float64(deltaX) / scale
			adjustedDeltaY := float64(deltaY) / scale

			// Convert screen movement to geographical coordinates using cached metersPerPixel
			latChange := adjustedDeltaY * mv.metersPerPixel / 111319.9
			lngChange := -adjustedDeltaX * mv.metersPerPixel / (111319.9 * math.Cos(mv.center.Lat*math.Pi/180))

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
	event.Op(gtx.Ops, tag)

	// Draw previous zoom level tiles first if we're between zoom levels
	if math.Abs(mv.zoom-float64(mv.targetZoom)) > 0.01 && len(mv.prevTiles) > 0 {
		prevScale := math.Pow(2, mv.zoom-float64(mv.prevZoom))
		for _, tile := range mv.prevTiles {
			var imageOp paint.ImageOp
			key := tiles.GetTileKey(tile)

			if cached, ok := mv.tileManager.GetCache().Get(key); ok {
				if imgOp, ok := cached.(paint.ImageOp); ok {
					imageOp = imgOp

					// Calculate positions for previous zoom level tiles
					centerWorldPx, centerWorldPy := tiles.CalculateWorldCoordinates(mv.center, float64(mv.prevZoom))
					screenCenterX := mv.size.X >> 1
					screenCenterY := mv.size.Y >> 1
					tileWorldPx := float64(tile.X * tiles.TileSize)
					tileWorldPy := float64(tile.Y * tiles.TileSize)
					finalX := screenCenterX + int(tileWorldPx-centerWorldPx)
					finalY := screenCenterY + int(tileWorldPy-centerWorldPy)

					if finalX+tiles.TileSize >= 0 && finalX <= mv.size.X &&
						finalY+tiles.TileSize >= 0 && finalY <= mv.size.Y {
						transformStack := op.Offset(image.Point{X: finalX, Y: finalY}).Push(gtx.Ops)
						scaleStack := op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Point{X: float32(prevScale), Y: float32(prevScale)})).Push(gtx.Ops)
						imageOp.Add(gtx.Ops)
						paint.PaintOp{}.Add(gtx.Ops)
						scaleStack.Pop()
						transformStack.Pop()
					}
				}
			}
		}
	}

	// Draw current zoom level tiles
	baseScale = math.Pow(2, mv.zoom-float64(mv.targetZoom))
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

		// Calculate positions with fractional precision
		centerWorldPx, centerWorldPy := tiles.CalculateWorldCoordinates(mv.center, float64(mv.targetZoom))
		screenCenterX := float64(mv.size.X >> 1)
		screenCenterY := float64(mv.size.Y >> 1)
		tileWorldPx := float64(tile.X * tiles.TileSize)
		tileWorldPy := float64(tile.Y * tiles.TileSize)

		// Apply zoom scaling to the position difference
		scale := math.Pow(2, mv.zoom-float64(mv.targetZoom))
		finalX := int(screenCenterX + (tileWorldPx-centerWorldPx)*scale)
		finalY := int(screenCenterY + (tileWorldPy-centerWorldPy)*scale)

		// Draw only if tile is visible
		scaledTileSize := int(float64(tiles.TileSize) * baseScale)
		if finalX+scaledTileSize >= 0 && finalX <= mv.size.X &&
			finalY+scaledTileSize >= 0 && finalY <= mv.size.Y {
			transformStack := op.Offset(image.Point{X: finalX, Y: finalY}).Push(gtx.Ops)
			scaleStack := op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Point{X: float32(baseScale), Y: float32(baseScale)})).Push(gtx.Ops)
			imageOp.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			scaleStack.Pop()
			transformStack.Pop()
		}
	}

	return layout.Dimensions{Size: mv.size}
}

func New(refresh chan struct{}) *MapView {
	tm := tiles.NewTileManager(
		tiles.NewCombinedTileProvider(
			tiles.NewOSMTileProvider(),
			tiles.NewLocalTileProvider(),
		),
		tiles.CacheImageOp,
	)
	tm.SetOnLoadCallback(func() {
		log.Println("onLoad")
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
		prevZoom:    4,
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
	if mv.cancelCurrent != nil {
		mv.cancelCurrent()
	}
	mv.currentCtx, mv.cancelCurrent = context.WithCancel(context.Background())

	mv.metersPerPixel = tiles.CalculateMetersPerPixel(mv.center.Lat, mv.targetZoom)

	newTargetZoom := int(math.Round(mv.zoom))
	if newTargetZoom != mv.targetZoom {
		mv.prevZoom = mv.targetZoom
		mv.prevTiles = mv.visibleTiles
		mv.targetZoom = newTargetZoom
	}

	mv.visibleTiles = tiles.CalculateVisibleTiles(mv.center, mv.targetZoom, mv.size)

	ctx := mv.currentCtx
	for _, tile := range mv.visibleTiles {
		if ctx.Err() != nil {
			return
		}
		go mv.tileManager.GetTile(tile)
	}
}
