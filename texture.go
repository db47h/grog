package grog

import (
	"image"
	"image/draw"

	"github.com/db47h/grog/gl"
)

// TextureFilter selects how to filter textures when minifying or magnifying.
//
type TextureFilter int32

// FilterMode values map directly to their OpenGL equivalents.
//
const (
	Nearest TextureFilter = gl.GL_NEAREST
	Linear                = gl.GL_LINEAR
)

// TextureWrap selects how textures wrap when texture coordinates get outside of
// the range [0, 1].
//
// When used in conjunction with github.com/db47h/grog/batch, the only settings
// that make sense are ClampToEdge (the default) and ClampToBorder.
//
type TextureWrap int32

// WrapMode values map directly to their OpenGL equivalents.
//
const (
	Repeat         TextureWrap = gl.GL_REPEAT
	MirroredRepeat             = gl.GL_MIRRORED_REPEAT
	ClampToEdge                = gl.GL_CLAMP_TO_EDGE
)

// A Texture is a Drawable that represents an OpenGL texture.
//
type Texture struct {
	width  int
	height int
	glID   uint32
}

type tp struct {
	wrapS, wrapT         TextureWrap
	minFilter, magFilter TextureFilter
}

// TextureParameter is implemented by functions setting texture parameters. See New.
//
type TextureParameter interface {
	set(*tp)
}

type textureOptionFunc func(*tp)

func (f textureOptionFunc) set(p *tp) {
	f(p)
}

// Wrap sets the GL_TEXTURE_WRAP_S and GL_TEXTURE_WRAP_T texture parameters.
//
func Wrap(wrapS, wrapT TextureWrap) TextureParameter {
	return textureOptionFunc(func(p *tp) {
		p.wrapS = wrapS
		p.wrapT = wrapT
	})
}

// Filter sets the GL_TEXTURE_MIN_FILTER and GL_TEXTURE_MAG_FILTER texture parameters.
//
func Filter(min, mag TextureFilter) TextureParameter {
	return textureOptionFunc(func(p *tp) {
		p.minFilter = min
		p.magFilter = mag
	})
}

// NewTexture Returns a new uninitialized texture of the given width and height.
//
func NewTexture(width, height int, params ...TextureParameter) *Texture {
	return newTexture(width, height, gl.GL_RGBA, nil, params...)
}

// TextureFromImage creates a new texture of the same dimensions as the source image.
// Regardless of the source image type, the resulting texture is always in RGBA
// format.
//
func TextureFromImage(src image.Image, params ...TextureParameter) *Texture {
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

func newTexture(width, height int, format int32, pix *uint8, params ...TextureParameter) *Texture {
	var tex uint32
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.GL_TEXTURE_2D, tex)

	t := &Texture{width: width, height: height, glID: tex}

	t.setParams(params...)

	// TODO: this works with RGBA images, need to adjust if we handle more formats.
	gl.PixelStorei(gl.GL_UNPACK_ALIGNMENT, 4)
	gl.TexImage2D(gl.GL_TEXTURE_2D, 0, format, int32(width), int32(height), 0, uint32(format), gl.GL_UNSIGNED_BYTE, gl.Ptr(pix))
	return t
}

// Parameters sets the given texture parameters.
//
func (t *Texture) Parameters(params ...TextureParameter) {
	if len(params) == 0 {
		return
	}
	gl.BindTexture(gl.GL_TEXTURE_2D, t.glID)
	t.setParams(params...)
}

func (t *Texture) setParams(params ...TextureParameter) {
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
}

// Bind binds the texture in the OpenGL context.
//
func (t *Texture) Bind() {
	gl.BindTexture(gl.GL_TEXTURE_2D, t.glID)
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
}

// GLCoords return the coordinates of the point pt mapped to the range [0, 1].
//
func (t *Texture) GLCoords(pt Point) Point {
	return Point{
		X: float32(pt.X) / float32(t.width),
		Y: float32(pt.Y) / float32(t.height),
	}
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

// Region is a Drawable that represents a sub-region in a Texture or
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

// Rect returns the region's bounding rectangle within the parent texture.
//
func (r *Region) Rect() image.Rectangle {
	return r.bounds
}

// Size retruns the size of the region.
//
func (r *Region) Size() image.Point {
	return r.bounds.Size()
}

// UV returns the regions's UV coordinates in the range [0, 1]
//
func (r *Region) UV() [4]float32 {
	w, h := float32(r.width), float32(r.height)
	u0, v0 := float32(r.bounds.Min.X)/w, float32(r.bounds.Min.Y)/h
	u1, v1 := float32(r.bounds.Max.X)/w, float32(r.bounds.Max.Y)/h
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
