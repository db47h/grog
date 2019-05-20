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
type ConcurrentBatch struct {
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

	curBuf     int
	inFlight   int
	drawBuf    [2][batchSize]drawCmd
	drawChan   chan []drawCmd
	vertexChan chan []float32
	texture    [2]grog.Drawable
	proj       [2][]float32
	view       [2]image.Rectangle
}

func NewConcurrent() (*ConcurrentBatch, error) {
	var (
		b = &ConcurrentBatch{
			drawChan:   make(chan []drawCmd, 1),
			vertexChan: make(chan []float32),
			proj: [2][]float32{
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

	batchInit(b.vbo, b.ebo)

	go worker(b.drawChan, b.vertexChan)

	return b, nil
}

func worker(in <-chan []drawCmd, out chan<- []float32) {
	var (
		buf     int
		v       [2][batchSize * floatsPerQuad]float32
		wg      sync.WaitGroup
		wc          = make(chan work)
		th      int = 500 // threshold for dispatching to child workers
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
			nc := color.RGBAModel.Convert(d.c).(color.RGBA)
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

func (b *ConcurrentBatch) Begin() {
	if b.index != 0 || b.inFlight > 0 {
		panic("call End() before Begin()")
	}
	batchBegin(b.vbo, b.ebo, b.program, b.attr.pos, b.attr.color, b.uniform.tex)
}

func (b *ConcurrentBatch) SetProjectionMatrix(projection [16]float32) {
	if b.index != 0 {
		b.flush()
	}
	copy(b.proj[b.curBuf], projection[:])
}

// SetView wraps SetProjectionMatrix(view.ProjectionMatrix()) and gl.Viewport() into
// a single call.
//
func (b *ConcurrentBatch) SetView(v *grog.View) {
	b.SetProjectionMatrix(v.ProjectionMatrix())
	b.view[b.curBuf] = v.GLRect()
}

func (b *ConcurrentBatch) Draw(d grog.Drawable, dp, scale grog.Point, rot float32, c color.Color) {
	if b.index >= batchSize {
		b.flush()
	}

	if b.index == 0 {
		b.texture[b.curBuf] = d
	} else if b.texture[b.curBuf].NativeID() != d.NativeID() {
		b.flush()
		b.texture[b.curBuf] = d
	}

	b.drawBuf[b.curBuf][b.index] = drawCmd{d, dp.X, dp.Y, scale.X, scale.Y, rot, c}
	b.index++
}

func (b *ConcurrentBatch) Flush() {
	b.flush()
	if b.inFlight > 0 {
		b.flush()
	}
}

func (b *ConcurrentBatch) flush() {
	var vertices []float32

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

	b.curBuf ^= 1
	b.index = 0

	if vertices != nil {
		cb := b.curBuf
		v := &b.view[cb]
		if v.Dx() > 0 {
			gl.Viewport(int32(v.Min.X), int32(v.Min.Y), int32(v.Dx()), int32(v.Dy()))
			v.Max.X = v.Min.X
		}
		if m33 := &b.proj[cb][15]; *m33 != 0 {
			gl.UniformMatrix4fv(b.uniform.cam, 1, gl.GL_FALSE, &b.proj[cb][0])
			*m33 = 0
		}

		b.texture[cb].Bind()
		gl.BufferSubData(gl.GL_ARRAY_BUFFER, 0, len(vertices)*4, gl.Ptr(&vertices[0]))
		gl.DrawElements(gl.GL_TRIANGLES, int32(len(vertices)/floatsPerQuad*indicesPerQuad), gl.GL_UNSIGNED_INT, nil)
	}
}

func (b *ConcurrentBatch) End() {
	b.Flush()
}
