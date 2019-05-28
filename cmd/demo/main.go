// +build !glfw

package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math/rand"
	"time"

	"github.com/db47h/grog"
	"github.com/db47h/grog/app"
	"github.com/db47h/grog/assets"
	"github.com/db47h/grog/batch"
	"github.com/db47h/grog/debug"
	"github.com/db47h/grog/text"
	"github.com/db47h/grog/texture"
	"github.com/db47h/ofs"
)

func main() {
	// preload assets
	fs := new(ofs.Overlay)
	_ = fs.Add(false, "cmd/demo/assets", "assets")
	mgr := assets.NewManager(fs, assets.FontPath("fonts"), assets.TexturePath("textures"))
	mgr.PreloadTexture("box.png", texture.Filter(texture.Linear, texture.Nearest))
	mgr.PreloadTexture("tile.png",
		texture.Filter(texture.ClampToEdge, texture.ClampToEdge),
		texture.Filter(texture.Linear, texture.Nearest))
	mgr.PreloadFont("DejaVuSansMono.ttf")

	err := app.Main(&myApp{mgr: mgr}, app.Title("grog Demo"), app.FullScreen())

	if err != nil {
		log.Print(err)
	}
}

type myApp struct {
	mgr   *assets.Manager
	b     *batch.ConcurrentBatch
	w, h  int
	ft    time.Duration
	fts   debug.Timer // average frame time
	boxes []box
}

type box struct {
	pos   grog.Point
	scale float32
	rot   float32

	dPos grog.Point
	dRot float32
	r    *texture.Region
}

func (b *box) update(dt float32) (grog.Point, float32) {
	return b.pos.Add(b.dPos.Mul(dt)), b.rot + b.dRot*dt
}

func (a *myApp) Init(w app.Window) error {
	log.Print(app.DriverVersion())

	// wait for assets
	if err := a.mgr.Wait(); err != nil {
		return err
	}

	// some init
	b, err := batch.NewConcurrent()
	if err != nil {
		return err
	}
	a.b = b
	sz := w.FrameBuffer().RootView().Size()
	a.w, a.h = sz.X, sz.Y

	var sprites [4]*texture.Region
	boxAtlas, _ := a.mgr.Texture("box.png")
	sprites[0] = boxAtlas.Region(image.Rect(1, 1, 33, 33), image.Pt(16, 16))
	sprites[1] = boxAtlas.Region(image.Rect(34, 1, 66, 33), image.Pt(16, 16))
	sprites[2] = boxAtlas.Region(image.Rect(1, 34, 33, 66), image.Pt(16, 16))
	sprites[3] = boxAtlas.Region(image.Rect(34, 34, 66, 66), image.Pt(16, 16))

	rand.Seed(424242)
	for i := 0; i < 70000; i++ {
		a.boxes = append(a.boxes, box{
			pos:   grog.PtI(a.w/2, a.h/2),
			scale: 2 - rand.Float32()*1.5,
			rot:   0,
			dPos:  grog.Pt(rand.Float32()*800-400, rand.Float32()*800-400),
			dRot:  rand.Float32(),
			r:     sprites[rand.Intn(4)],
		})
	}

	return nil
}

func (a *myApp) Terminate() error {
	return a.mgr.Close()
}

func (a *myApp) OnUpdate(dt time.Duration) {
	a.ft += dt
	t := float32(float64(dt) / float64(time.Second))
	for i := range a.boxes {
		b := &a.boxes[i]
		b.pos, b.rot = b.update(t)

		border := 16 * b.scale
		if b.pos.X < border {
			b.pos.X = border + border - b.pos.X
			b.dPos.X = -b.dPos.X
		}
		if b.pos.Y < border {
			b.pos.Y = border + border - b.pos.Y
			b.dPos.Y = -b.dPos.Y
		}
		if right := float32(a.w) - 16*b.scale; b.pos.X >= right {
			b.pos.X = right*2 - b.pos.X
			b.dPos.X = -b.dPos.X
		}
		if bottom := float32(a.h) - 16*b.scale; b.pos.Y >= bottom {
			b.pos.Y = bottom*2 - b.pos.Y
			b.dPos.Y = -b.dPos.Y
		}
	}
}

func (a *myApp) OnDraw(w app.Window, dt time.Duration) {
	b := a.b
	v := w.FrameBuffer().RootView()
	b.Begin()
	b.Camera(v)
	b.Clear(color.NRGBA{B: 50, A: 255})

	t := float32(float64(dt) / float64(time.Second))
	for i := range a.boxes {
		box := &a.boxes[i]
		pos, rot := box.update(t)
		b.Draw(box.r, pos, grog.Pt(box.scale, box.scale), rot, nil)
	}

	a.fts.Add(a.ft + dt)
	a.ft = -dt

	fnt, _ := a.mgr.FontDrawer("DejaVuSansMono.ttf", 16, text.HintingNone, texture.Nearest)
	debug.InfoBox(b, fnt, v, 0, fmt.Sprintf("%.0f fps", a.fts.PerSecond()))
	b.End()
}

func (a *myApp) OnFrameBufferSize(_ app.Window, w, h int) {
	a.w = w
	a.h = h
}
