package batch

import (
	"image/color"
	"math"

	"github.com/db47h/grog"
	"github.com/db47h/grog/gl"
)

const (
	floatsPerVertex = 8
	floatsPerQuad   = floatsPerVertex * 4
	indicesPerQuad  = 6
	batchSize       = 10000
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
	vbo      uint32
	ebo      uint32
	vertices []float32
	texture  grog.Drawable
	proj     [16]float32
	index    int
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

	indices := make([]uint32, batchSize*indicesPerQuad)
	b.vertices = make([]float32, batchSize*floatsPerQuad)
	for i, j := 0, uint32(0); i < len(indices); i, j = i+indicesPerQuad, j+4 {
		indices[i+0] = j + 0
		indices[i+1] = j + 1
		indices[i+2] = j + 2
		indices[i+3] = j + 2
		indices[i+4] = j + 1
		indices[i+5] = j + 3
	}

	gl.BindBuffer(gl.GL_ARRAY_BUFFER, b.vbo)
	gl.BufferData(gl.GL_ARRAY_BUFFER, batchSize*floatsPerQuad*4, nil, gl.GL_DYNAMIC_DRAW)

	gl.BindBuffer(gl.GL_ELEMENT_ARRAY_BUFFER, b.ebo)
	gl.BufferData(gl.GL_ELEMENT_ARRAY_BUFFER, len(indices)*4, gl.Ptr(&indices[0]), gl.GL_STATIC_DRAW)

	gl.Enable(gl.GL_BLEND)
	gl.BlendFunc(gl.GL_SRC_ALPHA, gl.GL_ONE_MINUS_SRC_ALPHA)
	// gl.BlendFunc(gl.GL_ONE, gl.GL_ONE_MINUS_SRC_ALPHA)

	return b, nil
}

func (b *Batch) Begin() {
	if b.index != 0 {
		panic("call Flush() before Begin()")
	}
	b.program.Use()
	gl.ActiveTexture(gl.GL_TEXTURE0)
	gl.Uniform1i(b.uniform.tex, 0)
	gl.BindBuffer(gl.GL_ARRAY_BUFFER, b.vbo)
	gl.BindBuffer(gl.GL_ELEMENT_ARRAY_BUFFER, b.ebo)
	gl.EnableVertexAttribArray(b.attr.pos)
	gl.VertexAttribOffset(b.attr.pos, 4, gl.GL_FLOAT, gl.GL_FALSE, floatsPerVertex*4, 0)
	gl.EnableVertexAttribArray(b.attr.color)
	gl.VertexAttribOffset(b.attr.color, 4, gl.GL_FLOAT, gl.GL_FALSE, floatsPerVertex*4, 4*4)
	// gl.BatchBegin(b.program, []float32(proj[:]), b.vbo, b.ebo, b.attr.pos, b.attr.color, b.uniform.cam, b.uniform.tex)
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
	gl.Viewport(int32(v.Min.X), int32(v.Min.Y), int32(v.Dx()), int32(v.Dy()))
}

func (b *Batch) Draw(d grog.Drawable, x, y, scaleX, scaleY, rot float32, c color.Color) {
	if b.index >= batchSize {
		b.Flush()
	}

	if b.index > 0 {
		if b.texture.NativeID() != d.NativeID() {
			b.Flush()
			b.texture = d
		}
	} else {
		b.texture = d
	}

	var rf, gf, bf, af float32 = 1.0, 1.0, 1.0, 1.0
	if c != nil {
		nc := color.NRGBAModel.Convert(c).(color.NRGBA)
		rf, gf, bf, af = float32(nc.R)/255.0, float32(nc.G)/255.0, float32(nc.B)/255.0, float32(nc.A)/255.0
	}

	// optimized version of ngl32 matrix transforms => +25% ups
	var m0, m1, m3, m4, m6, m7 float32 = 1, 0, 0, 1, float32(x), float32(y)
	if rot != 0 {
		sin, cos := float32(math.Sin(float64(rot))), float32(math.Cos(float64(rot)))
		m0, m1, m3, m4 = cos, sin, -sin, cos
	}

	o := d.Origin()
	tx, ty := -float32(o.X)*scaleX, -float32(o.Y)*scaleY
	m6, m7 = m0*tx+m3*ty+m6, m1*tx+m4*ty+m7

	sz := d.Size()
	sX, sY := scaleX*float32(sz.X), scaleY*float32(sz.Y)
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
	gl.BindTexture(gl.GL_TEXTURE_2D, b.texture.NativeID())
	if binder, ok := b.texture.(grog.Binder); ok {
		binder.OnBind()
	}

	gl.BufferSubData(gl.GL_ARRAY_BUFFER, 0, b.index*floatsPerQuad*4, gl.Ptr(&b.vertices[0]))
	gl.DrawElements(gl.GL_TRIANGLES, int32(b.index*indicesPerQuad), gl.GL_UNSIGNED_INT, nil)
	b.index = 0
	b.vertices = b.vertices[:0]
	// gl.BatchDraw(tex, vert)
}

func (b *Batch) End() {
	b.Flush()
}

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
