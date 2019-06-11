package main

import (
	"image"
	"log"

	"github.com/db47h/grog"
	"github.com/db47h/grog/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
)

func (a *myApp) ProcessEvents() bool {
	(*glfw.Window)(a.window).SwapBuffers()
	glfw.PollEvents()
	return (*glfw.Window)(a.window).ShouldClose()
}

type nativeWin glfw.Window

func (a *myApp) setupWindow() (err error) {
	if err := glfw.Init(); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			if a.window != nil {
				(*glfw.Window)(a.window).Destroy()
				a.window = nil
			}
			glfw.Terminate()
		}
	}()

	apiVer := gl.APIVersion()
	if apiVer.API == gl.OpenGL {
		glfw.WindowHint(glfw.ClientAPI, glfw.OpenGLAPI)
	} else {
		glfw.WindowHint(glfw.ClientAPI, glfw.OpenGLESAPI)
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
		return err
	}

	a.window = (*nativeWin)(window)

	// Init OpenGL
	window.MakeContextCurrent()
	gl.InitGo(glfw.GetProcAddress)

	glfw.SwapInterval(*vsync)
	fbSz := image.Pt(window.GetFramebufferSize())
	a.screen = grog.NewScreen(fbSz)
	gl.Viewport(0, 0, int32(fbSz.X), int32(fbSz.Y))

	log.Print("GLFW ", glfw.GetVersionString())
	ver := gl.RuntimeVersion()
	log.Printf("%s %d.%d %s", ver.API.String(), ver.Major, ver.Minor, gl.GetGoString(gl.GL_VENDOR))

	// setup callbacks
	window.SetScrollCallback(func(w *glfw.Window, xoff float64, yoff float64) {
		// save world coordinate under cursor
		v := a.topView
		p := v.ScreenToWorld(a.mouse)
		switch {
		case yoff < 0:
			v.Scale /= 1.1
		case yoff > 0:
			v.Scale *= 1.1
		}
		// move view to keep p0 under cursor
		v.Origin = v.Origin.Add(p).Sub(v.ScreenToWorld(a.mouse))
	})

	window.SetFramebufferSizeCallback(func(w *glfw.Window, width int, height int) {
		a.screen.SetSize(image.Pt(width, height))
		gl.Viewport(0, 0, int32(width), int32(height))
		a.updateViews()
	})

	window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		const scrollSpeed = 4
		if action == glfw.Repeat {
			return
		}
		if action == glfw.Release {
			switch key {
			case glfw.KeyUp, glfw.KeyW:
				a.dv.Y += scrollSpeed
			case glfw.KeyDown, glfw.KeyS:
				a.dv.Y -= scrollSpeed
			case glfw.KeyLeft, glfw.KeyA:
				a.dv.X += scrollSpeed
			case glfw.KeyRight, glfw.KeyD:
				a.dv.X -= scrollSpeed
			case glfw.KeyQ:
				a.dAngle -= 0.01
			case glfw.KeyE:
				a.dAngle += 0.01
			}
			return
		}

		switch key {
		case glfw.KeyEscape:
			w.SetShouldClose(true)
		case glfw.KeyUp, glfw.KeyW:
			a.dv.Y -= scrollSpeed
		case glfw.KeyDown, glfw.KeyS:
			a.dv.Y += scrollSpeed
		case glfw.KeyLeft, glfw.KeyA:
			a.dv.X -= scrollSpeed
		case glfw.KeyRight, glfw.KeyD:
			a.dv.X += scrollSpeed
		case glfw.KeyHome:
			a.topView.Origin = grog.Point{}
			a.topView.Scale = 1.0
			a.topView.Angle = 0
		case glfw.KeyQ:
			a.dAngle += 0.01
		case glfw.KeyE:
			a.dAngle -= 0.01
		case glfw.KeySpace:
			a.showTiles = !a.showTiles
		case glfw.Key1, glfw.KeyKP1:
			a.topView.Scale = 1
		case glfw.Key2, glfw.KeyKP2:
			a.topView.Scale = 2
		case glfw.Key3, glfw.KeyKP3:
			a.topView.Scale = 3
		case glfw.Key4, glfw.KeyKP4:
			a.topView.Scale = 4
		case glfw.Key5, glfw.KeyKP5:
			a.topView.Scale = 5
		case glfw.Key6, glfw.KeyKP6:
			a.topView.Scale = 6
		case glfw.Key7, glfw.KeyKP7:
			a.topView.Scale = 7
		case glfw.Key8, glfw.KeyKP8:
			a.topView.Scale = 8
		case glfw.KeyEqual, glfw.KeyKPAdd:
			a.topView.Scale *= 2
		case glfw.KeyMinus, glfw.KeyKPSubtract:
			a.topView.Scale /= 2
		}

	})

	window.SetMouseButtonCallback(func(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mod glfw.ModifierKey) {
		if button == glfw.MouseButton1 {
			switch action {
			case glfw.Press:
				a.mouseDrag = true
				a.mouseDragPt = a.topView.ScreenToWorld(a.mouse)
			case glfw.Release:
				a.mouseDrag = false
			}
		}
	})

	window.SetCursorPosCallback(func(w *glfw.Window, x, y float64) {
		a.mouse = grog.Pt(float32(x), float32(y))
		if a.mouseDrag {
			// set view center so that mouseDragPt is under the mouse
			a.topView.Origin = a.topView.Origin.Add(a.mouseDragPt).Sub(a.topView.ScreenToWorld(a.mouse))
		}
	})

	return nil
}

func (a *myApp) destroyWindow() {
	if a.window != nil {
		(*glfw.Window)(a.window).Destroy()
		glfw.Terminate()
	}
}
