package main

import (
	"bytes"
	"flag"
	"image"
	"image/color"
	"log"
	"math"
	"math/rand"
	"runtime"
	"time"

	"github.com/db47h/grog"
	"github.com/db47h/grog/asset"
	"github.com/db47h/grog/debug"
	"github.com/db47h/grog/gl"
	"github.com/db47h/grog/loop"
	"github.com/db47h/ofs"
)

func init() {
	// This is needed to arrange that main() runs on main thread.
	runtime.LockOSThread()
}

var (
	vsync       = flag.Int("v", 1, "vsync value for glfw.SwapInterval")
	spriteCount = flag.Int("n", 20000, "`number` of sprites in the top view")
)

func main() {
	flag.Parse()

	a := &myApp{}

	if err := a.init(); err != nil {
		log.Fatal(err)
	}
	defer a.terminate()

	l := &loop.FixedStep{}
	l.Run(a)
}

type myApp struct {
	window      *nativeWin // type defined in backend specific code
	b           grog.BatchRenderer
	mgr         *asset.Manager
	screen      *grog.Screen
	mouse       grog.Point
	mouseDrag   bool
	mouseDragPt grog.Point
	dv          grog.Point
	dAngle      float32
	topView     *grog.View
	textView    *grog.View
	mapView     *grog.View
	sp          [4]grog.Region
	tiles       []grog.Region
	rot         float32
	showTiles   bool
	fps         debug.Timer
	ups         debug.Timer
	fStart      time.Time
}

func (a *myApp) init() (err error) {
	var ovl ofs.Overlay

	if err = ovl.Add(false, "assets", "cmd/demo/assets"); err != nil {
		return err
	}
	a.mgr = asset.NewManager(&ovl, asset.TexturePath("textures"), asset.FontPath("fonts"), asset.FilePath("."))

	// preload assets
	assets, _ := a.mgr.Preload([]asset.Asset{
		asset.Font("Go-Regular.ttf"),
		asset.Texture("box.png"),
		asset.Texture("tile.png"),
	}, true)

	// the tricky part with error handling is that the asset manager is created
	// first and preload started before creating the window, but it needs to be
	// closed before the window and its associated GL context are destroyed.

	if err := a.setupWindow(); err != nil {
		_ = asset.Wait(assets)
		a.mgr.Close()
		return err
	}

	defer func() {
		if err != nil {
			_ = asset.Wait(assets)
			a.mgr.Close()
			a.destroyWindow()
		}
	}()

	a.topView = &grog.View{Fb: a.screen, Scale: 1.0, OrgPos: grog.OrgCenter}
	a.textView = &grog.View{Fb: a.screen, Scale: 1.0}
	a.mapView = &grog.View{Fb: a.screen, Scale: 1.0}

	// Retrieve assets: we should have some kind of loading screen, but for the
	// demo, just waiting for assets to finish loading should be sufficient.
	if err := asset.Wait(assets); err != nil {
		return err
	}

	b, err := grog.NewBatch(true)
	if err != nil {
		return err
	}
	a.b = b

	tex0, err := a.mgr.Texture("box.png", grog.Filter(grog.Linear, grog.Nearest))
	if err != nil {
		return err
	}
	a.sp[0] = *tex0.Region(image.Rect(1, 1, 33, 33), image.Pt(16, 16))
	a.sp[1] = *tex0.Region(image.Rect(34, 1, 66, 33), image.Pt(16, 16))
	a.sp[2] = *tex0.Region(image.Rect(1, 34, 33, 66), image.Pt(16, 16))
	a.sp[3] = *tex0.Region(image.Rect(34, 34, 66, 66), image.Pt(16, 16))

	tilesAtlas, _ := a.mgr.Texture("tile.png", grog.Filter(grog.ClampToEdge, grog.ClampToEdge), grog.Filter(grog.Linear, grog.Nearest))
	for i := 0; i < 8; i++ {
		for j := 0; j < 4; j++ {
			a.tiles = append(a.tiles, *tilesAtlas.Region(image.Rect(i*16, j*16, i*16+16, j*16+16), image.Pt(8, 8)))
		}
	}

	return nil
}

func (a *myApp) terminate() {
	a.mgr.Close()
	a.destroyWindow()
}

func (a *myApp) FrameStart(t time.Time) {
	// we want to keep track of the time stamp at the beginning of the frame in order to calculate ups
	a.fStart = t
}

func (a *myApp) Update(dt time.Duration) {
	a.rot += float32(dt) / float32(time.Second)
}

func (a *myApp) Draw(ft, lag time.Duration) {
	djv, _ := a.mgr.TextDrawer("DejaVuSansMono.ttf", 16, grog.HintingNone, grog.Nearest)
	dbg := debug.NewPrinter(a.b, djv)
	a.fps.Add(ft)
	b := a.b
	b.Begin()

	/// Top view
	v := a.topView
	v.Angle += a.dAngle
	v.Pan(a.dv)
	b.Camera(v)
	b.Clear(gl.Color{R: .2, G: .2, B: .2, A: 1})

	rand.Seed(424242)
	rot := a.rot + float32(lag)/float32(time.Second)
	if a.showTiles {
		const worldSz = 320 // 320*320 = 102400 tiles
		for i := -worldSz / 2; i < worldSz/2; i++ {
			for j := -worldSz / 2; j < worldSz/2; j++ {
				// use atlasRegion instead of grog.Region
				b.Draw((*atlasRegion)(&a.tiles[rand.Intn(len(a.tiles))]), grog.PtI(i*16, j*16), grog.Pt(1, 1), 0.0, nil)
			}
		}
	} else {
		for i := 0; i < *spriteCount/4; i++ {
			scale := grog.Pt(1, 1).Mul(rand.Float32() + 0.5)
			s := a.topView.Size()
			for sp := 0; sp < 4; sp++ {
				b.Draw(&a.sp[sp], grog.PtI(rand.Intn(s.X*2)-s.X, rand.Intn(s.Y*2)-s.Y), scale, rot*(rand.Float32()+.5), nil)
			}
		}
	}
	if a.mouse.In(v.Rect) {
		dbg.Print(v, debug.TopLeft, v.ScreenToWorld(a.mouse).String())
	}

	// bottom view
	v = a.textView
	b.Camera(v)
	b.Clear(gl.Color{R: .15, G: .15, B: .15, A: 1})
	go16, _ := a.mgr.TextDrawer("Go-Regular.ttf", 16, grog.HintingFull, grog.Nearest)
	// forcing lineHeight to an integer value will yield better looking text by preventing vertical subpixel rendering.
	lineHeight := float32(math.Ceil(float64(go16.Face().Metrics().Height.Ceil()) * 1.2))
	posY := lineHeight
	maxY := float32(v.Rect.Dy())
	for i := 0; i < 2; i++ {
		s := wallOfText
		for len(s) > 0 && posY < maxY {
			i := bytes.IndexByte(s, '\n')
			if i < 0 {
				break
			}
			go16.DrawBytes(b, s[:i], grog.Pt(0, posY), grog.Pt(1, 1), color.White)
			posY += lineHeight
			s = s[i+1:]
		}
	}
	if a.mouse.In(v.Rect) {
		dbg.Print(v, debug.TopLeft, v.ScreenToWorld(a.mouse).String())
	}

	// map view in lower right corner
	v = a.mapView
	b.Camera(v)
	b.Clear(gl.Color{R: 0, G: .5, B: 0, A: 1})
	for i := 0; i < 200/4; i++ {
		scale := grog.Pt(1, 1).Mul(rand.Float32() + 0.5)
		s := v.Size()
		for sp := 0; sp < 4; sp++ {
			b.Draw(&a.sp[sp], grog.PtI(rand.Intn(s.X), rand.Intn(s.Y)), scale, rot*(rand.Float32()+.5), nil)
		}
	}
	if a.mouse.In(v.Rect) {
		dbg.Print(v, debug.TopLeft, v.ScreenToWorld(a.mouse).String())
	}

	b.Flush()
	a.ups.Add(time.Since(a.fStart))

	dbg.Printf(a.topView, debug.TopRight, "%.0f fps / %.0f ups", a.fps.AveragePerSecond(), a.ups.AveragePerSecond())

	b.End()
}

func (a *myApp) updateViews() {
	fbSz := a.screen.Size()
	a.topView.Rect.Max = image.Pt(fbSz.X, fbSz.Y/2)
	a.textView.Rect = image.Rect(0, fbSz.Y/2, fbSz.X, fbSz.Y)
	a.mapView.Rect = image.Rectangle{Min: fbSz.Sub(image.Pt(200, 200)), Max: fbSz}
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

// atlasRegion is a custom drawable that works around texture bleeding when
// using a texture atlas.
//
// See http://download.nvidia.com/developer/NVTextureSuite/Atlas_Tools/Texture_Atlas_Whitepaper.pdf
//
// The UV function computes an arbitrary "epsilon" and adjusts UV coordinates by
// ±epsilon/texture_size.
//
type atlasRegion grog.Region

// Note that atlasRegion inherits methods from the embedded *Texture field; NOT from texture.Region
// se we need to redefine these.

func (r *atlasRegion) Origin() image.Point {
	return (*grog.Region)(r).Origin()
}

func (r *atlasRegion) Size() image.Point {
	return (*grog.Region)(r).Size()
}

func (r *atlasRegion) UV() [4]float32 {
	// these value work well with the demo material. YMMV.
	// One could also use alternate methods like doubling edges.
	const epsilonX = 2. / 16 / 256
	const epsilonY = 2. / 16 / 64
	uv := (*grog.Region)(r).UV()
	uv[0] += epsilonX
	uv[1] -= epsilonY
	uv[2] -= epsilonX
	uv[3] += epsilonY
	return uv
}
