# grog

grog is a fairly low-level 2D engine for Go on top of OpenGL 3.0+ or OpenGLES
2.0+.

## Features

The idea is to alleviate some of the pain of using the OpenGL API while using as
few abstractions as possible and still providing full access to the OpenGL API.

- NOT game oriented (although)
- An asset manager to handle asynchronous loading textures and fonts
- Batch drawing of textures and regions. The main loop loos like:

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

Press the escape key to quit.

On Linux, more precisely Ubuntu 18.04, there are a few animation hiccups when
NOT running in fullscreen mode. This is the same for all OpenGL applications.
Just run in fullscreen is you need smooth animations.

### Rationale

Before trying out grog, you might want to check out [ebiten], [engo] for games
or [fyne] for UI apps. They are much more feature rich than grog.

So why another engine? The existing ones have either an API I don't like, don't
expose OpenGL, are way too heavy for my needs, have tons of external
dependencies (licensing nightmare), or a combination of those.

grog's API semantics are very close to OpenGL's (for better or worse) and mix well
with custom OpenGL code.

grog's external dependencies are limited to `golang.org/x/...`,
`github.com/pkg/errors` and some of my own repositories (which are a will remain
under the same license as grog). There are possibly some reference to
`github.com/go-gl/mathgl` but these will be removed ASAP.


### Of OpenGL bindings, cgo and performance

The OpenGL API bindings require cgo and performance of cgo calls is notoriously
bad. This has however improved a lot and is currently around 80ns per call on a
low end CPU with Go 1.12.

In its current state, grog can display over 20000 fully animated sprites plus
3000 text characters at 130 fps on a low-end CPU (AMD FX6300). For reference, a
tile map of 16x16 tiles on a 1920x1080 screen needs 8100 sprites.

This is further optimized in grog by using a batch (nothing new here) and custom
OpenGL bindings (generated by [gogl], NOT [go-gl]) that allow writing part of
the GL code in C; like putting sequences of `glBindTexture`, `glBufferSubData`,
`glDrawElements` into a single C function (you get 3 cgo calls for the price of
one).

This last optimization is not yet enabled, and if this proves to give only minor
performance gains, we might end up using [go-gl] and/or [android-go] for the
OpenGL bindings.

Speaking of [go-gl], there's also [gomobile/gl] where OpenGL calls go through a
worker goroutine via a channel. It has the interesting property that code
calling OpenGL functions does not need to run on the main thread, but using a
channel results in a 450ns overhead per call on the same low-end CPU (half that
on a i5 6300U @2.5GHz). That's a maximum of 37 OpenGL calls per frame (not
taking into account other computations like model transformation matrices).

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
- View rotation.
- Decouple driver specific code (i.e. GLFW) from client code. This will very
  likely take the form of a wrapper on top of the driver (meh) and something
  like `View.Update` method.
- SDL2 driver.
- Add optional support for OpenGLES 3.x and higher versions of OpenGL (right
  now, OpenGLES 2.0 and OpenGL 3.0 only) => this depends on [gogl]
- Desktop support: the only driver right now is GLFW, which supports
  Win/Mac/Linux/BSDs. I can only test Linux, so any contributions to make it
  compile out of the box on anything else is welcome.
- Mobile support: Android support is planned. Contributions welcome for iOS.

### Tweaks

- faster glyph cache map
- add and test culling in the batch
- add font measurement methods to Font (i.e. don't have client code import x/image/font)
- add hints/tips to text package: like "for readable text, don't draw fonts at non-integer x/y coordinates"
- assets.Delete function
- rotated text rendering?
- The built-in features should require OpenGL 2.1 only (by making mipmap generation optional). Is it worth it?

[ebiten]: https://ebiten.org
[gogl]: https://github.com/db47h/gogl
[go-gl]: https://github.com/go-gl/glow
[engo]: https://github.com/EngoEngine/engo
[fyne]: https://fyne.io/
[android-go]: https://github.com/xlab/android-go
[gomobile/gl]: https://godoc.org/golang.org/x/mobile/gl