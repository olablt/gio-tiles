package maps

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

type LocalTileProvider struct{}

func NewLocalTileProvider() *LocalTileProvider {
	return &LocalTileProvider{}
}

func (p *LocalTileProvider) GetTile(tile Tile) (image.Image, error) {
	// Create a new 256x256 RGBA image (standard tile size)
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))

	// Fill with light blue background
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{200, 220, 255, 255}}, image.Point{}, draw.Src)

	// Create the text to draw
	text := fmt.Sprintf("%d/%d/%d", tile.Zoom, tile.X, tile.Y)

	// Draw text background
	textBgColor := color.RGBA{255, 255, 255, 220}
	padding := 10
	textWidth := basicfont.Face7x13.Metrics().Height.Round() * len(text) / 2
	textBgRect := image.Rect(
		128-textWidth/2-padding,
		120-padding,
		128+textWidth/2+padding,
		140+padding/2,
	)
	draw.Draw(img, textBgRect, &image.Uniform{textBgColor}, image.Point{}, draw.Over)

	// Set up the font drawer with black text
	point := fixed.Point26_6{
		X: fixed.I((256 - textWidth) / 2),
		Y: fixed.I(130), // Vertical position
	}
	
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.RGBA{0, 0, 0, 255}),
		Face: basicfont.Face7x13,
		Dot:  point,
	}

	d.DrawString(text)

	// Draw frame around tile
	borderColor := color.RGBA{100, 100, 100, 255}
	// Top border
	draw.Draw(img, image.Rect(0, 0, 256, 1), &image.Uniform{borderColor}, image.Point{}, draw.Src)
	// Bottom border
	draw.Draw(img, image.Rect(0, 255, 256, 256), &image.Uniform{borderColor}, image.Point{}, draw.Src)
	// Left border
	draw.Draw(img, image.Rect(0, 0, 1, 256), &image.Uniform{borderColor}, image.Point{}, draw.Src)
	// Right border
	draw.Draw(img, image.Rect(255, 0, 256, 256), &image.Uniform{borderColor}, image.Point{}, draw.Src)

	return img, nil
}
