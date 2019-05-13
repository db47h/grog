package main

import (
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
		ovl          = new(ofs.Overlay)
		viewX, viewY float32
		zoom         = 1.0
		screen       = &grog.View{Bounds: image.Rectangle{image.ZP, image.Pt(window.GetFramebufferSize())}, Zoom: float32(zoom)}
	)

	if err := ovl.Add(false, "assets", "cmd/demo/assets"); err != nil {
		panic(err)
	}
	assets := assets.NewManager(ovl)

	window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		if action == glfw.Release {
			return
		}
		switch key {
		case glfw.KeyEscape:
			w.SetShouldClose(true)
		case glfw.KeyUp:
			viewY -= 4 / float32(zoom)
		case glfw.KeyDown:
			viewY += 4 / float32(zoom)
		case glfw.KeyRight:
			viewX += 4 / float32(zoom)
		case glfw.KeyLeft:
			viewX -= 4 / float32(zoom)
		case glfw.KeyHome:
			zoom = 1.0
			viewX, viewY = 0, 0
		}
	})

	window.SetScrollCallback(func(w *glfw.Window, xoff float64, yoff float64) {
		zoom *= 1 + yoff/20
	})

	window.SetFramebufferSizeCallback(func(w *glfw.Window, width int, height int) {
		screen.Bounds.Max = image.Pt(width, height)
		gl.Viewport(0, 0, int32(width), int32(height))
	})

	b, err := batch.New()
	if err != nil {
		panic(err)
	}
	assets.LoadTexture("textures/box.png",
		texture.Filter(gl.GL_LINEAR_MIPMAP_LINEAR, gl.GL_LINEAR))
	tex0, err := assets.Texture("textures/box.png")
	if err != nil {
		panic(err)
	}
	sp0 := tex0.Region(image.Rect(1, 1, 66, 66), image.Pt(32, 32))
	sp1 := sp0.Region(image.Rect(33, 33, 65, 65), image.Pt(16, 16))

	assets.LoadFont("fonts/Go-Regular.ttf")
	gofont, err := assets.Font("fonts/Go-Regular.ttf")
	if err != nil {
		panic(err)
	}
	go26, err := gofont.NewFace(36, font.HintingNone)
	if err != nil {
		panic(err)
	}
	tex1 := texture.New(text.TextImage(go26, "Hello, Woyrld!"))

	// static init
	gl.ClearColor(0, 0, 0.5, 1.0)
	gl.Viewport(int32(screen.Bounds.Min.X), int32(screen.Bounds.Min.Y), int32(screen.Bounds.Max.X), int32(screen.Bounds.Max.Y))

	var (
		ts  = time.Now()
		fps [64]float64
		ups [64]float64
		ti          = 0
		rot float32 = 0
	)

	for !window.ShouldClose() {
		now := time.Now()
		dt := float64(now.Sub(ts)) / float64(time.Second)
		ts = now

		gl.Clear(gl.GL_COLOR_BUFFER_BIT)

		screen.Zoom = float32(zoom)
		screen.CenterOn(viewX, viewY)

		b.Begin()
		b.SetProjectionMatrix(screen.ProjectionMatrix())
		rand.Seed(424242)
		rot += float32(dt)
		for i := 0; i < 10000; i++ {
			scale := rand.Float32() + 0.5
			b.Draw(sp0, float32(rand.Intn(screen.Bounds.Dx())-screen.Bounds.Dx()/2), float32(rand.Intn(screen.Bounds.Dy())-screen.Bounds.Dy()/2), scale, scale, rot*(rand.Float32()+.5), color.NRGBA{255, 255, 255, 255})
			b.Draw(sp1, float32(rand.Intn(screen.Bounds.Dx())-screen.Bounds.Dx()/2), float32(rand.Intn(screen.Bounds.Dy())-screen.Bounds.Dy()/2), scale, scale, rot*(rand.Float32()+.5), color.NRGBA{255, 255, 255, 255})
		}
		b.Draw(tex1, 0, 0, 1, 1, 0, color.NRGBA{0, 0, 0, 255})
		// rot += float32(dt / 2)
		// b.Draw(sp0, 0, -40, 1.0, 1.0, rot, color.NRGBA{255, 255, 255, 255})
		// b.Draw(sp1, 0, 0, 1.0, 1.0, 0, color.NRGBA{0, 0, 0, 255})

		b.End()
		ups[ti] = float64(time.Since(ts)) / float64(time.Second)

		window.SwapBuffers()
		glfw.PollEvents()
		fps[ti], ti = dt, (ti+1)&63
	}
	var sum float64
	for _, dt := range fps {
		sum += dt
	}
	log.Printf("%v fps", float64(len(fps))/sum)
	sum = 0
	for _, dt := range ups {
		sum += dt
	}
	log.Printf("%v ups", float64(len(ups))/sum)
}
