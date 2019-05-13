package texture

import (
	"image"
	"image/draw"

	"github.com/db47h/grog"
	"github.com/db47h/grog/gl"
)

type Texture struct {
	width  int
	height int
	glID   uint32
}

type config struct {
	wrapS, wrapT         int32
	minFilter, magFilter int32
}

type ParameterFunc func(*config)

func Wrap(wrapS, wrapT int32) ParameterFunc {
	return func(cfg *config) {
		cfg.wrapS = wrapS
		cfg.wrapT = wrapT
	}
}

func Filter(min, mag int32) ParameterFunc {
	return func(cfg *config) {
		cfg.minFilter = min
		cfg.magFilter = mag
	}
}

func doMipmap(filter int32) bool {
	switch filter {
	case gl.GL_NEAREST_MIPMAP_NEAREST, gl.GL_NEAREST_MIPMAP_LINEAR, gl.GL_LINEAR_MIPMAP_NEAREST, gl.GL_LINEAR_MIPMAP_LINEAR:
		return true
	default:
		return false
	}
}

func New(src image.Image, params ...ParameterFunc) *Texture {
	var (
		pix    []uint8
		format int32
		sr     = src.Bounds()
		dr     = image.Rectangle{Max: sr.Size()}
	)
	switch i := src.(type) {
	case *image.NRGBA:
		pix = i.Pix
		format = gl.GL_RGBA
	default:
		dst := image.NewNRGBA(dr)
		draw.Draw(dst, dr, src, sr.Min, draw.Src)
		pix = dst.Pix
		format = gl.GL_RGBA
	}

	var tex uint32
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.GL_TEXTURE_2D, tex)

	cfg := config{
		wrapS:     gl.GL_REPEAT,
		wrapT:     gl.GL_REPEAT,
		minFilter: gl.GL_LINEAR_MIPMAP_LINEAR,
		magFilter: gl.GL_LINEAR,
	}
	for _, f := range params {
		f(&cfg)
	}

	gl.TexParameteri(gl.GL_TEXTURE_2D, gl.GL_TEXTURE_WRAP_S, cfg.wrapS)
	gl.TexParameteri(gl.GL_TEXTURE_2D, gl.GL_TEXTURE_WRAP_T, cfg.wrapT)
	gl.TexParameteri(gl.GL_TEXTURE_2D, gl.GL_TEXTURE_MIN_FILTER, cfg.minFilter)
	gl.TexParameteri(gl.GL_TEXTURE_2D, gl.GL_TEXTURE_MAG_FILTER, cfg.magFilter)

	gl.TexImage2D(gl.GL_TEXTURE_2D, 0, format, int32(dr.Dx()), int32(dr.Dy()), 0, uint32(format), gl.GL_UNSIGNED_BYTE, gl.Ptr(&pix[0]))
	if doMipmap(cfg.minFilter) || doMipmap(cfg.magFilter) {
		gl.GenerateMipmap(gl.GL_TEXTURE_2D)
	}

	return &Texture{dr.Dx(), dr.Dy(), tex}
}

func (t *Texture) GLCoords(pt image.Point) (glX float32, glY float32) {
	return float32(pt.X) / float32(t.width),
		float32(pt.Y) / float32(t.height)
}

func (t *Texture) Origin() image.Point {
	return image.ZP
}

func (t *Texture) Size() image.Point {
	return image.Point{t.width, t.height}
}

func (t *Texture) UV() [4]float32 {
	return [4]float32{0, 1, 1, 0}
}

func (t *Texture) NativeID() uint32 {
	return uint32(t.glID)
}

func (t *Texture) Region(bounds image.Rectangle, origin image.Point) grog.Drawable {
	u0, v0 := t.GLCoords(bounds.Min)
	u1, v1 := t.GLCoords(bounds.Max)
	return &region{
		Texture: t,
		origin:  image.Pt(origin.X, origin.Y),
		bounds:  bounds,
		uv:      [4]float32{u0, v1, u1, v0}, // flip texture vertically
	}
}

type region struct {
	*Texture
	origin image.Point
	bounds image.Rectangle
	uv     [4]float32
}

func (r *region) Origin() image.Point {
	return r.origin
}

func (r *region) Size() image.Point {
	return r.bounds.Size()
}

func (r *region) UV() [4]float32 {
	return r.uv
}

func (r *region) Region(bounds image.Rectangle, origin image.Point) grog.Drawable {
	bounds = bounds.Add(r.bounds.Min)
	origin = origin.Add(r.bounds.Min)
	u0, v0 := r.GLCoords(bounds.Min)
	u1, v1 := r.GLCoords(bounds.Max)
	return &region{
		Texture: r.Texture,
		origin:  image.Pt(origin.X, origin.Y),
		bounds:  bounds,
		uv:      [4]float32{u0, v1, u1, v0}, // flip texture vertically
	}
}
