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
	// see subPixels() in github.com/golang/freetype/truetype/face.go
	SubPixelsX    = 8
	subPixelBiasX = 4
	subPixelMaskX = -8
	SubPixelsY    = 8
	subPixelBiasY = 4
	subPixelMaskY = -8
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
// 	dst := image.NewNRGBA(image.Rect(0, 0, sz.X+2, sz.Y+2))
// 	d := font.Drawer{
// 		Dst:  dst,
// 		Src:  image.NewUniform(color.Opaque),
// 		Face: f.face,
// 		Dot:  fixed.Point26_6{X: (-b.Min.X + 64) & -64, Y: (-b.Min.Y + 64) & -64},
// 	}
// 	d.DrawString(s)
// 	return dst
// }

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

func (d *Drawer) DrawBytes(b grog.Drawer, x, y float32, s []byte, c color.Color) (advance float32) {
	dot := fixed.Point26_6{X: fixed.Int26_6(x * 64), Y: fixed.Int26_6(y * 64)}
	sp := dot.X
	prev := rune(-1)
	for len(s) > 0 {
		r, sz := utf8.DecodeRune(s)
		s = s[sz:]
		if prev >= 0 {
			dot.X += d.face.Kern(prev, r)
		}
		dp, glyph, advance := d.Glyph(dot, r)
		if glyph != nil {
			b.Draw(glyph, float32(dp.X), float32(dp.Y), 1, 1, 0, c)
		}
		dot.X += advance
		prev = r
	}
	return float32(dot.X-sp) / 64
}

func (d *Drawer) DrawString(b grog.Drawer, x, y float32, s string, c color.Color) (advance float32) {
	dot := fixed.Point26_6{X: fixed.Int26_6(x * 64), Y: fixed.Int26_6(y * 64)}
	sp := dot.X
	prev := rune(-1)
	for _, r := range s {
		if prev >= 0 {
			dot.X += d.face.Kern(prev, r)
		}
		dp, glyph, advance := d.Glyph(dot, r)
		if glyph != nil {
			b.Draw(glyph, float32(dp.X), float32(dp.Y), 1, 1, 0, c)
		}
		dot.X += advance
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
			d.p = image.Pt(0, d.p.Y+d.lh)
			tr = tr.Add(image.Pt(-tr.Min.X, d.lh))
		}
		if tr.Max.Y > sz.Y {
			t = nil
		}
	}
	if t == nil {
		t = texture.FromImage(image.NewNRGBA(image.Rect(0, 0, TextureSize, TextureSize)),
			texture.Wrap(texture.ClampToBorder, texture.ClampToBorder),
			texture.Filter(texture.LinearMipmapLinear, d.mf))
		d.ts = append(d.ts, t)
		d.p = image.Point{}
		tr = dr.Add(image.Pt(-dr.Min.X, -dr.Min.Y))
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

func (d *Drawer) Close() error {
	for _, t := range d.ts {
		t.Delete()
	}
	return d.face.Close()
}

// BoundBytes returns the bounding box of s with f, drawn at a dot equal to the origin, as well as the advance.
//
// It is equivalent to BoundString(string(s)) but may be more efficient.
//
func (d *Drawer) BoundBytes(s []byte) (bounds fixed.Rectangle26_6, advance fixed.Int26_6) {
	return font.BoundBytes(d.face, s)
}

// BoundString returns the bounding box of s with f, drawn at a dot equal to the origin, as well as the advance.
//
func (d *Drawer) BoundString(s string) (bounds fixed.Rectangle26_6, advance fixed.Int26_6) {
	return font.BoundString(d.face, s)
}

// MeasureBytes returns how far dot would advance by drawing s.
//
func (d *Drawer) MeasureBytes(s []byte) (advance fixed.Int26_6) {
	return font.MeasureBytes(d.face, s)
}

// MeasureString returns how far dot would advance by drawing s.
//
func (d *Drawer) MeasureString(s string) (advance fixed.Int26_6) {
	return font.MeasureString(d.face, s)
}
