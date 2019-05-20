package main

import (
	"bytes"
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

func main() {
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

	// Load and init assets
	var (
		ovl         = new(ofs.Overlay)
		fb          grog.Screen
		mouse       image.Point
		mouseDrag   bool
		mouseDragPt [2]float32
		topView     = &grog.View{S: &fb, Scale: 1.0}
		textView    = &grog.View{S: &fb, Scale: 1.0}
		mapView     = &grog.View{S: &fb, Scale: 1.0}
	)
	fb.W, fb.H = window.GetFramebufferSize()

	if err := ovl.Add(false, "assets", "cmd/demo/assets"); err != nil {
		panic(err)
	}
	assets := assets.NewManager(ovl,
		assets.TexturePath("textures"),
		assets.FontPath("fonts"),
		assets.FilePath("."))

	window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		if action == glfw.Release {
			return
		}
		switch key {
		case glfw.KeyEscape:
			w.SetShouldClose(true)
		case glfw.KeyUp, glfw.KeyW:
			topView.CenterY -= 8 / float32(topView.Scale)
		case glfw.KeyDown, glfw.KeyS:
			topView.CenterY += 8 / float32(topView.Scale)
		case glfw.KeyLeft, glfw.KeyA:
			topView.CenterX -= 8 / float32(topView.Scale)
		case glfw.KeyRight, glfw.KeyD:
			topView.CenterX += 8 / float32(topView.Scale)
		case glfw.KeyHome:
			topView.CenterX, topView.CenterY = 0, 0
			topView.Scale = 1.0
			topView.Angle = 0
		case glfw.KeyQ:
			topView.Angle -= 0.01
		case glfw.KeyE:
			topView.Angle += 0.01
		}
	})

	window.SetMouseButtonCallback(func(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mod glfw.ModifierKey) {
		if button == glfw.MouseButton1 {
			switch action {
			case glfw.Press:
				mouseDrag = true
				x, y := topView.ViewToWorld(topView.ScreenToView(mouse))
				mouseDragPt = [2]float32{x, y}
			case glfw.Release:
				mouseDrag = false
			}
		}
	})

	window.SetScrollCallback(func(w *glfw.Window, xoff float64, yoff float64) {
		topView.Scale *= 1 + float32(yoff)/20
	})

	window.SetFramebufferSizeCallback(func(w *glfw.Window, width int, height int) {
		fb.W, fb.H = width, height
	})
	// glfw.SetCursorPosCallback
	window.SetCursorPosCallback(func(w *glfw.Window, x, y float64) {
		mouse = image.Pt(int(x), int(y))
		if mouseDrag {
			// set view center so that mouseDragPt is under the mouse
			x, y := topView.ViewToWorld(topView.ScreenToView(mouse))
			topView.CenterX += mouseDragPt[0] - x
			topView.CenterY += mouseDragPt[1] - y
		}
	})

	b, err := batch.NewConcurrent()
	if err != nil {
		panic(err)
	}
	assets.LoadTexture("box.png", texture.Filter(gl.GL_LINEAR_MIPMAP_LINEAR, gl.GL_NEAREST))
	assets.LoadTexture("text.png")
	assets.LoadFont("Go-Regular.ttf")
	assets.LoadFont("DejaVuSansMono.ttf")
	if err = assets.Wait(); err != nil {
		panic(err)
	}

	tex0, _ := assets.Texture("box.png")
	sp0 := tex0.Region(image.Rect(1, 1, 66, 66), image.Pt(32, 32))
	sp1 := sp0.Region(image.Rect(33, 33, 65, 65), image.Pt(16, 16))
	// tex1, _ := assets.Texture("text.png")
	// sp1 := tex1.Region(image.Rectangle{Min: image.Point{}, Max: tex1.Size()}, image.Pt(0, 0))

	go16, _ := assets.FontDrawer("Go-Regular.ttf", 16, text.HintingFull, texture.Nearest)
	djv16, _ := assets.FontDrawer("DejaVuSansMono.ttf", 16, text.HintingNone, texture.Nearest)

	mapBg := texture.New(16, 16)
	mapBg.SetSubImage(image.Rect(0, 0, 16, 16), image.NewUniform(color.White), image.ZP)

	// static init
	glfw.SwapInterval(1)
	gl.ClearColor(0, 0, 0.5, 1.0)

	dbgView := &grog.View{S: &fb, Scale: 1}

	const statSize = 32
	var (
		ts  = time.Now()
		fps [statSize]float64
		ups [statSize]float64
		ti          = 0
		rot float32 = 0
	)

	for !window.ShouldClose() {
		now := time.Now()
		dt := float64(now.Sub(ts)) / float64(time.Second)
		ts = now

		gl.Clear(gl.GL_COLOR_BUFFER_BIT)

		b.Begin()
		topView.Viewport(0, 0, fb.W, fb.H/2, grog.OrgUnchanged)
		b.SetView(topView)

		rand.Seed(424242)
		rot += float32(dt)
		for i := 0; i < 25000; i++ {
			scale := rand.Float32() + 0.5
			s := topView.Size()
			b.Draw(sp0, float32(rand.Intn(s.X)-s.X/2), float32(rand.Intn(s.Y)-s.Y/2), scale, scale, rot*(rand.Float32()+.5), nil)
			b.Draw(sp1, float32(rand.Intn(s.X)-s.X/2), float32(rand.Intn(s.Y)-s.Y/2), scale, scale, rot*(rand.Float32()+.5), nil)
		}

		{
			mx, my := topView.ViewToWorld(topView.ScreenToView(mouse))
			mpos := fmt.Sprintf("%.2f %.2f", mx, my)
			mp, mw, _ := djv16.BoundString(mpos)
			b.Draw(mapBg, 0, 0, float32(mw.X)/16, float32(mw.Y)/16, 0, color.Black)
			djv16.DrawString(b, mpos, float32(mp.X), float32(mp.Y), 1, 1, color.White)
		}

		textView.Viewport(0, fb.H/2, fb.W, fb.H/2, grog.OrgTopLeft)
		b.SetView(textView)
		lineHeight := float32(go16.Face().Metrics().Height.Ceil()) * 1.2
		// forcing lineHeight to an integer value will yield better looking text by preventing vertical subpixel rendering.
		lineHeight = float32(int(lineHeight))
		posY := lineHeight
		for i := 0; i < 3; i++ {
			s := wallOfText
			for len(s) > 0 {
				i := bytes.IndexByte(s, '\n')
				if i < 0 {
					break
				}
				go16.DrawBytes(b, s[:i], 0, posY, 1, 1, color.White)
				posY += lineHeight
				s = s[i+1:]
			}
		}

		// map in lower right corner
		mapView.Viewport(fb.W-200, fb.H-200, 200, 200, grog.OrgTopLeft)
		b.SetView(mapView)
		b.Draw(mapBg, 0, 0, 200.0/16.0, 200.0/16.0, 0, nil)
		for i := 0; i < 20; i++ {
			scale := rand.Float32() + 0.5
			s := mapView.Size()
			b.Draw(sp0, float32(rand.Intn(s.X)), float32(rand.Intn(s.Y)), scale, scale, rot*(rand.Float32()+.5), nil)
			b.Draw(sp1, float32(rand.Intn(s.X)), float32(rand.Intn(s.Y)), scale, scale, rot*(rand.Float32()+.5), nil)
		}

		{
			mx, my := mapView.ViewToWorld(mapView.ScreenToView(mouse))
			mpos := fmt.Sprintf("%.2f %.2f", mx, my)
			mp, mw, _ := djv16.BoundString(mpos)
			b.Draw(mapBg, 0, 0, float32(mw.X)/16, float32(mw.Y)/16, 0, color.Black)
			djv16.DrawString(b, mpos, float32(mp.X), float32(mp.Y), 1, 1, color.White)
		}

		// Flush the batch in order to collect accurate-ish update statistics
		b.Flush()
		ups[ti] = float64(time.Since(ts)) / float64(time.Second)

		// debug

		fups := fmt.Sprintf("%.0f fps / %.0f ups", avg(fps[:]), avg(ups[:]))
		// Deliberately enlarge fps text for testing purposes
		dbgPt, dbgSz, _ := djv16.BoundString(fups)
		dbgSz = dbgSz.Add(image.Pt(2, 2)).Mul(2)
		dbgPt = dbgPt.Add(image.Pt(1, 1)).Mul(2)
		dbgView.Viewport(fb.W-dbgSz.X, 0, dbgSz.X, dbgSz.Y, grog.OrgTopLeft)
		b.SetView(dbgView)
		b.Draw(mapBg, 0, 0, float32(dbgSz.X)/16.0, float32(dbgSz.Y)/16.0, 0, nil)
		djv16.DrawString(b, fups, float32(dbgPt.X), float32(dbgPt.Y), 2.0, 2.0, color.Black)
		b.End()

		window.SwapBuffers()
		glfw.PollEvents()
		fps[ti], ti = dt, (ti+1)&(statSize-1)
	}
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
