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

// A Texture is a grog.Drawable that represents an OpenGL texture.
//
type Texture struct {
	width  int
	height int
	glID   uint32
	mipmap bool
	dirty  bool
}

type tp struct {
	wrapS, wrapT         WrapMode
	minFilter, magFilter FilterMode
	border               color.Color
}

// Parameter is implemented by functions setting texture parameters. See New.
//
type Parameter interface {
	set(*tp)
}

type optionFunc func(*tp)

func (f optionFunc) set(p *tp) {
	f(p)
}

// Wrap sets the GL_TEXTURE_WRAP_S and GL_TEXTURE_WRAP_T texture parameters.
//
func Wrap(wrapS, wrapT WrapMode) Parameter {
	return optionFunc(func(p *tp) {
		p.wrapS = wrapS
		p.wrapT = wrapT
	})
}

// Filter sets the GL_TEXTURE_MIN_FILTER and GL_TEXTURE_MAG_FILTER texture parameters.
//
func Filter(min, mag FilterMode) Parameter {
	return optionFunc(func(p *tp) {
		p.minFilter = min
		p.magFilter = mag
	})
}

// BorderColor sets the GL_TEXTURE_BORDER_COLOR texture parameter.
//
func BorderColor(c color.Color) Parameter {
	return optionFunc(func(p *tp) {
		p.border = c
	})
}

// New Returns a new uninitialized texture of the given width and height.
//
func New(width, height int, params ...Parameter) *Texture {
	return newTexture(width, height, gl.GL_RGBA, nil, params...)
}

// FromImage creates a new texture of the same dimensions as the source image.
// Regardless of the source image type, the resulting texture is always in RGBA
// format.
//
func FromImage(src image.Image, params ...Parameter) *Texture {
	var (
		pix    *uint8
		format int32
		sr     = src.Bounds()
		dr     = image.Rectangle{Max: sr.Size()}
	)
	switch i := src.(type) {
	case *image.RGBA:
		pix = &i.Pix[0]
		format = gl.GL_RGBA
	default:
		dst := image.NewRGBA(dr)
		draw.Draw(dst, dr, src, sr.Min, draw.Src)
		pix = &dst.Pix[0]
		format = gl.GL_RGBA
	}
	return newTexture(dr.Dx(), dr.Dy(), format, pix, params...)
}

func newTexture(width, height int, format int32, pix *uint8, params ...Parameter) *Texture {
	var tex uint32
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.GL_TEXTURE_2D, tex)

	t := &Texture{width: width, height: height, glID: tex}

	t.setParams(params...)

	// TODO: this works with RGBA images, need to adjust if we handle more formats.
	gl.PixelStorei(gl.GL_UNPACK_ALIGNMENT, 4)
	gl.TexImage2D(gl.GL_TEXTURE_2D, 0, format, int32(width), int32(height), 0, uint32(format), gl.GL_UNSIGNED_BYTE, gl.Ptr(pix))
	if t.dirty && pix != nil {
		gl.GenerateMipmap(gl.GL_TEXTURE_2D)
		t.dirty = false
	}
	return t
}

// Parameters sets the given texture parameters.
//
func (t *Texture) Parameters(params ...Parameter) {
	if len(params) == 0 {
		return
	}
	gl.BindTexture(gl.GL_TEXTURE_2D, t.glID)
	t.setParams(params...)
}

func (t *Texture) setParams(params ...Parameter) {
	var tp tp
	for _, p := range params {
		p.set(&tp)
	}
	if tp.wrapS != 0 {
		gl.TexParameteri(gl.GL_TEXTURE_2D, gl.GL_TEXTURE_WRAP_S, int32(tp.wrapS))
	}
	if tp.wrapT != 0 {
		gl.TexParameteri(gl.GL_TEXTURE_2D, gl.GL_TEXTURE_WRAP_T, int32(tp.wrapT))
	}
	if tp.minFilter != 0 {
		gl.TexParameteri(gl.GL_TEXTURE_2D, gl.GL_TEXTURE_MIN_FILTER, int32(tp.minFilter))
	}
	if tp.magFilter != 0 {
		gl.TexParameteri(gl.GL_TEXTURE_2D, gl.GL_TEXTURE_MAG_FILTER, int32(tp.magFilter))
	}
	if tp.border != nil {
		c := color.RGBAModel.Convert(tp.border).(color.RGBA)
		bc := [...]float32{float32(c.R) / 255, float32(c.G) / 255, float32(c.B) / 255, float32(c.A) / 255}
		gl.TexParameterfv(gl.GL_TEXTURE_2D, gl.GL_TEXTURE_BORDER_COLOR, &bc[0])
	}
	switch tp.minFilter {
	case NearestMipmapNearest, LinearMipmapLinear, LinearMipmapNearest, NearestMipmapLinear:
		t.mipmap = true
		t.dirty = true
	case Nearest, Linear:
		t.mipmap = false
		t.dirty = false
	}
}

// Bind binds the texture and regenerates mipmaps if needed.
//
func (t *Texture) Bind() {
	gl.BindTexture(gl.GL_TEXTURE_2D, t.glID)
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
	if i, ok := src.(*image.RGBA); ok && sr == src.Bounds() {
		pix = &i.Pix[0]
	} else {
		r := image.Rectangle{Min: image.ZP, Max: sz}
		dst := image.NewRGBA(r)
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

// GLCoords return the coordinates of the point pt mapped to the range [0, 1].
//
func (t *Texture) GLCoords(pt image.Point) (glX float32, glY float32) {
	return float32(pt.X) / float32(t.width),
		float32(pt.Y) / float32(t.height)
}

// Origin retruns the point of origin of the texture.
//
func (t *Texture) Origin() image.Point {
	return image.ZP
}

// Size returns the size of the texture.
//
func (t *Texture) Size() image.Point {
	return image.Point{t.width, t.height}
}

// UV returns the texture's UV coordinates in the range [0, 1]
//
func (t *Texture) UV() [4]float32 {
	return [4]float32{0, 1, 1, 0}
}

// NativeID returns the native identifier of the texture.
//
func (t *Texture) NativeID() uint32 {
	return uint32(t.glID)
}

// Delete deletes the texture.
//
func (t *Texture) Delete() {
	gl.DeleteTextures(1, &t.glID)
}

// Region returns a region within the texture.
//
func (t *Texture) Region(bounds image.Rectangle, origin image.Point) *Region {
	return &Region{
		Texture: t,
		origin:  image.Pt(origin.X, origin.Y),
		bounds:  bounds,
	}
}

// Region is a grog.Drawable that represents a sub-region in a Texture or
// another Region.
//
type Region struct {
	*Texture
	origin image.Point
	bounds image.Rectangle
}

// Origin retruns the point of origin of the region.
//
func (r *Region) Origin() image.Point {
	return r.origin
}

// Size retruns the size of the region.
//
func (r *Region) Size() image.Point {
	return r.bounds.Size()
}

// UV returns the regions's UV coordinates in the range [0, 1]
//
func (r *Region) UV() [4]float32 {
	u0, v0 := r.GLCoords(r.bounds.Min)
	u1, v1 := r.GLCoords(r.bounds.Max)
	return [4]float32{u0, v1, u1, v0}
}

// Region returns a sub-region within the Region.
//
func (r *Region) Region(bounds image.Rectangle, origin image.Point) *Region {
	return &Region{
		Texture: r.Texture,
		origin:  origin.Add(r.bounds.Min),
		bounds:  bounds.Add(r.bounds.Min),
	}
}
