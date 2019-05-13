package text

import (
	"image"
	"image/color"
	"io/ioutil"
	"log"

	"github.com/db47h/ofs"
	"github.com/golang/freetype/truetype"
	"github.com/pkg/errors"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

const dpi = 72.0

type Font interface {
	NewFace(size float64, hinting font.Hinting) (font.Face, error)
}

type fnt truetype.Font

func (f *fnt) NewFace(size float64, hinting font.Hinting) (font.Face, error) {
	return truetype.NewFace((*truetype.Font)(f), &truetype.Options{Size: size, DPI: dpi, Hinting: hinting}), nil
}

func LoadFont(ovl *ofs.Overlay, name string) (Font, error) {
	f, err := ovl.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read font %s", name)
	}
	tf, err := truetype.Parse(data)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse font %s", name)
	}
	return (*fnt)(tf), err
}

func TextImage(f font.Face, s string) image.Image {
	m := f.Metrics()
	b, _ := font.BoundString(f, s)
	sz := b.Max.Sub(b.Min)
	w, h := sz.X.Ceil()+2, sz.Y.Ceil()+2 // 2px bigger in order to prevent artifacts when zooming in
	log.Printf("%#v", m)
	log.Print(w, h)
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
