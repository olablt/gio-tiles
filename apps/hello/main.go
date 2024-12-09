package main

import (
	"os"

	"gioui.org/app"
	"gioui.org/op"
	"github.com/olablt/gio-tiles/mapview"
)

func main() {
	refresh := make(chan struct{}, 1)
	mv := mapview.New(refresh)
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
