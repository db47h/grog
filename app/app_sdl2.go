// +build sdl2

package app

import (
	"fmt"
	"image"

	"github.com/db47h/grog"
	"github.com/db47h/grog/gl"
	"github.com/veandco/go-sdl2/sdl"
)

func DriverVersion() string {
	glVer := gl.RuntimeVersion()
	var sdlVer sdl.Version
	sdl.GetVersion(&sdlVer)
	return fmt.Sprintf("SDL %d.%d.%d - %s %s %d.%d", sdlVer.Major, sdlVer.Minor, sdlVer.Patch, gl.GetGoString(gl.GL_VENDOR), glVer.API.String(), glVer.Major, glVer.Minor)
}

var drv driver = new(sdlDriver)

type sdlDriver struct {
	w *window
}

type window struct {
	fb                *grog.Screen
	sdl               *sdl.Window
	onFrameBufferSize FrameBufferSizeHandler

	setViewport bool
}

func (d *sdlDriver) init(a Interface, opts ...WindowOption) error {
	err := sdl.Init(sdl.INIT_EVERYTHING)
	if err != nil {
		panic(err)
	}

	glAPI := gl.APIVersion()
	switch glAPI.API {
	case gl.OpenGL:
		if gl.CoreProfile {
			err = sdl.GLSetAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_CORE)
		} else {
			err = sdl.GLSetAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_COMPATIBILITY)
		}
	case gl.OpenGLES:
		err = sdl.GLSetAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_ES)
	}
	if err != nil {
		return err
	}
	if err := sdl.GLSetAttribute(sdl.GL_CONTEXT_MAJOR_VERSION, glAPI.Major); err != nil {
		return err
	}
	if err := sdl.GLSetAttribute(sdl.GL_CONTEXT_MINOR_VERSION, glAPI.Minor); err != nil {
		return err
	}

	// sdl.GLSetAttribute(sdl.GL_RED_SIZE, 8);
	// sdl.GLSetAttribute(sdl.GL_GREEN_SIZE, 8);
	// sdl.GLSetAttribute(sdl.GL_BLUE_SIZE, 8);
	// sdl.GLSetAttribute(sdl.GL_ALPHA_SIZE, 8);

	if err := sdl.GLSetAttribute(sdl.GL_DOUBLEBUFFER, 1); err != nil {
		return err
	}

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

func (d *sdlDriver) createWindow(opts ...WindowOption) error {
	cfg := winCfg{title: "grog Window", x: sdl.WINDOWPOS_CENTERED, y: sdl.WINDOWPOS_CENTERED, w: 800, h: 600}
	for _, o := range opts {
		o.set(&cfg)
	}
	var flags uint32 = sdl.WINDOW_OPENGL | sdl.WINDOW_RESIZABLE
	if cfg.fullScreen {
		flags |= sdl.WINDOW_FULLSCREEN_DESKTOP
	}
	if cfg.hidden {
		flags |= sdl.WINDOW_HIDDEN
	}

	w, err := sdl.CreateWindow(cfg.title, int32(cfg.x), int32(cfg.y), int32(cfg.w), int32(cfg.h), flags)
	if err != nil {
		return err
	}

	if _, err := w.GLCreateContext(); err != nil {
		return err
	}

	gl.InitGo(sdl.GLGetProcAddress)

	ww, wh := w.GLGetDrawableSize()

	d.w = &window{
		fb:  grog.NewScreen(image.Pt(int(ww), int(wh))),
		sdl: w,
	}
	return nil
}

func (d *sdlDriver) terminate() {
	sdl.Quit()
}

func (d *sdlDriver) window() Window {
	return d.w
}

func (d *sdlDriver) pollEvents() bool {
	wid, _ := d.w.sdl.GetID()
	for {
		e := sdl.PollEvent()
		if e == nil {
			return false
		}
		switch e := e.(type) {
		case *sdl.QuitEvent:
			return true
		case *sdl.WindowEvent:
			if e.WindowID != wid {
				break
			}
			switch e.Event {
			case sdl.WINDOWEVENT_CLOSE:
				return true
			case sdl.WINDOWEVENT_RESIZED:
				ww, wh := d.w.sdl.GLGetDrawableSize()
				d.w.fb.Resize(image.Pt(int(ww), int(wh)))
				d.w.setViewport = true
				if h := d.w.onFrameBufferSize; h != nil {
					h.OnFrameBufferSize(d.w, int(ww), int(wh))
				}
			}
		}
	}
}

func (w *window) NativeHandle() interface{} {
	return w.sdl
}

func (w *window) FrameBuffer() grog.FrameBuffer {
	return w.fb
}

func (w *window) destroy() {
	_ = w.sdl.Destroy()
}

func (w *window) swapInterval(i int) {
	_ = sdl.GLSetSwapInterval(i)
}

func (w *window) swapBuffers() {
	w.sdl.GLSwap()
}

func (w *window) update() {
	if w.setViewport {
		sz := w.fb.Size()
		gl.Viewport(0, 0, int32(sz.X), int32(sz.Y))
		w.setViewport = false
	}
}
