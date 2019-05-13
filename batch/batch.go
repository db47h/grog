package batch

import (
	"image/color"

	"github.com/db47h/grog"
	"github.com/db47h/grog/gl"
	"github.com/go-gl/mathgl/mgl32"
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
	texture  uint32
	proj     mgl32.Mat4
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
	gl.BufferData(gl.GL_ARRAY_BUFFER, len(b.vertices)*4, nil, gl.GL_DYNAMIC_DRAW)

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

func (b *Batch) SetProjectionMatrix(projection mgl32.Mat4) {
	if b.index != 0 {
		b.flush()
	}
	gl.UniformMatrix4fv(b.uniform.cam, 1, gl.GL_FALSE, &projection[0])
	b.proj = projection
}

func (b *Batch) Draw(d grog.Drawable, x, y, scaleX, scaleY float32, rot float32, color color.NRGBA) {
	var idx int
	tex := d.NativeID()
	if b.index > 0 {
		if b.texture != tex {
			b.flush()
			b.texture = tex
		}
		idx = b.index * floatsPerQuad
	} else {
		b.texture = tex
	}
	rf, gf, bf, af := float32(color.R)/255.0, float32(color.G)/255.0, float32(color.B)/255.0, float32(color.A)/255.0
	uv := d.UV()
	sz := d.Size()
	o := d.Origin()
	model := mgl32.Translate2D(x, y)
	if rot != 0 {
		model = model.Mul3(mgl32.Rotate3DZ(rot))
	}
	model = model.Mul3(mgl32.Translate2D(-float32(o.X)*scaleX, -float32(o.Y)*scaleY))
	model = model.Mul3(mgl32.Scale2D(float32(sz.X)*scaleX, float32(sz.Y)*scaleY))
	// tl := model.Mul3x1(mgl32.Vec3{0, 1, 1})
	tlX, tlY := model[0]*0+model[3]*1+model[6]*1, model[1]*0+model[4]*1+model[7]*1
	// tr := model.Mul3x1(mgl32.Vec3{1, 1, 1})
	trX, trY := model[0]*1+model[3]*1+model[6]*1, model[1]*1+model[4]*1+model[7]*1
	// bl := model.Mul3x1(mgl32.Vec3{0, 0, 1})
	blX, blY := model[0]*0+model[3]*0+model[6]*1, model[1]*0+model[4]*0+model[7]*1
	// br := model.Mul3x1(mgl32.Vec3{1, 0, 1})
	brX, brY := model[0]*1+model[3]*0+model[6]*1, model[1]*1+model[4]*0+model[7]*1
	copy(b.vertices[idx:], []float32{
		tlX, tlY, uv[0], uv[1], rf, gf, bf, af, // top left
		trX, trY, uv[2], uv[1], rf, gf, bf, af, // top right
		blX, blY, uv[0], uv[3], rf, gf, bf, af, // bottom left
		brX, brY, uv[2], uv[3], rf, gf, bf, af, // bottom right
	})
	b.index++
	if b.index >= batchSize {
		b.flush()
	}
}

func (b *Batch) flush() {
	gl.BindTexture(gl.GL_TEXTURE_2D, b.texture)

	gl.BufferSubData(gl.GL_ARRAY_BUFFER, 0, b.index*floatsPerQuad*4, gl.Ptr(&b.vertices[0]))
	gl.DrawElements(gl.GL_TRIANGLES, int32(b.index*indicesPerQuad), gl.GL_UNSIGNED_INT, nil)
	b.index = 0
	// gl.BatchDraw(tex, vert)
}

func (b *Batch) End() {
	b.flush()
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
