// +build glfw

package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"log"
	"math/rand"
	"runtime"
	"time"

	"github.com/db47h/grog"
	"github.com/db47h/grog/assets"
	"github.com/db47h/grog/batch"
	"github.com/db47h/grog/gl"
	"github.com/db47h/grog/text"
	"github.com/db47h/grog/texture"
	"github.com/db47h/ofs"
	"github.com/go-gl/glfw/v3.2/glfw"
)

func init() {
	// This is needed to arrange that main() runs on main thread.
	// See documentation for functions that are only allowed to be called from the main thread.
	runtime.LockOSThread()
}

var (
	vsync = flag.Int("v", 1, "vsync value for glfw.SwapInterval")
)

func main() {
	flag.Parse()

	// preload assets
	var ovl ofs.Overlay
	if err := ovl.Add(false, "assets", "cmd/demo/assets"); err != nil {
		panic(err)
	}
	mgr := assets.NewManager(&ovl,
		assets.TexturePath("textures"),
		assets.FontPath("fonts"),
		assets.FilePath("."))
	mgr.PreloadTexture("box.png", texture.Filter(texture.Linear, texture.Nearest))
	mgr.PreloadTexture("tile.png",
		texture.Filter(texture.ClampToEdge, texture.ClampToEdge),
		texture.Filter(texture.Linear, texture.Nearest))
	mgr.PreloadFont("Go-Regular.ttf")

	// Init GLFW & window
	if err := glfw.Init(); err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	apiVer := gl.APIVersion()
	switch apiVer.API {
	case gl.OpenGL:
		glfw.WindowHint(glfw.ClientAPI, glfw.OpenGLAPI)
	case gl.OpenGLES:
		glfw.WindowHint(glfw.ClientAPI, glfw.OpenGLESAPI)
	default:
		panic("unsupported API")
	}
	glfw.WindowHint(glfw.ContextVersionMajor, apiVer.Major)
	glfw.WindowHint(glfw.ContextVersionMinor, apiVer.Minor)
	if gl.CoreProfile {
		glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	}
	glfw.WindowHint(glfw.Samples, 4)

	monitor := glfw.GetPrimaryMonitor()
	mode := monitor.GetVideoMode()
	glfw.WindowHint(glfw.RedBits, mode.RedBits)
	glfw.WindowHint(glfw.GreenBits, mode.GreenBits)
	glfw.WindowHint(glfw.BlueBits, mode.BlueBits)
	glfw.WindowHint(glfw.RefreshRate, mode.RefreshRate)
	window, err := glfw.CreateWindow(mode.Width, mode.Height, "grog demo", monitor, nil)
	if err != nil {
		panic(err)
	}

	// Init OpenGL
	window.MakeContextCurrent()
	gl.InitGo(glfw.GetProcAddress)

	log.Print("glfw ", glfw.GetVersionString())
	ver := gl.RuntimeVersion()
	log.Printf("%s %d.%d %s", ver.API.String(), ver.Major, ver.Minor, gl.GetGoString(gl.GL_VENDOR))

	// program state and glfw callbacks
	var (
		screen      = grog.NewScreen(image.Pt(window.GetFramebufferSize()))
		mouse       grog.Point
		mouseDrag   bool
		mouseDragPt grog.Point
		dv          grog.Point
		dAngle      float32
		topView     = &grog.View{Fb: screen, Scale: 1.0, OrgPos: grog.OrgCenter}
		textView    = &grog.View{Fb: screen, Scale: 1.0}
		mapView     = &grog.View{Fb: screen, Scale: 1.0}
		showTiles   bool
	)
	fbSz := screen.Size()
	gl.Viewport(0, 0, int32(fbSz.X), int32(fbSz.Y))

	window.SetScrollCallback(func(w *glfw.Window, xoff float64, yoff float64) {
		// save world coordinate under cursor
		p := topView.ScreenToWorld(mouse)
		switch {
		case yoff < 0:
			topView.Scale /= 1.1
		case yoff > 0:
			topView.Scale *= 1.1
		}
		// move view to keep p0 under cursor
		topView.Origin = topView.Origin.Add(p).Sub(topView.ScreenToWorld(mouse))
	})

	window.SetFramebufferSizeCallback(func(w *glfw.Window, width int, height int) {
		screen.Resize(image.Pt(width, height))
		gl.Viewport(0, 0, int32(width), int32(height))
	})

	window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		const scrollSpeed = 4
		if action == glfw.Repeat {
			return
		}
		if action == glfw.Release {
			switch key {
			case glfw.KeyUp, glfw.KeyW:
				dv.Y += scrollSpeed
			case glfw.KeyDown, glfw.KeyS:
				dv.Y -= scrollSpeed
			case glfw.KeyLeft, glfw.KeyA:
				dv.X += scrollSpeed
			case glfw.KeyRight, glfw.KeyD:
				dv.X -= scrollSpeed
			case glfw.KeyQ:
				dAngle -= 0.01
			case glfw.KeyE:
				dAngle += 0.01
			}
			return
		}

		switch key {
		case glfw.KeyEscape:
			w.SetShouldClose(true)
		case glfw.KeyUp, glfw.KeyW:
			dv.Y -= scrollSpeed
		case glfw.KeyDown, glfw.KeyS:
			dv.Y += scrollSpeed
		case glfw.KeyLeft, glfw.KeyA:
			dv.X -= scrollSpeed
		case glfw.KeyRight, glfw.KeyD:
			dv.X += scrollSpeed
		case glfw.KeyHome:
			topView.Origin = grog.Point{}
			topView.Scale = 1.0
			topView.Angle = 0
		case glfw.KeyQ:
			dAngle += 0.01
		case glfw.KeyE:
			dAngle -= 0.01
		case glfw.KeySpace:
			showTiles = !showTiles
		case glfw.Key1, glfw.KeyKP1:
			topView.Scale = 1
		case glfw.Key2, glfw.KeyKP2:
			topView.Scale = 2
		case glfw.Key3, glfw.KeyKP3:
			topView.Scale = 3
		case glfw.Key4, glfw.KeyKP4:
			topView.Scale = 4
		case glfw.Key5, glfw.KeyKP5:
			topView.Scale = 5
		case glfw.Key6, glfw.KeyKP6:
			topView.Scale = 6
		case glfw.Key7, glfw.KeyKP7:
			topView.Scale = 7
		case glfw.Key8, glfw.KeyKP8:
			topView.Scale = 8
		case glfw.KeyEqual, glfw.KeyKPAdd:
			topView.Scale *= 2
		case glfw.KeyMinus, glfw.KeyKPSubtract:
			topView.Scale /= 2
		}

	})

	window.SetMouseButtonCallback(func(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mod glfw.ModifierKey) {
		if button == glfw.MouseButton1 {
			switch action {
			case glfw.Press:
				mouseDrag = true
				mouseDragPt = topView.ScreenToWorld(mouse)
			case glfw.Release:
				mouseDrag = false
			}
		}
	})

	window.SetCursorPosCallback(func(w *glfw.Window, x, y float64) {
		mouse = grog.Pt(float32(x), float32(y))
		if mouseDrag {
			// set view center so that mouseDragPt is under the mouse
			topView.Origin = topView.Origin.Add(mouseDragPt).Sub(topView.ScreenToWorld(mouse))
		}
	})

	// Retrieve assets: we should have some kind of loading screen, but for the
	// demo, just waiting for assets to finish loading should be sufficient.
	if err = mgr.Wait(); err != nil {
		panic(err)
	}

	tex0, _ := mgr.Texture("box.png")
	sp0 := tex0.Region(image.Rect(1, 1, 66, 66), image.Pt(32, 32))
	sp1 := sp0.Region(image.Rect(33, 33, 65, 65), image.Pt(16, 16))
	// tex1, _ := assets.Texture("text.png")
	// sp1 := tex1.Region(image.Rectangle{Min: image.Point{}, Max: tex1.Size()}, image.Pt(0, 0))

	go16, _ := mgr.FontDrawer("Go-Regular.ttf", 16, text.HintingFull, texture.Nearest)

	tilesAtlas, _ := mgr.Texture("tile.png")
	var tiles []texture.Region
	for i := 0; i < 8; i++ {
		for j := 0; j < 4; j++ {
			tiles = append(tiles, *tilesAtlas.Region(image.Rect(i*16, j*16, i*16+16, j*16+16), image.Pt(8, 8)))
		}
	}

	// debug
	djv16, _ := mgr.FontDrawer("DejaVuSansMono.ttf", 16, text.HintingNone, texture.Nearest)
	dbg := debugSystem(djv16, screen)

	// setup a concurrent batch
	b, err := batch.NewConcurrent()
	if err != nil {
		panic(err)
	}

	// time and stats tracking
	const statSize = 32
	var (
		ts  = time.Now()
		fps [statSize]float64
		ups [statSize]float64
		ti          = 0
		rot float32 = 0
	)

	// static init
	glfw.SwapInterval(*vsync)
	bgColor := gl.Color{R: .2, G: .2, B: .2, A: 1}

	for !window.ShouldClose() {
		now := time.Now()
		dt := float64(now.Sub(ts)) / float64(time.Second)
		ts = now

		b.Begin()
		fbSz := screen.Size()
		topView.Rect.Max = image.Pt(fbSz.X, fbSz.Y/2)
		topView.Angle += dAngle
		topView.Pan(dv)
		b.Camera(topView)
		b.Clear(bgColor)

		rand.Seed(424242)
		rot += float32(dt)
		if showTiles {
			const worldSz = 320 // 320*320 = 102400 tiles
			for i := -worldSz / 2; i < worldSz/2; i++ {
				for j := -worldSz / 2; j < worldSz/2; j++ {
					// use atlasRegion instead of texture.Region
					b.Draw((*atlasRegion)(&tiles[rand.Intn(len(tiles))]), grog.PtI(i*16, j*16), grog.Pt(1, 1), 0.0, nil)
				}
			}
		} else {
			for i := 0; i < 20000; i++ {
				scale := grog.Pt(1, 1).Mul(rand.Float32() + 0.5)
				s := topView.Size()
				b.Draw(sp0, grog.PtI(rand.Intn(s.X*2)-s.X, rand.Intn(s.Y*2)-s.Y), scale, rot*(rand.Float32()+.5), nil)
				b.Draw(sp1, grog.PtI(rand.Intn(s.X*2)-s.X, rand.Intn(s.Y*2)-s.Y), scale, rot*(rand.Float32()+.5), nil)
			}
		}

		dbg(b, topView, 0, topView.ScreenToWorld(mouse).String())

		textView.Rect = image.Rectangle{Min: image.Pt(0, fbSz.Y/2), Max: fbSz}
		b.Camera(textView)
		b.Clear(gl.Color{R: .15, G: .15, B: .15, A: 1})
		lineHeight := float32(go16.Face().Metrics().Height.Ceil()) * 1.2
		// forcing lineHeight to an integer value will yield better looking text by preventing vertical subpixel rendering.
		lineHeight = float32(int(lineHeight))
		posY := lineHeight
		for i := 0; i < 2; i++ {
			s := wallOfText
			for len(s) > 0 {
				i := bytes.IndexByte(s, '\n')
				if i < 0 {
					break
				}
				go16.DrawBytes(b, s[:i], grog.Pt(0, posY), grog.Pt(1, 1), color.White)
				posY += lineHeight
				s = s[i+1:]
			}
		}
		dbg(b, textView, 0, textView.ScreenToWorld(mouse).String())

		// map in lower right corner
		mapView.Rect = image.Rectangle{Min: fbSz.Sub(image.Pt(200, 200)), Max: fbSz}
		b.Camera(mapView)
		b.Clear(gl.Color{R: 0, G: .5, B: 0, A: 1})
		for i := 0; i < 20; i++ {
			scale := grog.Pt(1, 1).Mul(rand.Float32() + 0.5)
			s := mapView.Size()
			b.Draw(sp0, grog.PtI(rand.Intn(s.X), rand.Intn(s.Y)), scale, rot*(rand.Float32()+.5), nil)
			b.Draw(sp1, grog.PtI(rand.Intn(s.X), rand.Intn(s.Y)), scale, rot*(rand.Float32()+.5), nil)
		}
		dbg(b, mapView, 0, mapView.ScreenToWorld(mouse).String())

		// Flush the batch in order to collect accurate-ish update statistics
		b.Flush()
		ups[ti] = float64(time.Since(ts)) / float64(time.Second)
		dbg(b, topView, 1, fmt.Sprintf("%.0f fps / %.0f ups", avg(fps[:]), avg(ups[:])))
		b.End()

		window.SwapBuffers()
		glfw.PollEvents()
		fps[ti], ti = dt, (ti+1)&(statSize-1)
	}

	// do not defer this or the program will crash with SIGSEGV (because of destroyed GL context)
	mgr.Close()
}

func avg(vs []float64) float64 {
	sum := float64(0)
	for _, v := range vs {
		sum += v
	}
	return float64(len(vs)) / sum
}

var wallOfText = []byte(`Lorem ipsum dolor sit amet, consectetur adipiscing elit. Curabitur tempus fermentum semper. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Donec aliquet odio sed lacus tincidunt, non hendrerit massa facilisis.
Donec maximus tempus sapien, quis tincidunt nunc cursus porta. Duis malesuada vestibulum sollicitudin. Morbi porta tortor ac dui porttitor pharetra. In at efficitur justo. Donec vitae nisi est. Morbi quis interdum nisi.
Fusce eu turpis tincidunt massa venenatis hendrerit.

Suspendisse potenti. In finibus tempus nibh, quis auctor nisi. Etiam eu neque mauris. Cras egestas aliquet pretium. Nulla facilisi. Suspendisse aliquet purus non purus varius cursus. In non nibh ut elit vehicula dignissim
eu eu leo. Suspendisse potenti. Cras et nisl tristique, tempus neque ac, fermentum sem.

Maecenas iaculis sem eget dui congue sodales. Interdum et malesuada fames ac ante ipsum primis in faucibus. Phasellus vulputate purus convallis magna consequat dictum. Nullam rhoncus dolor sit amet sodales convallis.
Phasellus sagittis rhoncus felis sed mattis. Praesent in dui ut lorem facilisis varius vitae non turpis. Suspendisse potenti. Suspendisse nec facilisis ligula, sit amet ultricies mi. Etiam magna turpis, dictum sit amet
efficitur eget, interdum tristique elit. Curabitur lectus nulla, vestibulum at eros ac, posuere tempor nisl. Nunc varius elit non faucibus imperdiet. Mauris in nunc posuere ligula consequat vulputate. Aliquam id est eu
ex sollicitudin convallis. Nullam eleifend mauris sed mauris efficitur molestie. In suscipit semper bibendum. In suscipit nulla a molestie lobortis.

In vulputate orci nec sem tempus viverra. Vestibulum sodales dapibus erat, in ultricies justo sagittis vel. Cras malesuada lacinia elit, cursus euismod magna finibus a. In sem mi, tincidunt a lectus sit amet, convallis
porttitor massa. Donec lorem ligula, tempor at tempor fermentum, aliquam eu ligula. Aliquam erat volutpat. Sed malesuada, velit eget lacinia finibus, lacus ante tincidunt turpis, vitae hendrerit massa tortor sed lectus.
Etiam ut egestas nunc. Nam imperdiet vitae enim id blandit.

Nulla in risus fermentum felis feugiat dignissim. Nullam luctus est mi, at tincidunt dolor bibendum eu. Nulla porta neque aliquam, dignissim lorem condimentum, aliquam massa. Pellentesque gravida, sem eget cursus
bibendum, justo nibh dignissim justo, et ultricies felis massa a justo. Nunc a diam at augue aliquet condimentum nec at purus. Vivamus eget neque sed augue sodales imperdiet. Curabitur sit amet aliquet sem, ac maximus
libero. Mauris et viverra eros. Donec gravida accumsan turpis, in maximus neque condimentum id. Suspendisse eget nibh lectus. Duis sem leo, rutrum vitae aliquam id, cursus sit amet velit. Quisque nec ultricies dui.
Pellentesque cursus diam posuere mi ullamcorper, quis condimentum quam dignissim. Cras auctor id libero nec elementum.
`)

// atlasRegion is a custom drawable that works around texture bleeding when
// using a texture atlas.
//
// See http://download.nvidia.com/developer/NVTextureSuite/Atlas_Tools/Texture_Atlas_Whitepaper.pdf
//
// The UV function computes an arbitrary "epsilon" and adjusts UV coordinates by
// ±epsilon/texture_size.
//
type atlasRegion texture.Region

// Note that atlasRegion inherits methods from the embedded *Texture field; NOT from texture.Region
// se we need to redefine these.

func (r *atlasRegion) Origin() image.Point {
	return (*texture.Region)(r).Origin()
}

func (r *atlasRegion) Size() image.Point {
	return (*texture.Region)(r).Size()
}

func (r *atlasRegion) UV() [4]float32 {
	// these value work well with the demo material. YMMV.
	// One could also use alternate methods like doubling edges.
	const epsilonX = 2. / 16 / 256
	const epsilonY = 2. / 16 / 64
	uv := (*texture.Region)(r).UV()
	uv[0] += epsilonX
	uv[1] -= epsilonY
	uv[2] -= epsilonX
	uv[3] += epsilonY
	return uv
}

func debugSystem(td *text.Drawer, fb grog.FrameBuffer) func(b grog.Drawer, v *grog.View, pos int, s string) {
	dbgView := &grog.View{Fb: fb, Scale: 1}
	return func(b grog.Drawer, v *grog.View, pos int, s string) {
		p, sz, _ := td.BoundString(s)
		sz = sz.Add(image.Pt(2, 2))
		p = p.Add(image.Pt(1, 1))
		switch pos {
		case 0:
			dbgView.Rect = image.Rectangle{Min: v.Rect.Min, Max: v.Rect.Min.Add(sz)}
		case 1:
			dbgView.Rect = image.Rect(v.Rect.Max.X-sz.X, v.Rect.Min.Y, v.Rect.Max.X, v.Rect.Min.Y+sz.Y)
		}
		b.Camera(dbgView)
		b.Clear(gl.Color{A: 1})
		td.DrawString(b, s, grog.PtPt(p), grog.Pt(1, 1), color.White)
	}
}
