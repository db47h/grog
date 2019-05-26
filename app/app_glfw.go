package app

import (
	"fmt"
	"log"
	"time"

	"github.com/db47h/grog"
	"github.com/db47h/grog/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/pkg/errors"
)

func DriverVersion() string {
	ver := gl.RuntimeVersion()
	return fmt.Sprintf("GLFW %s - %s %s %d.%d", glfw.GetVersionString(), gl.GetGoString(gl.GL_VENDOR), ver.API.String(), ver.Major, ver.Minor)
}

var drv driver = new(glfwDriver)

type glfwDriver struct {
	w *window
	a Interface
}

func (d *glfwDriver) init(a Interface, opts ...WindowOption) error {
	if err := glfw.Init(); err != nil {
		return err
	}
	d.a = a

	apiVer := gl.APIVersion()
	switch apiVer.API {
	case gl.OpenGL:
		glfw.WindowHint(glfw.ClientAPI, glfw.OpenGLAPI)
	case gl.OpenGLES:
		glfw.WindowHint(glfw.ClientAPI, glfw.OpenGLESAPI)
	default:
		return errors.Errorf("Unsupported API %d", apiVer.API)
	}
	glfw.WindowHint(glfw.ContextVersionMajor, apiVer.Major)
	glfw.WindowHint(glfw.ContextVersionMinor, apiVer.Minor)
	if gl.CoreProfile {
		glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	}
	glfw.WindowHint(glfw.Samples, 4)

	if err := d.createWindow(opts...); err != nil {
		return err
	}

	// setup callbacks
	w := d.w
	if h, ok := a.(FrameBufferSizeHandler); ok {
		w.onFrameBufferSize = h
	}

	return nil
}

func (d *glfwDriver) terminate() {
	glfw.Terminate()
}

func (d *glfwDriver) createWindow(opts ...WindowOption) error {
	cfg := winCfg{title: "grog Window", x: -1, y: -1, w: 800, h: 600}
	for _, o := range opts {
		o.set(&cfg)
	}

	var (
		monitor *glfw.Monitor
		width   = cfg.w
		height  = cfg.h
	)
	if cfg.fullScreen {
		monitor = glfw.GetPrimaryMonitor()
		mode := monitor.GetVideoMode()
		glfw.WindowHint(glfw.RedBits, mode.RedBits)
		glfw.WindowHint(glfw.GreenBits, mode.GreenBits)
		glfw.WindowHint(glfw.BlueBits, mode.BlueBits)
		glfw.WindowHint(glfw.RefreshRate, mode.RefreshRate)
		width = mode.Width
		height = mode.Height
	}
	if cfg.hidden || (!cfg.fullScreen && cfg.x >= 0 && cfg.y >= 0) {
		glfw.WindowHint(glfw.Visible, glfw.False)
	} else {
		glfw.WindowHint(glfw.Visible, glfw.True)
	}
	w, err := glfw.CreateWindow(width, height, cfg.title, monitor, nil)
	if err != nil {
		return err
	}
	if !cfg.fullScreen && cfg.x >= 0 && cfg.y >= 0 {
		w.SetPos(cfg.x, cfg.y)
		if !cfg.hidden {
			w.Show()
		}
	}

	w.MakeContextCurrent()
	gl.InitGo(glfw.GetProcAddress)

	glfw.SwapInterval(1)

	fw, fh := w.GetFramebufferSize()
	d.w = &window{glfw: w, fb: grog.FrameBuffer{W: fw, H: fh}}
	w.SetFramebufferSizeCallback(d.w.glfwFrameBufferSizeCallback)

	return nil
}

// Main runs the main event loop until all windows are closed.
//
func (d *glfwDriver) run() {
	const dt = time.Second / 60
	glfw.PollEvents()
	var (
		tPrev = time.Now()
		tAcc  time.Duration
		w     = d.w
	)
	for !w.glfw.ShouldClose() {
		now := time.Now()
		ft := now.Sub(tPrev)
		tPrev = now
		tAcc += ft
		i := 0
		for tAcc > dt {
			d.a.OnUpdate(dt)
			tAcc -= dt
			i++
		}
		log.Print(i)
		if w.setViewport {
			gl.Viewport(0, 0, int32(w.fb.W), int32(w.fb.H))
			w.setViewport = false
		}
		d.a.OnDraw(w, tAcc)
		w.glfw.SwapBuffers()
		glfw.PollEvents()
	}
}

func (d *glfwDriver) window() Window {
	return d.w
}

type window struct {
	fb                grog.FrameBuffer
	glfw              *glfw.Window
	onFrameBufferSize FrameBufferSizeHandler

	setViewport bool
}

func (w *window) NativeHandle() interface{} {
	return w.glfw
}

func (w *window) FrameBuffer() *grog.FrameBuffer {
	return &w.fb
}

func (w *window) Destroy() {
	w.glfw.Destroy()
}

func (w *window) glfwFrameBufferSizeCallback(_ *glfw.Window, width int, height int) {
	w.fb.W, w.fb.H = width, height
	w.setViewport = true
	if h := w.onFrameBufferSize; h != nil {
		h.OnFrameBufferSize(w, width, height)
	}
}
