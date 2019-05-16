package text

import (
	"image"
	"image/color"
	"unicode/utf8"

	"github.com/db47h/grog/batch"
	"github.com/db47h/grog/gl"
	"github.com/db47h/grog/texture"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

const (
	// see subPixels() in github.com/golang/freetype/truetype/face.go
	SubPixelsX    = 4
	subPixelBiasX = 8
	subPixelMaskX = -16
)

// Texture size for font glyph texture atlas.
//
var TextureSize int32 = 1024

var maxTextureSize int

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

type Font struct {
	face   font.Face
	glyphs []texture.Region
	cache  map[cacheKey]cacheValue
	ts     []*texture.Texture // current texture
	p      image.Point        // current point
	lh     int                // line height in current texture
}

type cacheKey struct {
	r  rune
	fx uint8
}

type cacheValue struct {
	index int // glyph index
	adv   fixed.Int26_6
}

func NewFont(f font.Face) *Font {
	return &Font{
		face:  f,
		cache: make(map[cacheKey]cacheValue),
	}
}

func (f *Font) Face() font.Face {
	return f.face
}

func (f *Font) DrawBytes(b *batch.Batch, x, y float32, s []byte, c color.Color) {
	var dotX = fixed.Int26_6(x * 64)
	x = float32(dotX.Floor())
	prev := rune(-1)
	for len(s) > 0 {
		r, sz := utf8.DecodeRune(s)
		s = s[sz:]
		if prev >= 0 {
			dotX += f.face.Kern(prev, r)
		}
		ix, glyph, advance := f.Glyph(fixed.Point26_6{X: dotX, Y: 0}, r)
		if glyph != nil {
			b.Draw(glyph, x+float32(ix), y, 1, 1, 0, c)
		}
		dotX += advance
		prev = r
	}
}

func textureSize() int {
	if maxTextureSize != 0 {
		return maxTextureSize
	}
	var tw int32
	gl.GetIntegerv(gl.GL_MAX_TEXTURE_SIZE, &tw)
	if tw > TextureSize || tw == 0 {
		tw = TextureSize
	}
	maxTextureSize = int(tw)
	return maxTextureSize
}

func (f *Font) currentTexture() *texture.Texture {
	l := len(f.ts)
	if l == 0 {
		return nil
	}
	return f.ts[l-1]
}

func (f *Font) Glyph(dot fixed.Point26_6, r rune) (x int, gr *texture.Region, advance fixed.Int26_6) {
	dx := (dot.X + subPixelBiasX) & subPixelMaskX
	ix, fx := int(dx>>6), dx&0x3f

	key := cacheKey{r, uint8(fx)}
	if v, ok := f.cache[key]; ok {
		if idx := v.index; idx >= 0 {
			return ix, &f.glyphs[idx], v.adv
		}
		return 0, nil, v.adv
	}

	dr, mask, maskp, advance, ok := f.face.Glyph(fixed.Point26_6{X: dot.X & 0x3f, Y: 0}, r)
	if !ok {
		return 0, nil, 0
	}
	sz := dr.Size()
	if sz.X == 0 || sz.Y == 0 {
		f.cache[key] = cacheValue{-1, advance}
		return 0, nil, advance
	}
	// adjust point of origin to account for rounding when quantizing subPixels
	org := image.Pt(-dr.Min.X+(ix-dot.X.Floor()), -dr.Min.Y)
	tr := dr.Add(image.Pt(-dr.Min.X+f.p.X, -dr.Min.Y+f.p.Y))
	t := f.currentTexture()
	if t != nil {
		sz := t.Size()
		if tr.Max.X > sz.X {
			f.p = image.Pt(1, f.p.Y+f.lh)
			tr = tr.Add(image.Pt(1-tr.Min.X, f.lh))
		}
		if tr.Max.Y > sz.Y {
			t = nil
		}
	}
	if t == nil {
		ts := textureSize()
		t = texture.FromImage(image.NewNRGBA(image.Rect(0, 0, ts, ts)), texture.Filter(gl.GL_LINEAR_MIPMAP_LINEAR, gl.GL_NEAREST))
		f.ts = append(f.ts, t)
		f.p = image.Pt(1, 1) // one pixel gap around glyphs
		tr = dr.Add(image.Pt(1-dr.Min.X, 1-dr.Min.Y))
		f.lh = 0
	}
	t.SetSubImage(tr, mask, maskp)
	f.p.X += tr.Dx() + 1
	if h := tr.Dy() + 1; h > f.lh {
		f.lh = h
	}
	index := len(f.glyphs)
	f.glyphs = append(f.glyphs, *t.Region(tr, org))
	f.cache[key] = cacheValue{index, advance}
	return ix, &f.glyphs[index], advance
}

func (f *Font) Close() error {
	for _, t := range f.ts {
		t.Delete()
	}
	return f.face.Close()
}
