package grog

import (
	"image"
	"image/color"
	"math"
	"runtime"
	"sync"

	"github.com/db47h/grog/gl"
)

type drawCmd struct {
	d              Drawable
	x, y           float32
	scaleX, scaleY float32
	rot            float32
	c              color.Color
}

type work struct {
	cmds     []drawCmd
	vertices []float32
}

// A concurrentBatch draws sprites in batches and computes model transformations concurrently.
//
type concurrentBatch struct {
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

	drawChan   chan []drawCmd
	vertexChan chan []float32
	inFlight   int
	cb         int
	buf        [2]struct {
		cmds    [batchSize]drawCmd
		texture Drawable
		proj    []float32
		view    image.Rectangle
		updView bool
	}
}

func newConcurrentBatch() (*concurrentBatch, error) {
	var (
		b = &concurrentBatch{
			drawChan:   make(chan []drawCmd, 1),
			vertexChan: make(chan []float32),
		}
		err error
	)
	b.buf[0].proj = make([]float32, 16)
	b.buf[1].proj = make([]float32, 16)

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

		var rf, gf, bf, af float32 = 1, 1, 1, 1
		if d.c != nil {
			c := gl.ColorModel.Convert(d.c).(gl.Color)
			rf, gf, bf, af = c.R, c.G, c.B, c.A
		}

		// optimized version of ngl32 matrix transforms => +25% ups
		var m0, m1, m3, m4 float32 = 1, 0, 0, 1
		if rot := d.rot; rot != 0 {
			sin, cos := float32(math.Sin(float64(rot))), float32(math.Cos(float64(rot)))
			m0, m1, m3, m4 = cos, sin, -sin, cos
		}

		o := d.d.Origin()
		tx, ty := float32(o.X)*d.scaleX, float32(o.Y)*d.scaleY
		m6, m7 := d.x-m0*tx-m3*ty, d.y-m1*tx-m4*ty

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

func (b *concurrentBatch) Begin() {
	if b.index != 0 || b.inFlight > 0 {
		panic("call End() before Begin()")
	}
	batchBegin(b.vbo, b.ebo, b.program, b.attr.pos, b.attr.color, b.uniform.tex)
}

// Camera sets the camera for world to screen transforms and clipping region.
//
func (b *concurrentBatch) Camera(c Camera) {
	if b.index != 0 {
		b.flush()
	}
	proj := c.ProjectionMatrix()
	copy(b.buf[b.cb].proj, proj[:])
	b.buf[b.cb].view = c.GLRect()
	b.buf[b.cb].updView = true
}

func (b *concurrentBatch) Draw(d Drawable, dp, scale Point, rot float32, c color.Color) {
	if b.index >= batchSize {
		b.flush()
	}

	if b.index == 0 {
		b.buf[b.cb].texture = d
	} else if b.buf[b.cb].texture.NativeID() != d.NativeID() {
		b.flush()
		b.buf[b.cb].texture = d
	}

	b.buf[b.cb].cmds[b.index] = drawCmd{d, dp.X, dp.Y, scale.X, scale.Y, rot, c}
	b.index++
}

func (b *concurrentBatch) Flush() {
	b.flush()
	ab := b.cb ^ 1
	if b.inFlight > 0 || b.buf[ab].updView {
		b.flush()
	}
}

func (b *concurrentBatch) flush() {
	var vertices []float32

	// get result of last transform
	if b.inFlight > 0 {
		vertices = <-b.vertexChan
		b.inFlight--
	}

	// send more work before drawing
	if b.index > 0 {
		b.inFlight++
		b.drawChan <- b.buf[b.cb].cmds[:b.index]
	}

	b.cb ^= 1
	b.index = 0

	cb := &b.buf[b.cb]

	if cb.updView {
		v := cb.view
		gl.Scissor(int32(v.Min.X), int32(v.Min.Y), int32(v.Dx()), int32(v.Dy()))
		gl.UniformMatrix4fv(b.uniform.cam, 1, gl.GL_FALSE, &cb.proj[0])
		cb.updView = false
	}

	if vertices != nil {
		cb.texture.Bind()
		gl.BufferSubData(gl.GL_ARRAY_BUFFER, 0, len(vertices)*4, gl.Ptr(&vertices[0]))
		gl.DrawElements(gl.GL_TRIANGLES, int32(len(vertices)/floatsPerQuad*indicesPerQuad), gl.GL_UNSIGNED_INT, nil)
	}
}

func (b *concurrentBatch) End() {
	b.Flush()
}

func (b *concurrentBatch) Clear(c color.Color) {
	// TODO: optimize out the need to do a full flush
	b.Flush()
	if c != nil {
		c := gl.ColorModel.Convert(c).(gl.Color)
		gl.ClearColor(c.R, c.G, c.B, c.A)
	}
	gl.Clear(gl.GL_COLOR_BUFFER_BIT)
}
