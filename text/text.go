package text

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"strings"

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
	b, _ := font.BoundString(f.face, strings.Repeat("H", 64))
	r := image.Rect(b.Min.X.Floor(), b.Min.Y.Floor(), b.Max.X.Ceil(), b.Max.Y.Ceil())
	sz := r.Size()
	dst := image.NewNRGBA(image.Rect(0, 0, sz.X+2, sz.Y+2))
	d := font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(color.Opaque),
		Face: f.face,
		Dot:  fixed.Point26_6{X: -b.Min.X + 64, Y: -b.Min.Y + 64},
	}
	// d.DrawString(s)
	for i := 0; i < 64; i++ {
		d.DrawBytes([]byte{'H'})
		d.Dot.X = (d.Dot.X & ^(1<<6 - 1)) + fixed.Int26_6(i)
		log.Print(d.Dot.X)
	}
	of, _ := os.Create("os.png")
	png.Encode(of, dst)
	of.Close()
	return dst
}

// var fcache struct {
// 	m sync.Mutex
// 	fs map[]
// }

// func Font(f Facer, size float64, h font.Hinting)

// func DrawText(f Facer, s string, size float64, font.Hinting)
