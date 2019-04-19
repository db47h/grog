package main

import (
	"log"
	"runtime"

	//	gl "github.com/go-gl/gl/v3.1/gles2"
	// "github.com/go-gl/gl/v3.3-core/gl"

	"github.com/db47h/grog/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
)

func init() {
	// This is needed to arrange that main() runs on main thread.
	// See documentation for functions that are only allowed to be called from the main thread.
	runtime.LockOSThread()
}

func main() {
	if err := glfw.Init(); err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	// glfw.WindowHint(glfw.ContextVersionMajor, 3)
	// glfw.WindowHint(glfw.ContextVersionMinor, 3)
	// glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	// glfw.WindowHint(glfw.ClientAPI, glfw.OpenGLAPI)
	glfw.WindowHint(glfw.ClientAPI, glfw.OpenGLESAPI)
	glfw.WindowHint(glfw.ContextVersionMajor, 2)
	glfw.WindowHint(glfw.ContextVersionMinor, 0)
	glfw.WindowHint(glfw.Samples, 4)

	window, err := glfw.CreateWindow(800, 600, "Testing", nil, nil)
	if err != nil {
		panic(err)
	}

	window.MakeContextCurrent()
	if err := gl.Init(); err != nil {
		panic(err)
	}

	log.Print(glfw.GetVersionString())
	log.Print(gl.Version())

	window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		switch key {
		case glfw.KeyEscape:
			if action == glfw.Press {
				w.SetShouldClose(true)
			}
		}
	})

	window.SetFramebufferSizeCallback(func(w *glfw.Window, width int, height int) {
		gl.Viewport(0, 0, int32(width), int32(height))
	})

	for !window.ShouldClose() {
		gl.ClearColor(0.5, 0, 0, 0)
		gl.Clear(gl.COLOR_BUFFER_BIT)
		// C.doClear()
		// Do OpenGL stuff.
		window.SwapBuffers()
		glfw.PollEvents()
	}
}
