// +build ignore

package app

import (
	"fmt"
	"image"
	"log"

	"github.com/db47h/grog"
	"github.com/db47h/grog/app/event"
	"github.com/db47h/grog/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/pkg/errors"
)

// Driver version returns the app driver version string.
// It may not work before calling add.Main.
//
func DriverVersion() string {
	ver := gl.RuntimeVersion()
	return fmt.Sprintf("GLFW %s - %s %s %d.%d", glfw.GetVersionString(), gl.GetGoString(gl.GL_VENDOR), ver.API.String(), ver.Major, ver.Minor)
}

var drv driver = new(glfwDriver)

type glfwDriver struct {
	w *window
}

type window struct {
	app  Interface
	fb   *grog.Screen
	glfw *glfw.Window

	quit        bool
	setViewport bool
}

func (d *glfwDriver) init(a Interface, opts ...WindowOption) error {
	if !glfw.Init() {
		return glfw.GetError()
	}

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

	d.w.app = a
	w := d.w
	// setup callbacks
	w.glfw.SetFramebufferSizeCallback(w.onFrameBufferSize)
	w.glfw.SetKeyCallback(w.onKey)
	w.glfw.SetCloseCallback(w.onClose)

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
	w := glfw.CreateWindow(width, height, cfg.title, monitor, nil)
	if w == nil {
		return glfw.GetError()
	}
	if !cfg.fullScreen && cfg.x >= 0 && cfg.y >= 0 {
		w.SetPos(cfg.x, cfg.y)
		if !cfg.hidden {
			w.Show()
		}
	}

	w.MakeContextCurrent()
	gl.InitGo(glfw.GetProcAddress)

	d.w = &window{glfw: w, fb: grog.NewScreen(image.Pt(w.GetFramebufferSize()))}

	return nil
}

func (d *glfwDriver) pollEvents() bool {
	glfw.PollEvents()
	return d.w.glfw.ShouldClose() || d.w.quit
}

func (d *glfwDriver) window() Window {
	return d.w
}

func (w *window) NativeHandle() interface{} {
	return w.glfw
}

func (w *window) FrameBuffer() grog.FrameBuffer {
	return w.fb
}

func (w *window) destroy() {
	w.glfw.Destroy()
}

func (w *window) swapInterval(i int) {
	glfw.SwapInterval(i)
}

func (w *window) swapBuffers() {
	w.glfw.SwapBuffers()
}

func (w *window) update() {
	if w.setViewport {
		sz := w.fb.Size()
		gl.Viewport(0, 0, int32(sz.X), int32(sz.Y))
		w.setViewport = false
	}
}

func (w *window) onFrameBufferSize(_ *glfw.Window, width int, height int) {
	w.fb.Resize(image.Pt(width, height))
	w.setViewport = true
	w.quit = w.app.ProcessEvent(event.FrameBufferSize{Width: width, Height: height})
}

func (w *window) onKey(_ *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	if action == glfw.Press {
		log.Printf("%d %s %q", scancode, Key(key), glfw.GetKeyName(key, scancode))
	}
}

func (w *window) onClose(_ *glfw.Window) {
	w.quit = w.app.ProcessEvent(event.WindowClose{})
}
