package batch

import (
	"image/color"
	"math"

	"github.com/db47h/grog"
	"github.com/db47h/grog/gl"
)

// A Batch draws sprites in batches.
//
type Batch struct {
	program gl.Program
	attr    struct {
		pos   uint32
		color uint32
	}
	uniform struct {
		cam int32
		tex int32
	}
	vbo   uint32
	ebo   uint32
	index int

	vertices []float32
	texture  grog.Drawable
	proj     [16]float32
}

func New() (*Batch, error) {
	var (
		b   = new(Batch)
		err error
	)
	b.program, err = loadShaders()
	if err != nil {
		return nil, err
	}
	b.attr.pos, err = b.program.AttribLocation("aPos")
	if err != nil {
		return nil, err
	}
	b.attr.color, err = b.program.AttribLocation("aColor")
	if err != nil {
		return nil, err
	}
	b.uniform.cam = b.program.UniformLocation("uProjection")
	b.uniform.tex = b.program.UniformLocation("uTexture")
	gl.GenBuffers(1, &b.vbo)
	gl.GenBuffers(1, &b.ebo)

	b.vertices = make([]float32, batchSize*floatsPerQuad)
	batchInit(b.vbo, b.ebo)

	return b, nil
}

func (b *Batch) Begin() {
	if b.index != 0 {
		panic("call Flush() before Begin()")
	}
	batchBegin(b.vbo, b.ebo, b.program, b.attr.pos, b.attr.color, b.uniform.tex)
}

func (b *Batch) SetProjectionMatrix(projection [16]float32) {
	if b.index != 0 {
		b.Flush()
	}
	gl.UniformMatrix4fv(b.uniform.cam, 1, gl.GL_FALSE, &projection[0])
	b.proj = projection
}

// SetView wraps SetProjectionMatrix(view.ProjectionMatrix()) and gl.Viewport() into
// a single call.
//
func (b *Batch) SetView(v *grog.View) {
	b.SetProjectionMatrix(v.ProjectionMatrix())
	r := v.GLRect()
	gl.Viewport(int32(r.Min.X), int32(r.Min.Y), int32(r.Dx()), int32(r.Dy()))
}

func (b *Batch) Draw(d grog.Drawable, dp, scale grog.Point, rot float32, c color.Color) {
	if b.index >= batchSize {
		b.Flush()
	}

	if b.index == 0 {
		b.texture = d
	} else if b.texture.NativeID() != d.NativeID() {
		b.Flush()
		b.texture = d
	}

	var rf, gf, bf, af float32 = 1.0, 1.0, 1.0, 1.0
	if c != nil {
		nc := color.RGBAModel.Convert(c).(color.RGBA)
		rf, gf, bf, af = float32(nc.R)/255.0, float32(nc.G)/255.0, float32(nc.B)/255.0, float32(nc.A)/255.0
	}

	// optimized version of ngl32 matrix transforms => +25% ups
	var m0, m1, m3, m4, m6, m7 float32 = 1, 0, 0, 1, dp.X, dp.Y
	if rot != 0 {
		sin, cos := float32(math.Sin(float64(rot))), float32(math.Cos(float64(rot)))
		m0, m1, m3, m4 = cos, sin, -sin, cos
	}

	o := d.Origin()
	tx, ty := -float32(o.X)*scale.X, -float32(o.Y)*scale.Y
	m6, m7 = m0*tx+m3*ty+m6, m1*tx+m4*ty+m7

	sz := d.Size()
	sX, sY := scale.X*float32(sz.X), scale.Y*float32(sz.Y)
	m0 *= sX
	m1 *= sX
	m3 *= sY
	m4 *= sY

	uv := d.UV()
	b.vertices = append(b.vertices,
		// top left
		m3+m6, m4+m7, uv[0], uv[1], rf, gf, bf, af,
		// top right
		m0+m3+m6, m1+m4+m7, uv[2], uv[1], rf, gf, bf, af,
		// bottom left
		m6, m7, uv[0], uv[3], rf, gf, bf, af,
		// bottom right
		m0+m6, m1+m7, uv[2], uv[3], rf, gf, bf, af,
	)
	b.index++
}

func (b *Batch) Flush() {
	if b.index == 0 {
		return
	}
	b.texture.Bind()

	gl.BufferSubData(gl.GL_ARRAY_BUFFER, 0, b.index*floatsPerQuad*4, gl.Ptr(&b.vertices[0]))
	gl.DrawElements(gl.GL_TRIANGLES, int32(b.index*indicesPerQuad), gl.GL_UNSIGNED_INT, nil)
	b.index = 0
	b.vertices = b.vertices[:0]
	// gl.BatchDraw(tex, vert)
}

func (b *Batch) End() {
	b.Flush()
}
