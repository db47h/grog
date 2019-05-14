package text

import (
	"image"
	"image/color"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Hinting selects how to quantize a vector font's glyph nodes.
//
// Not all fonts support hinting.
//
// This is a convenience duplicate of golang.org/x/image/font#Hinting
//
type Hinting int

const (
	HintingNone Hinting = iota
	HintingVertical
	HintingFull
)

type Font struct {
	face font.Face
}

func NewFont(f font.Face) *Font {
	return &Font{face: f}
}

func TextImage(f *Font, s string) image.Image {
	b, _ := font.BoundString(f.face, s)
	sz := b.Max.Sub(b.Min)
	w, h := sz.X.Ceil()+2, sz.Y.Ceil()+2 // 2px bigger in order to prevent artifacts when zooming in
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	d := font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(color.White),
		Face: f.face,
		Dot:  fixed.Point26_6{X: -b.Min.X + 1<<6, Y: -b.Min.Y + 1<<6},
	}
	d.DrawString(s)
	return dst
}

// var fcache struct {
// 	m sync.Mutex
// 	fs map[]
// }

// func Font(f Facer, size float64, h font.Hinting)

// func DrawText(f Facer, s string, size float64, font.Hinting)
