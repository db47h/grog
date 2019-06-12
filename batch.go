package grog

import (
	"image/color"
	"math"

	"github.com/db47h/grog/gl"
)

const (
	floatsPerVertex = 8
	floatsPerQuad   = floatsPerVertex * 4
	indicesPerQuad  = 6
	batchSize       = 5000
)

func loadShaders() (gl.Program, error) {
	var (
		vertex, frag gl.Shader
		err          error
	)
	vertex, err = gl.NewShader(gl.GL_VERTEX_SHADER, vertexShader)
	if err != nil {
		return 0, err
	}
	defer vertex.Delete()
	frag, err = gl.NewShader(gl.GL_FRAGMENT_SHADER, fragmentShader)
	if err != nil {
		return 0, err
	}
	defer frag.Delete()

	program, err := gl.NewProgram(vertex, frag)
	if err != nil {
		return 0, err
	}

	return program, nil
}

func batchInit(vbo, ebo uint32) {
	indices := make([]uint32, batchSize*indicesPerQuad)
	for i, j := 0, uint32(0); i < len(indices); i, j = i+indicesPerQuad, j+4 {
		indices[i+0] = j + 0
		indices[i+1] = j + 1
		indices[i+2] = j + 2
		indices[i+3] = j + 2
		indices[i+4] = j + 1
		indices[i+5] = j + 3
	}

	gl.BindBuffer(gl.GL_ARRAY_BUFFER, vbo)
	gl.BufferData(gl.GL_ARRAY_BUFFER, batchSize*floatsPerQuad*4, nil, gl.GL_DYNAMIC_DRAW)

	gl.BindBuffer(gl.GL_ELEMENT_ARRAY_BUFFER, ebo)
	gl.BufferData(gl.GL_ELEMENT_ARRAY_BUFFER, len(indices)*4, gl.Ptr(&indices[0]), gl.GL_STATIC_DRAW)

	gl.Enable(gl.GL_SCISSOR_TEST)
	gl.Enable(gl.GL_BLEND)
	gl.BlendFunc(gl.GL_ONE, gl.GL_ONE_MINUS_SRC_ALPHA)
}

func batchBegin(vbo, ebo uint32, program gl.Program, pos, color uint32, texture int32) {
	program.Use()
	gl.ActiveTexture(gl.GL_TEXTURE0)
	gl.Uniform1i(texture, 0)
	gl.BindBuffer(gl.GL_ARRAY_BUFFER, vbo)
	gl.BindBuffer(gl.GL_ELEMENT_ARRAY_BUFFER, ebo)
	gl.EnableVertexAttribArray(pos)
	gl.VertexAttribOffset(pos, 4, gl.GL_FLOAT, gl.GL_FALSE, floatsPerVertex*4, 0)
	gl.EnableVertexAttribArray(color)
	gl.VertexAttribOffset(color, 4, gl.GL_FLOAT, gl.GL_FALSE, floatsPerVertex*4, 4*4)
}

func NewBatch(concurrent bool) (BatchRenderer, error) {
	if concurrent {
		return newConcurrentBatch()
	}
	return newBatch()
}

// A batch draws sprites in batches.
//
type batch struct {
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
	texture  Drawable
	proj     [16]float32
}

func newBatch() (*batch, error) {
	var (
		b   = new(batch)
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

func (b *batch) Begin() {
	if b.index != 0 {
		panic("call Flush() before Begin()")
	}
	batchBegin(b.vbo, b.ebo, b.program, b.attr.pos, b.attr.color, b.uniform.tex)
}

// Camera sets the camera for world to screen transforms and clipping region.
//
func (b *batch) Camera(c Camera) {
	if b.index != 0 {
		b.Flush()
	}
	proj := c.ProjectionMatrix()
	gl.UniformMatrix4fv(b.uniform.cam, 1, gl.GL_FALSE, &proj[0])
	b.proj = proj
	r := c.GLRect()
	gl.Scissor(int32(r.Min.X), int32(r.Min.Y), int32(r.Dx()), int32(r.Dy()))
}

func (b *batch) Draw(d Drawable, dp, scale Point, rot float32, c color.Color) {
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
		c := gl.ColorModel.Convert(c).(gl.Color)
		rf, gf, bf, af = c.R, c.G, c.B, c.A
	}

	// optimized version of ngl32 matrix transforms => +25% ups
	var m0, m1, m3, m4 float32 = 1, 0, 0, 1
	if rot != 0 {
		sin, cos := float32(math.Sin(float64(rot))), float32(math.Cos(float64(rot)))
		m0, m1, m3, m4 = cos, sin, -sin, cos
	}

	o := d.Origin()
	tx, ty := float32(o.X)*scale.X, float32(o.Y)*scale.Y
	m6, m7 := dp.X-m0*tx-m3*ty, dp.Y-m1*tx-m4*ty

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

func (b *batch) Flush() {
	if b.index == 0 {
		return
	}
	b.texture.Bind()

	gl.BufferSubData(gl.GL_ARRAY_BUFFER, 0, b.index*floatsPerQuad*4, gl.Ptr(&b.vertices[0]))
	gl.DrawElements(gl.GL_TRIANGLES, int32(b.index*indicesPerQuad), gl.GL_UNSIGNED_INT, nil)
	b.index = 0
	b.vertices = b.vertices[:0]
}

func (b *batch) End() {
	b.Flush()
}

func (b *batch) Clear(c color.Color) {
	b.Flush()
	if c != nil {
		c := gl.ColorModel.Convert(c).(gl.Color)
		gl.ClearColor(c.R, c.G, c.B, c.A)
	}
	gl.Clear(gl.GL_COLOR_BUFFER_BIT)
}

func (b *batch) Close() {
	b.program.Delete()
	gl.DeleteBuffers(1, &b.ebo)
	gl.DeleteBuffers(1, &b.vbo)
}
