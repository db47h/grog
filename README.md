# grog

grog is a fairly low-level 2D engine for Go on top of OpenGL 3.0+ or OpenGLES
2.0+.

## Features

The idea is to alleviate some of the pain of using the OpenGL API while using as
few abstractions as possible and still providing full access to the OpenGL API.

- An asset manager to handle asynchronous loading textures and fonts
- Batch drawing of textures and regions. The main loop looks like:

    ```go
        // create a new batch
        b := batch.New()

        for !window.ShouldClose() {
            gl.Clear(gl.GL_COLOR_BUFFER_BIT)
            b.Begin()
            // screen is a grog.View that takes care of computing projection matrices
            b.SetView(screen)
            // sprites is defined somewhere else as var sprites []texture.Region
            for i := range sprites {
                b.Draw(&sprites[i], spritePos[i].X, spritePos[i].Y, 1, 1, 0, color.NRGBA{A: 255})
            }
            b.End()

            glfw.SwapBuffers()
            glfw.PollEvents()
        }
    ```

- Text rendering (with very decent results).
- Support for multiple independent views with out of the box for
  zooming/panning.

Not really features, but worth mentioning:

- NOT game oriented
- No built-in Z coordinate handling. Z-order must be managed by the client code
  (just draw in the proper order). This might end-up being implemented,
  depending on available time for trying different solutions.
- All OpenGL calls must be done from the main thread (this is not required on
  some OSes, but your code will not be portable).

### Demo app

Run the demo:

```bash
go run ./cmd/demo
```

This will use the OpenGL 3.0 API. You can try with GLES2 API:

```bash
go run -tags gles2 ./cmd/demo
```

Use the arrow keys to pan the view, mouse wheel to zoom-in/out and escape to quit.

The "ups" value in the top right corner of the screen is 1/(average_render_time)
and can be misleading: you can have 60 fps and 120 ups but with the CPU at only
10% load and its clock speed well below its maximum. So 120 ups here doesn't
mean that you can draw twice as many quads, it's in fact much more. The actual
limit is when the ups value gets very close to the fps value.

On Linux, more precisely Ubuntu 18.04, there are a few animation hiccups when
NOT running in fullscreen mode. This is the same for all OpenGL applications.
Just run in fullscreen if you need smooth animations.

### Rationale

Before trying out grog, you might want to check out [ebiten] or [engo] for
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
- github.com/go-gl/glfw
- some of my own repositories (which are and will remain under the same license
  as grog).

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
(but 100% GPU load) and 110 fps respectively. Again, channels are twice as fast
on the i5 compared to the FX and this clearly shows that channels are the
bottleneck in this scenario.

Note that I did not test much further on the i5 since its GTX 940MX GPU gets to
100% load at about 40000 quads total, regardless of using concurrent code or
not. The FX6300 tested has a GTX 1050 ti which reaches 80% load with 70000
quads.

Speaking of Go channels, there's also [gomobile/gl] where OpenGL calls go
through a worker goroutine via a channel. It has the interesting property that
code calling OpenGL functions does not need to run on the main thread. However,
considering that OpenGL is a state machine, state changes must be issued in a
specific order in order to obtain a specific result. As a consequence, the
rendering code must either use some complex sync mechanism between goroutines
drawing hud, minimap, main view and whatnot, or do everything from a single
goroutine (back to square one). There are so many ways around this main-thread
limitation that I don't really see any real benefit here. Additionally, using a
channel results in a 450ns overhead per call on the same low-end CPU (half that
on a i5 6300U @2.3GHz). This doesn't bode well performance wise, but I might
test it at some point.

### Supported platforms

Desktop: the only driver right now is GLFW, which supports Win/Mac/Linux/BSDs. I
can only test Linux, so any contributions to make it compile out of the box on
anything else is welcome.

Mobile: Android support is planned. Contributions welcome for iOS.

Only OpenGL will be supported for the time being. I may however add at least
an SDL2 driver as an alternative to GLFW on desktop.

## TODO

### Missing features

In no particular order:

- Ad a "render target" mechanism to make rendering to textures or frame buffers
  easier.
- Decouple driver specific code (i.e. GLFW) from client code. This will very
  likely take the form of a wrapper on top of the driver (meh) and something
  like `View.Update` method.
- SDL2 driver.
- add and test culling in the batch
- Add optional support for OpenGLES 3.x and higher versions of OpenGL (right
  now, OpenGLES 2.0 and OpenGL 3.0 only) => this depends on [gogl]
- rotated text rendering?

### Tweaks

- faster glyph cache map
- add hints/tips to text package: like "for readable text, don't draw fonts at non-integer x/y coordinates"
- The built-in features should require OpenGL 2.1 only (by making mipmap generation optional). Is it worth it?

[ebiten]: https://ebiten.org
[gogl]: https://github.com/db47h/gogl
[go-gl]: https://github.com/go-gl/glow
[engo]: https://github.com/EngoEngine/engo
[fyne]: https://fyne.io/
[android-go]: https://github.com/xlab/android-go
[gomobile/gl]: https://godoc.org/golang.org/x/mobile/gl