package main

import (
    "log"
    "os"
    "image"
    "gioui.org/app"
    "gioui.org/io/system"
    "gioui.org/layout"
    "gioui.org/op"
    "gioui.org/op/paint"
    "gioui.org/unit"
    "gioui.org/widget"
    "./maps"
)

type MapView struct {
    tileManager *maps.TileManager
    center      maps.LatLng
    zoom        int
    list        *widget.List
}

func NewMapView() *MapView {
    return &MapView{
        tileManager: maps.NewTileManager(),
        center:      maps.LatLng{Lat: 51.507222, Lng: -0.1275}, // London
        zoom:        12,
        list: &widget.List{
            List: layout.List{
                Axis: layout.Vertical,
            },
        },
    }
}

func (mv *MapView) Layout(gtx layout.Context) layout.Dimensions {
    // Calculate center tile
    centerTile := maps.LatLngToTile(mv.center, mv.zoom)
    
    // Create operations stack
    ops := new(op.Ops)
    
    return layout.Flex{}.Layout(gtx,
        layout.Rigid(func(gtx layout.Context) layout.Dimensions {
            // Draw visible tiles
            tile, err := mv.tileManager.GetTile(centerTile)
            if err != nil {
                log.Printf("Error loading tile: %v", err)
                return layout.Dimensions{}
            }

            imageOp := paint.NewImageOp(tile)
            imageOp.Add(ops)
            
            return widget.Image{
                Src: imageOp,
                Fit: widget.Contain,
            }.Layout(gtx)
        }),
    )
}

func main() {
    go func() {
        w := app.NewWindow(
            app.Title("Map Viewer"),
            app.Size(unit.Dp(800), unit.Dp(600)),
        )

        mv := NewMapView()
        
        var ops op.Ops
        for e := range w.Events() {
            switch e := e.(type) {
            case system.DestroyEvent:
                os.Exit(0)
            case system.FrameEvent:
                gtx := layout.NewContext(&ops, e)
                mv.Layout(gtx)
                e.Frame(gtx.Ops)
            }
        }
    }()
    app.Main()
}
