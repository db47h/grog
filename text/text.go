package text

import (
	"image"
	"image/color"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

type Facer interface {
	NewFace(size float64, hinting font.Hinting) (font.Face, error)
}

func TextImage(f font.Face, s string) image.Image {
	b, _ := font.BoundString(f, s)
	sz := b.Max.Sub(b.Min)
	w, h := sz.X.Ceil()+2, sz.Y.Ceil()+2 // 2px bigger in order to prevent artifacts when zooming in
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	d := font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(color.White),
		Face: f,
		Dot:  fixed.Point26_6{X: -b.Min.X + 1<<6, Y: -b.Min.Y + 1<<6},
	}
	d.DrawString(s)
	return dst
}

//func DrawText(fnt *truetype.Font, s string, size float64)
