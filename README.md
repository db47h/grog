# grog

grog is a fairly low-level 2D library for Go on top of OpenGL 2.1+ or OpenGLES
2.0+.

## Features

The idea is to alleviate some of the pain of using the OpenGL API while using as
few abstractions as possible and still providing full access to the OpenGL API.

- An asset manager to handle asynchronous loading textures and fonts
- Batch drawing of textures and regions. The main loop looks like:

    ```go
        // keep track of screen or framebuffer geometry
        var fb grog.Screen
        // keep frame buffer size updated in the appropriate callback
        sw, sh := window.GetFramebufferSize()
        fb.Resize(image.Pt(sw, sh))
        gl.Viewport(0, 0, sw, sh)

        // create a new concurrent batch
        b := batch.NewConcurrent()

        for !window.ShouldClose() {
            b.Begin()

            b.Camera(fb.RootView())
            b.Clear(color.RGBA{R:0,G:0,B:0,A:255})

            // sprites is defined somewhere else as var sprites []texture.Region
            for i := range sprites {
                b.Draw(&sprites[i], spritePos[i], grog.Pt(1, 1), 0, nil)
            }
            b.End()

            glfw.SwapBuffers()
            glfw.PollEvents()
        }
    ```

- Concurrent and non-concurrent batch.
- Text rendering (with very decent results).
- Support for multiple independent views with out of the box support for
  zooming/panning.
- grog is NOT tied into any OpenGL context creation toolkit like [GLFW] or
  [SDL2]. Use the one you like, roll your own event loop. See
  `cmd/demo/demo_glfw.go`.

Not really features, but worth mentioning:

- No built-in Z coordinate handling. Z-order must be managed by the client code
  (just draw in the proper order). This might end-up being implemented,
  depending on available time for trying different solutions.
- All OpenGL calls must be done from the main thread (this is required on some
  OSes).

## Demo app

Run the demo:

```bash
go run ./cmd/demo
```

This will use the OpenGL 2.1 API with GLFW for window creation and OpenGL
context handling. You can try with GLES2 API:

```bash
go run -tags "gles2" ./cmd/demo
```

Left mouse button + mouse or the arrow keys to pan the top view, mouse wheel to
zoom-in/out and escape to quit. Press space to switch to a tilemap view with 320x
320 tiles of 16x16 pixels (that's 102400 tiles).

The "ups" value in the top right corner of the screen is
1/(average_render_time). This value can be misleading: you can have 60 fps and
120 ups but with the CPU at only 10% load and its clock speed well below its
maximum. So 120 ups here doesn't mean that you can draw twice as many quads,
it's in fact much more. The actual limit is when the ups value gets very close
to the fps value.

On Linux, more precisely Ubuntu 18.04, there are a few animation hiccups when
NOT running in fullscreen mode. This is the same for all OpenGL applications. (I
suspect the compositor to silently drop frames). Just run in fullscreen if you
need smooth animations.

## Rationale

Before trying out grog, you might want to check out [pixel], [ebiten] or [engo] for
games, or [fyne] for UI apps. These engines are much more feature rich than grog.

So why another engine? The existing ones have either an API I don't like, don't
expose OpenGL, are way too heavy for my needs, have tons of external
dependencies (licensing nightmare), or a combination of those.

grog's API semantics are very close to OpenGL's (for better or worse) and mix
well with custom OpenGL code.

grog's external dependencies are limited to:

- golang.org/x/...
- github.com/pkg/errors
- github.com/golang/freetype
- some of my own repositories (which are and will remain under the same license
  as grog).

Additionally, you will need github.com/go-gl/glfw,
github.com/veandco/go-sdl2/sdl or any other toolkit capable of creating GL
contexts.

### Of OpenGL bindings, cgo and performance

The OpenGL API bindings require cgo and performance of cgo calls is notoriously
bad. This has however improved a lot and is currently around 80ns per call on a
low end CPU with Go 1.12.

This is optimized in grog by using a batch (nothing new here), available in
single threaded and concurrent versions plus custom OpenGL bindings (generated
by [gogl], NOT [go-gl]) that allow writing part of the GL code in C; like
placing the usual call sequence `glBindTexture`, `glBufferSubData` and
`glDrawElements` into a single C function; you get 3 cgo calls for the price of
one.

This last optimization is not yet enabled, and if this proves to give only minor
performance gains, we might end up using [go-gl] and/or [android-go] for the
OpenGL bindings.

In its current state, grog can display over 70000 fully animated quads
(including text rendering) at 60 fps on a low-end CPU (AMD FX6300) with the
concurrent batch, and 50000 with the non-concurrent version. For reference, a
tile map of 16x16 tiles on a 1920x1080 screen needs 8100 quads.

The concurrent batch does the model matrix transforms concurrently. This works
well as long as quads are drawn from texture atlases. If `gl.BindTexture` needs
to be called for every quad drawn (i.e. the batch is flushed after every quad),
the 60 fps limit is reached at 5200 quads for the non-concurrent batch and only
640 for the concurrent one. On a i5 6300U @2.3GHz, this same test runs at 30 fps
and 110 fps respectively. Channels are twice as fast on the i5 compared to the
FX and this clearly shows that channels are the bottleneck in this scenario.

Speaking of Go channels, there's also [gomobile/gl] where OpenGL calls go
through a worker goroutine via a channel. It has the interesting property that
code calling OpenGL functions does not need to run on the main thread. However,
considering that OpenGL is a state machine, state changes must be issued in a
specific order in order to obtain a specific result. As a consequence, the
rendering code must either use some complex sync mechanism between goroutines
that draw things, or do everything from a single goroutine (back to square one).
There are so many ways around this main-thread limitation that I don't really
see any real benefit here. Additionally, using a channel results in a 450ns
overhead per call on the same low-end CPU (half that on a i5 6300U @2.3GHz).
This doesn't bode well performance wise, but I might test it at some point.

## Supported platforms

Desktop: Anywhere you can create an use an OpenGL context with the OpenGL 2.1
API. This should cover Windows, macOS, Linux and BSDs. The gl sub-package may
not provide the proper build flags for Windows/macOS; contributions welcome!

Mobile: Android support is planned. Contributions welcome for iOS.

Only OpenGL and OpenGL ES will be supported for the time being.

## TODO

### Missing features

In no particular order:

- add and test culling in the batch
- Add optional support for OpenGLES 3.x and higher versions of OpenGL (right
  now, OpenGLES 2.0 and OpenGL 2.1 only) => this depends on [gogl]
- rotated text rendering

### Tweaks

- assets: add bulk preload functions (i.e. `PreloadTextures(names ...string)`)
- assets: the asset manager should be able to notify when something is loaded,
  at least to get textures configured and uploaded to the gpu.
- text: faster glyph cache map.
- text: Implement a custom rasterizer based on golang.org/x/image/vector?
- text: add hints/tips to package, like "for readable text, don't draw fonts at
  non-integer x/y coordinates"
- batch: reduce allocs/GC usage.

## grog is not a full fledged engine (yet)

By a full fledged engine, I mean an abstraction layer on top of SDL2 or GLFW
(plus event loop). Both have very different event handling mechanisms: one uses
an event queue while the other uses callbacks. The engine needs to somehow merge
both into a single coherent API.

There's been some work on this topic (see the app package in the engine branch)
but it quickly ran into some major issues, especially around input handling.

In a properly behaved game where the hero is moved with the WASD keys, because
users with non QWERTY keyboards should not have to configure keyboard mappings
before playing, all we care about in the event loop is the physical position
(scancode) of the keys, regardless of the user's keyboard layout (mapping of
scancodes to keycodes).

However, in the keyboard configuration UI, we want to display the keycode. So
depending on the keyboard layout, we'll display something like "Move up -> W" or
"Move Up -> Z". Same in some tutorial text: "Press Y to drop your weapon", or
"Press Z..." for a German keyboard.

This is really easy to get right with SDL2 where scancodes are a well defined
enumeration (regardless of OS), not so much with GLFW. Merging both behaviors
into a single coherent API is no small task. Also this requires GLFW 3.3 for
which Go bindings are not yet in a usable state: see go-gl/glfw#219, and
go-gl/glfw#234 that needs to be backported. So I ended up disabling GLFW support
until that PR landed, and besides some dumb init code for GL with SDL, the end
result was a useless 1:1 layer on top of the SDL API.

Since the whole thing got pointless, for the time being, the main focus is on
the core library. The demo is slowly being rewritten to decouple the backend
from the demo code so that you can just grab the backend init code to get
quickly up and running with your own application.

[GLFW]: https://github.com/go-gl/glfw
[SDL2]: https://github.com/veandco/go-sdl2
[pixel]: https://github.com/faiface/pixel
[ebiten]: https://ebiten.org
[gogl]: https://github.com/db47h/gogl
[go-gl]: https://github.com/go-gl/glow
[engo]: https://github.com/EngoEngine/engo
[fyne]: https://fyne.io/
[android-go]: https://github.com/xlab/android-go
[gomobile/gl]: https://godoc.org/golang.org/x/mobile/gl