package text

import (
	"image"
	"image/color"
	"unicode/utf8"

	"github.com/db47h/grog"
	"github.com/db47h/grog/texture"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

const (
	// see subPixels() in https://github.com/golang/freetype/blob/master/truetype/face.go
	SubPixelsX = 8
	// 32 / SubPixelsX
	subPixelBiasX = 4
	// -64 / SubPixelsX
	subPixelMaskX = -8
	SubPixelsY    = 1
	subPixelBiasY = 32
	subPixelMaskY = -64
)

// Texture size for font glyph texture atlas.
// This value should be adjusted to be no larger than gl.GL_MAX_TEXTURE_SIZE:
//
//	var mts int32
//	gl.GetIntegerv(gl.GL_MAX_TEXTURE_SIZE, &mts)
//	if mts > int(TextureSize) || mts == 0 {
//		TextureSize = int(mts)
//	}
//
var TextureSize int = 1024

// func TextImage(f *Font, s string) image.Image {
// 	b, _ := font.BoundString(f.face, s)
// 	r := image.Rect(b.Min.X.Floor(), b.Min.Y.Floor(), b.Max.X.Ceil(), b.Max.Y.Ceil())
// 	sz := r.Size()
// 	dst := image.NewRGBA(image.Rect(0, 0, sz.X+2, sz.Y+2))
// 	d := font.Drawer{
// 		Dst:  dst,
// 		Src:  image.NewUniform(color.Opaque),
// 		Face: f.face,
// 		Dot:  fixed.Point26_6{X: (-b.Min.X + 64) & -64, Y: (-b.Min.Y + 64) & -64},
// 	}
// 	d.DrawString(s)
// 	return dst
// }

// Drawer draws text.
//
// A Drawer is not safe for concurrent use by multiple goroutines, since its Face is not.
//
type Drawer struct {
	face   font.Face
	glyphs []texture.Region
	cache  map[cacheKey]cacheValue
	ts     []*texture.Texture // current texture
	p      image.Point        // current point
	lh     int                // line height in current texture
	mf     texture.FilterMode
}

type cacheKey struct {
	r  rune
	fx uint8
	fy uint8
}

type cacheValue struct {
	index int // glyph index
	adv   fixed.Int26_6
}

// Hinting selects how to quantize a vector font's glyph nodes.
//
// Not all fonts support hinting.
//
// This is a convenience duplicate of golang.org/x/image/font#Hinting
//
type Hinting int

const (
	HintingNone     Hinting = Hinting(font.HintingNone)
	HintingVertical         = Hinting(font.HintingVertical)
	HintingFull             = Hinting(font.HintingFull)
)

// NewDrawer returns a new text Drawer using the given font face. The magFilter is
// the texture filter used when up-scaling.
//
func NewDrawer(f font.Face, magFilter texture.FilterMode) *Drawer {
	return &Drawer{
		face:  f,
		cache: make(map[cacheKey]cacheValue),
		mf:    magFilter,
	}
}

func (d *Drawer) Face() font.Face {
	return d.face
}

// DrawBytes uses the provided batch to draw s at coordinates x, y with the given color. It returns the advance.
//
// It is equivalent to DrawString(b, x, y, string(s), c) but may be more efficient.
//
func (d *Drawer) DrawBytes(batch grog.Drawer, s []byte, dp, scale grog.Point, c color.Color) (advance float32) {
	dot := fixed.Point26_6{X: fixed.Int26_6(dp.X * 64), Y: fixed.Int26_6(dp.Y * 64)}
	sp := dot.X
	prev := rune(-1)
	for len(s) > 0 {
		r, sz := utf8.DecodeRune(s)
		s = s[sz:]
		if prev >= 0 {
			dot.X += d.face.Kern(prev, r)
		}
		gp, glyph, advance := d.Glyph(dot, r)
		if glyph != nil {
			batch.Draw(glyph, grog.PtPt(gp), scale, 0, c)
		}
		dot.X += advance.Mul(fixed.Int26_6(scale.X * 64))
		prev = r
	}
	return float32(dot.X-sp) / 64
}

// DrawString uses the provided batch to draw s at coordinates x, y with the given color. It returns the advance.
//
func (d *Drawer) DrawString(batch grog.Drawer, s string, dp, scale grog.Point, c color.Color) (advance float32) {
	dot := fixed.Point26_6{X: fixed.Int26_6(dp.X * 64), Y: fixed.Int26_6(dp.Y * 64)}
	sp := dot.X
	prev := rune(-1)
	for _, r := range s {
		if prev >= 0 {
			dot.X += d.face.Kern(prev, r)
		}
		gp, glyph, advance := d.Glyph(dot, r)
		if glyph != nil {
			batch.Draw(glyph, grog.PtPt(gp), scale, 0, c)
		}
		dot.X += advance.Mul(fixed.Int26_6(scale.X * 64))
		prev = r
	}
	return float32(dot.X-sp) / 64
}

func (d *Drawer) currentTexture() *texture.Texture {
	l := len(d.ts)
	if l == 0 {
		return nil
	}
	return d.ts[l-1]
}

// Glyph returns the glyph texture Region for rune r drawn at dot, the draw
// point (for batch.Draw) as well as the advance.
//
func (d *Drawer) Glyph(dot fixed.Point26_6, r rune) (dp image.Point, gr *texture.Region, advance fixed.Int26_6) {
	dx, dy := (dot.X+subPixelBiasX)&subPixelMaskX, (dot.Y+subPixelBiasY)&subPixelMaskY
	ix, iy := int(dx>>6), int(dy>>6)

	key := cacheKey{r, uint8(dx & 0x3f), uint8(dy & 0x3f)}
	if v, ok := d.cache[key]; ok {
		if idx := v.index; idx >= 0 {
			return image.Point{X: ix, Y: iy}, &d.glyphs[idx], v.adv
		}
		return image.Point{}, nil, v.adv
	}

	dr, mask, maskp, advance, ok := d.face.Glyph(fixed.Point26_6{X: dot.X & 0x3f, Y: dot.Y & 0x3f}, r)
	if !ok {
		return image.Point{}, nil, 0
	}
	sz := dr.Size()
	if sz.X == 0 || sz.Y == 0 {
		// empty glyph
		d.cache[key] = cacheValue{-1, advance}
		return image.Point{}, nil, advance
	}
	// adjust point of origin to account for rounding when quantizing subPixels
	org := image.Pt(-dr.Min.X+(ix-dot.X.Floor()), -dr.Min.Y+(iy-dot.Y.Floor()))
	tr := dr.Add(image.Pt(-dr.Min.X+d.p.X, -dr.Min.Y+d.p.Y))
	t := d.currentTexture()
	if t != nil {
		sz := t.Size()
		if tr.Max.X > sz.X {
			d.p = image.Pt(1, d.p.Y+d.lh)
			tr = tr.Add(image.Pt(-tr.Min.X+d.p.X, d.lh))
		}
		if tr.Max.Y > sz.Y {
			t = nil
		}
	}
	if t == nil {
		t = texture.FromImage(image.NewRGBA(image.Rect(0, 0, TextureSize, TextureSize)),
			texture.Filter(texture.Linear, d.mf))
		d.ts = append(d.ts, t)
		d.p = image.Point{1, 1}
		tr = dr.Add(image.Pt(-dr.Min.X+d.p.X, -dr.Min.Y+d.p.Y))
		d.lh = 0
	}
	t.SetSubImage(tr, mask, maskp)
	d.p.X += tr.Dx() + 1
	if h := tr.Dy() + 1; h > d.lh {
		d.lh = h
	}
	index := len(d.glyphs)
	d.glyphs = append(d.glyphs, *t.Region(tr, org))
	d.cache[key] = cacheValue{index, advance}
	return image.Point{X: ix, Y: iy}, &d.glyphs[index], advance
}

// BoundBytes returns the draw point and pixel size of s, as well as the advance.
//
// It is equivalent to BoundString(string(s)) but may be more efficient.
//
func (d *Drawer) BoundBytes(s []byte) (dot image.Point, size image.Point, advance float32) {
	b, adv := font.BoundBytes(d.face, s)
	dot = image.Pt(-b.Min.X.Floor(), -b.Min.Y.Floor())
	return dot, image.Pt(b.Max.X.Ceil(), b.Max.Y.Ceil()).Sub(dot), float32(adv) / 64

}

// BoundString returns the draw point and pixel size of s, as well as the advance.
//
func (d *Drawer) BoundString(s string) (dot image.Point, size image.Point, advance float32) {
	b, adv := font.BoundString(d.face, s)
	dot = image.Pt(-b.Min.X.Floor(), -b.Min.Y.Floor())
	return dot, image.Pt(b.Max.X.Ceil(), b.Max.Y.Ceil()).Add(dot), float32(adv) / 64
}

// MeasureBytes returns how far dot would advance by drawing s.
//
func (d *Drawer) MeasureBytes(s []byte) (advance float32) {
	return float32(font.MeasureBytes(d.face, s)) / 64
}

// MeasureString returns how far dot would advance by drawing s.
//
func (d *Drawer) MeasureString(s string) (advance float32) {
	return float32(font.MeasureString(d.face, s)) / 64
}
