package texture

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/db47h/grog/gl"
)

// FilterMode selects how to filter textures.
//
type FilterMode int32

// FilterMode values map directly to their OpenGL equivalents.
//
const (
	Nearest              FilterMode = gl.GL_NEAREST
	Linear                          = gl.GL_LINEAR
	NearestMipmapNearest            = gl.GL_NEAREST_MIPMAP_NEAREST
	NearestMipmapLinear             = gl.GL_NEAREST_MIPMAP_LINEAR
	LinearMipmapNearest             = gl.GL_LINEAR_MIPMAP_NEAREST
	LinearMipmapLinear              = gl.GL_LINEAR_MIPMAP_LINEAR
)

// WrapMode selects how textures wrap when texture coordinates get outside of
// the range [0, 1].
//
// When used in conjunction with github.com/db47h/grog/batch, the only settings
// that make sense are ClampToEdge (the default) and ClampToBorder.
//
type WrapMode int32

// WrapMode values map directly to their OpenGL equivalents.
//
const (
	Repeat         WrapMode = gl.GL_REPEAT
	MirroredRepeat          = gl.GL_MIRRORED_REPEAT
	ClampToEdge             = gl.GL_CLAMP_TO_EDGE
	ClampToBorder           = gl.GL_CLAMP_TO_BORDER
)

type Texture struct {
	width  int
	height int
	glID   uint32
	mipmap bool
	dirty  bool
}

type config struct {
	wrapS, wrapT         WrapMode
	minFilter, magFilter FilterMode
	border               []float32
}

type ParameterFunc func(*config)

func Wrap(wrapS, wrapT WrapMode) ParameterFunc {
	return func(cfg *config) {
		cfg.wrapS = wrapS
		cfg.wrapT = wrapT
	}
}

func Filter(min, mag FilterMode) ParameterFunc {
	return func(cfg *config) {
		cfg.minFilter = min
		cfg.magFilter = mag
	}
}

func BorderColor(c color.Color) ParameterFunc {
	return func(cfg *config) {
		c := color.NRGBAModel.Convert(c).(color.NRGBA)
		cfg.border = []float32{float32(c.R) / 255, float32(c.G) / 255, float32(c.B) / 255, float32(c.A) / 255}
	}
}

func doMipmap(filter FilterMode) bool {
	switch filter {
	case NearestMipmapNearest, LinearMipmapLinear, LinearMipmapNearest, NearestMipmapLinear:
		return true
	default:
		return false
	}
}

func New(width, height int, params ...ParameterFunc) *Texture {
	return newTexture(width, height, gl.GL_RGBA, nil, params...)
}

func FromImage(src image.Image, params ...ParameterFunc) *Texture {
	var (
		pix    *uint8
		format int32
		sr     = src.Bounds()
		dr     = image.Rectangle{Max: sr.Size()}
	)
	switch i := src.(type) {
	case *image.NRGBA:
		pix = &i.Pix[0]
		format = gl.GL_RGBA
	default:
		dst := image.NewNRGBA(dr)
		draw.Draw(dst, dr, src, sr.Min, draw.Src)
		pix = &dst.Pix[0]
		format = gl.GL_RGBA
	}
	return newTexture(dr.Dx(), dr.Dy(), format, pix, params...)
}

func newTexture(width, height int, format int32, pix *uint8, params ...ParameterFunc) *Texture {
	var tex uint32
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.GL_TEXTURE_2D, tex)

	cfg := config{
		wrapS:     gl.GL_CLAMP_TO_EDGE,
		wrapT:     gl.GL_CLAMP_TO_EDGE,
		minFilter: gl.GL_LINEAR_MIPMAP_LINEAR,
		magFilter: gl.GL_LINEAR,
	}
	for _, f := range params {
		f(&cfg)
	}

	gl.TexParameteri(gl.GL_TEXTURE_2D, gl.GL_TEXTURE_WRAP_S, int32(cfg.wrapS))
	gl.TexParameteri(gl.GL_TEXTURE_2D, gl.GL_TEXTURE_WRAP_T, int32(cfg.wrapT))
	gl.TexParameteri(gl.GL_TEXTURE_2D, gl.GL_TEXTURE_MIN_FILTER, int32(cfg.minFilter))
	gl.TexParameteri(gl.GL_TEXTURE_2D, gl.GL_TEXTURE_MAG_FILTER, int32(cfg.magFilter))

	if cfg.wrapS == ClampToBorder || cfg.wrapT == ClampToBorder {
		c := cfg.border
		if c == nil {
			c = []float32{0, 0, 0, 0}
		}
		gl.TexParameterfv(gl.GL_TEXTURE_2D, gl.GL_TEXTURE_BORDER_COLOR, &c[0])
	}

	// TODO: this works with RGBA images, need to adjust if we handle more formats.
	gl.PixelStorei(gl.GL_UNPACK_ALIGNMENT, 4)
	gl.TexImage2D(gl.GL_TEXTURE_2D, 0, format, int32(width), int32(height), 0, uint32(format), gl.GL_UNSIGNED_BYTE, gl.Ptr(pix))

	t := &Texture{width: width, height: height, glID: tex}

	if doMipmap(cfg.minFilter) || doMipmap(cfg.magFilter) {
		if pix != nil {
			gl.GenerateMipmap(gl.GL_TEXTURE_2D)
		}
		t.mipmap = true
	}
	return t
}

func (t *Texture) OnBind() {
	if t.dirty {
		gl.GenerateMipmap(gl.GL_TEXTURE_2D)
		t.dirty = false
	}
}

// SetSubImage draws src to the texture. It works identically to draw.Draw with op set to draw.Src.
//
func (t *Texture) SetSubImage(dr image.Rectangle, src image.Image, sp image.Point) {
	var (
		pix    *uint8
		format uint32 = gl.GL_RGBA
		sz            = dr.Size()
		sr            = image.Rectangle{Min: sp, Max: sp.Add(sz)}
	)
	if sz.X == 0 || sz.Y == 0 {
		return
	}
	if i, ok := src.(*image.NRGBA); ok && sr == src.Bounds() {
		pix = &i.Pix[0]
	} else {
		r := image.Rectangle{Min: image.ZP, Max: sz}
		dst := image.NewNRGBA(r)
		draw.Draw(dst, r, src, sp, draw.Src)
		pix = &dst.Pix[0]
	}

	gl.BindTexture(gl.GL_TEXTURE_2D, t.glID)
	gl.PixelStorei(gl.GL_UNPACK_ALIGNMENT, 4)
	gl.TexSubImage2D(gl.GL_TEXTURE_2D, 0, int32(dr.Min.X), int32(dr.Min.Y), int32(sz.X), int32(sz.Y), format, gl.GL_UNSIGNED_BYTE, gl.Ptr(pix))
	if t.mipmap {
		t.dirty = true
	}
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

func (t *Texture) Delete() {
	gl.DeleteTextures(1, &t.glID)
}

func (t *Texture) Region(bounds image.Rectangle, origin image.Point) *Region {
	u0, v0 := t.GLCoords(bounds.Min)
	u1, v1 := t.GLCoords(bounds.Max)
	return &Region{
		Texture: t,
		origin:  image.Pt(origin.X, origin.Y),
		bounds:  bounds,
		uv:      [4]float32{u0, v1, u1, v0}, // flip texture vertically
	}
}

type Region struct {
	*Texture
	origin image.Point
	bounds image.Rectangle
	uv     [4]float32
}

func (r *Region) Origin() image.Point {
	return r.origin
}

func (r *Region) Size() image.Point {
	return r.bounds.Size()
}

func (r *Region) UV() [4]float32 {
	return r.uv
}

func (r *Region) Region(bounds image.Rectangle, origin image.Point) *Region {
	bounds = bounds.Add(r.bounds.Min)
	origin = origin.Add(r.bounds.Min)
	u0, v0 := r.GLCoords(bounds.Min)
	u1, v1 := r.GLCoords(bounds.Max)
	return &Region{
		Texture: r.Texture,
		origin:  image.Pt(origin.X, origin.Y),
		bounds:  bounds,
		uv:      [4]float32{u0, v1, u1, v0}, // flip texture vertically
	}
}
