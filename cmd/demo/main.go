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
	"golang.org/x/image/font"
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
		ovl      = new(ofs.Overlay)
		fbW, fbH = window.GetFramebufferSize()
		topView  = &grog.View{Scale: 1.0}
		textView = &grog.View{Scale: 1.0}
		mapView  = &grog.View{Scale: 1.0}
	)

	if err := ovl.Add(false, "assets", "cmd/demo/assets"); err != nil {
		panic(err)
	}
	assets := assets.NewManager(ovl, &assets.Config{
		TexturePath: "textures",
		FontPath:    "fonts",
	})

	window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		if action == glfw.Release {
			return
		}
		switch key {
		case glfw.KeyEscape:
			w.SetShouldClose(true)
		case glfw.KeyUp, glfw.KeyW:
			topView.CenterY += 8 / float32(topView.Scale)
		case glfw.KeyDown, glfw.KeyS:
			topView.CenterY -= 8 / float32(topView.Scale)
		case glfw.KeyRight, glfw.KeyD:
			topView.CenterX -= 8 / float32(topView.Scale)
		case glfw.KeyLeft, glfw.KeyA:
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

	window.SetScrollCallback(func(w *glfw.Window, xoff float64, yoff float64) {
		topView.Scale *= 1 + float32(yoff)/20
	})

	window.SetFramebufferSizeCallback(func(w *glfw.Window, width int, height int) {
		fbW, fbH = width, height
	})

	b, err := batch.NewConcurrent()
	if err != nil {
		panic(err)
	}
	assets.LoadTexture("box.png",
		texture.Filter(gl.GL_LINEAR_MIPMAP_LINEAR, gl.GL_LINEAR))
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

	go16, _ := assets.Font("Go-Regular.ttf", 16, text.HintingFull, texture.Nearest)
	djv16, _ := assets.Font("DejaVuSansMono.ttf", 16, text.HintingNone, texture.Nearest)

	mapBg := texture.New(16, 16)
	mapBg.SetSubImage(image.Rect(0, 0, 16, 16), image.NewUniform(color.White), image.ZP)

	// static init
	glfw.SwapInterval(1)
	gl.ClearColor(0, 0, 0.5, 1.0)

	var (
		dbgView    = &grog.View{Scale: 1}
		dbgW, dbgH int
		dbgX, dbgY int
	)
	{
		b, _ := font.BoundString(djv16.Face(), "00 fps / 00000 ups")
		dbgW, dbgH = (b.Max.X-b.Min.X).Ceil()+2, (b.Max.Y-b.Min.Y).Ceil()+2
		dbgX, dbgY = 1-b.Min.X.Floor(), 1-b.Min.Y.Floor()
	}

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
		topView.Viewport(0, fbH/2, fbW, fbH/2, grog.OrgUnchanged)
		b.SetView(topView)

		rand.Seed(424242)
		rot += float32(dt)
		for i := 0; i < 25000; i++ {
			scale := rand.Float32() + 0.5
			w, h := topView.W, topView.H
			b.Draw(sp0, float32(rand.Intn(w)-w/2), float32(rand.Intn(h)-h/2), scale, scale, rot*(rand.Float32()+.5), nil)
			b.Draw(sp1, float32(rand.Intn(w)-w/2), float32(rand.Intn(h)-h/2), scale, scale, rot*(rand.Float32()+.5), nil)
		}

		textView.Viewport(0, 0, fbW, fbH/2, grog.OrgTopLeft)
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
				go16.DrawBytes(b, 0, posY, s[:i], color.White)
				posY += lineHeight
				s = s[i+1:]
			}
		}

		// map in lower right corner
		mapView.Viewport(fbW-200, 0, 200, 200, grog.OrgTopLeft)
		b.SetView(mapView)
		b.Draw(mapBg, 0, 0, 200.0/16.0, 200.0/16.0, 0, nil)
		for i := 0; i < 20; i++ {
			scale := rand.Float32() + 0.5
			b.Draw(sp0, float32(rand.Intn(mapView.W)), float32(rand.Intn(mapView.H)), scale, scale, rot*(rand.Float32()+.5), nil)
			b.Draw(sp1, float32(rand.Intn(mapView.W)), float32(rand.Intn(mapView.H)), scale, scale, rot*(rand.Float32()+.5), nil)
		}

		// Flush the batch in order to collect accurate-ish update statistics
		b.Flush()
		ups[ti] = float64(time.Since(ts)) / float64(time.Second)

		// debug

		dbgView.Viewport(fbW-dbgW, fbH-dbgH, dbgW, dbgH, grog.OrgTopLeft)
		b.SetView(dbgView)
		b.Draw(mapBg, 0, 0, float32(dbgW)/16.0, float32(dbgH)/16.0, 0, nil)
		fups := fmt.Sprintf("%.0f fps / %.0f ups", avg(fps[:]), avg(ups[:]))
		djv16.DrawString(b, float32(dbgX), float32(dbgY), fups, color.Black)
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
