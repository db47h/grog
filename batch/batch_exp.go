// +build exp

package batch

import (
	"image"
	"image/color"
	"math"
	"runtime"
	"sync"

	"github.com/db47h/grog"
	"github.com/db47h/grog/gl"
)

const (
	floatsPerVertex = 8
	floatsPerQuad   = floatsPerVertex * 4
	indicesPerQuad  = 6
	batchSize       = 10000
)

type drawCmd struct {
	d              grog.Drawable
	x, y           float32
	scaleX, scaleY float32
	rot            float32
	c              color.Color
}

type work struct {
	cmds     []drawCmd
	vertices []float32
}

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
	vbo uint32
	ebo uint32

	curBuf     int
	inFlight   int
	drawBuf    [2][batchSize]drawCmd
	drawChan   chan []drawCmd
	vertexChan chan []float32
	texture    [2]grog.Drawable
	// proj is a slice instead of a [2][16]float32 because we pass a pointer to
	// it to cgo and we don't want cgo to hold onto the whole structure memory
	proj  [2][]float32
	view  [2]image.Rectangle
	index int
}

func New() (*Batch, error) {
	var (
		b = &Batch{
			drawChan:   make(chan []drawCmd, 1),
			vertexChan: make(chan []float32),
			proj: [...][]float32{
				make([]float32, 16),
				make([]float32, 16),
			},
		}
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
	// b.vertices = make([]float32, batchSize*floatsPerQuad)
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

	go worker(b.drawChan, b.vertexChan)

	return b, nil
}

func worker(in <-chan []drawCmd, out chan<- []float32) {
	var (
		buf     int
		v       [2][batchSize * floatsPerQuad]float32
		wg      sync.WaitGroup
		wc          = make(chan work)
		th      int = 100 // threshold for dispatching to child workers
		workers     = runtime.NumCPU()
	)

	for i := 0; i < workers; i++ {
		go func() {
			for wi := range wc {
				processCmds(wi.cmds, wi.vertices)
				wg.Done()
			}
		}()
	}

	for cmds := range in {
		start := 0
		count := len(cmds)
		if count > th {
			count = (count + workers - 1) / workers
			for start < len(cmds) {
				end := start + count
				if end > len(cmds) {
					end = len(cmds)
				}
				if end > start {
					wg.Add(1)
					wc <- work{cmds: cmds[start:end], vertices: v[buf][start*floatsPerQuad : end*floatsPerQuad]}
					start = end
				}
			}
			wg.Wait()
		} else {
			processCmds(cmds, v[buf][:])
		}
		out <- v[buf][:len(cmds)*floatsPerQuad]
		buf ^= 1
	}
	close(wc)
}

func processCmds(cmds []drawCmd, vertices []float32) {
	for i := range cmds {
		d := &cmds[i]

		var rf, gf, bf, af float32 = 1.0, 1.0, 1.0, 1.0
		if d.c != nil {
			nc := color.NRGBAModel.Convert(d.c).(color.NRGBA)
			rf, gf, bf, af = float32(nc.R)/255.0, float32(nc.G)/255.0, float32(nc.B)/255.0, float32(nc.A)/255.0
		}

		// optimized version of ngl32 matrix transforms => +25% ups
		var m0, m1, m3, m4, m6, m7 float32 = 1, 0, 0, 1, float32(d.x), float32(d.y)
		if rot := d.rot; rot != 0 {
			sin, cos := float32(math.Sin(float64(rot))), float32(math.Cos(float64(rot)))
			m0, m1, m3, m4 = cos, sin, -sin, cos
		}

		o := d.d.Origin()
		tx, ty := -float32(o.X)*d.scaleX, -float32(o.Y)*d.scaleY
		m6, m7 = m0*tx+m3*ty+m6, m1*tx+m4*ty+m7

		sz := d.d.Size()
		sX, sY := d.scaleX*float32(sz.X), d.scaleY*float32(sz.Y)
		m0 *= sX
		m1 *= sX
		m3 *= sY
		m4 *= sY

		uv := d.d.UV()
		copy(vertices[i*floatsPerQuad:], []float32{
			// top left
			m3 + m6, m4 + m7, uv[0], uv[1], rf, gf, bf, af,
			// top right
			m0 + m3 + m6, m1 + m4 + m7, uv[2], uv[1], rf, gf, bf, af,
			// bottom left
			m6, m7, uv[0], uv[3], rf, gf, bf, af,
			// bottom right
			m0 + m6, m1 + m7, uv[2], uv[3], rf, gf, bf, af,
		})
	}
}

func (b *Batch) Begin() {
	if b.index != 0 || b.inFlight > 0 {
		panic("call End() before Begin()")
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
		b.flush()
	}
	copy(b.proj[b.curBuf], projection[:])
}

// SetView wraps SetProjectionMatrix(view.ProjectionMatrix()) and gl.Viewport() into
// a single call.
//
func (b *Batch) SetView(v *grog.View) {
	b.SetProjectionMatrix(v.ProjectionMatrix())
	b.view[b.curBuf] = v.Rectangle
	// gl.Viewport(int32(v.Min.X), int32(v.Min.Y), int32(v.Dx()), int32(v.Dy()))
}

func (b *Batch) Draw(d grog.Drawable, x, y, scaleX, scaleY, rot float32, c color.Color) {
	if b.index >= batchSize {
		b.flush()
	}

	if b.index == 0 {
		b.texture[b.curBuf] = d
	} else if b.texture[b.curBuf].NativeID() != d.NativeID() {
		b.flush()
		b.texture[b.curBuf] = d
	}

	b.drawBuf[b.curBuf][b.index] = drawCmd{d, x, y, scaleX, scaleY, rot, c}
	b.index++
}

func (b *Batch) Flush() {
	b.flush()
	if b.inFlight > 0 {
		b.flush()
	}
}

func (b *Batch) flush() {
	var (
		vertices []float32
		altBuf   = b.curBuf ^ 1
	)

	// get result of last transform
	if b.inFlight > 0 {
		vertices = <-b.vertexChan
		b.inFlight--
	}

	// send more work before drawing
	if b.index > 0 {
		b.inFlight++
		b.drawChan <- b.drawBuf[b.curBuf][:b.index]
	}

	if len(vertices) > 0 {
		v := &b.view[altBuf]
		gl.Viewport(int32(v.Min.X), int32(v.Min.Y), int32(v.Dx()), int32(v.Dy()))
		gl.UniformMatrix4fv(b.uniform.cam, 1, gl.GL_FALSE, &b.proj[altBuf][0])

		tex := b.texture[altBuf]
		gl.BindTexture(gl.GL_TEXTURE_2D, tex.NativeID())
		if binder, ok := tex.(grog.Binder); ok {
			binder.OnBind()
		}
		gl.BufferSubData(gl.GL_ARRAY_BUFFER, 0, len(vertices)*4, gl.Ptr(&vertices[0]))
		gl.DrawElements(gl.GL_TRIANGLES, int32(len(vertices)/floatsPerVertex*indicesPerQuad), gl.GL_UNSIGNED_INT, nil)
	}

	if b.index == 0 {
		return
	}

	oldBuf := b.curBuf
	b.curBuf ^= 1
	copy(b.proj[b.curBuf], b.proj[oldBuf])
	b.view[b.curBuf] = b.view[oldBuf]
	b.index = 0
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
