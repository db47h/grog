package app

import (
	"runtime"
	"time"

	"github.com/db47h/grog"
	"github.com/db47h/grog/app/event"
)

func init() {
	runtime.LockOSThread()
}

func Main(a Interface, opts ...WindowOption) error {
	if err := drv.init(a, opts...); err != nil {
		return err
	}
	defer drv.terminate()
	drv.window().swapInterval(1)
	if err := a.Init(drv.window()); err != nil {
		return err
	}
	run(drv, a)
	return a.Terminate()
}

// run runs the main event loop.
//
func run(d driver, a Interface) {
	// TODO: make these constants customizable
	const (
		dt = time.Second / 120
		// cap highest frame time at 1fps, slowing down the simulation if necessary
		ftHigh = time.Second
		// upper fps cap
		ftTick = time.Second / 60
		capFps = false
	)

	var (
		tPrev = time.Now()
		tAcc  time.Duration
		w     = d.window()
		t     *time.Ticker
	)

	if capFps {
		t = time.NewTicker(ftTick)
	}

	for !d.pollEvents() {
		var now time.Time
		if capFps {
			now = <-t.C
		} else {
			now = time.Now()
		}
		ft := now.Sub(tPrev)
		if ft > ftHigh {
			ft = ftHigh
		}
		tAcc += ft
		tPrev = now
		for tAcc > dt {
			a.OnUpdate(dt)
			tAcc -= dt
		}
		w.update()
		a.OnDraw(w, tAcc)
		d.window().swapBuffers()
	}
	if t != nil {
		t.Stop()
	}
}

type Window interface {
	NativeHandle() interface{}
	FrameBuffer() grog.FrameBuffer
	destroy()
	swapInterval(int)
	swapBuffers()
	update()
}

type driver interface {
	init(Interface, ...WindowOption) error
	terminate()
	window() Window
	pollEvents() (quit bool)
}

type Interface interface {
	Init(Window) error
	Terminate() error

	OnUpdate(time.Duration)
	OnDraw(Window, time.Duration)
	ProcessEvent(event.Interface) (quit bool)
}

type WindowOption interface {
	set(*winCfg)
}

type winCfg struct {
	fullScreen bool
	hidden     bool
	x, y, w, h int
	title      string
}

type winOption func(*winCfg)

func (f winOption) set(cfg *winCfg) {
	f(cfg)
}

func Title(title string) WindowOption {
	return winOption(func(cfg *winCfg) {
		cfg.title = title
	})
}

func Pos(x, y int) WindowOption {
	return winOption(func(cfg *winCfg) {
		cfg.x, cfg.y = x, y
	})
}

func Size(w, h int) WindowOption {
	return winOption(func(cfg *winCfg) {
		cfg.w, cfg.h = w, h
	})
}

func FullScreen() WindowOption {
	return winOption(func(cfg *winCfg) {
		cfg.fullScreen = true
	})
}

func Visible(b bool) WindowOption {
	return winOption(func(cfg *winCfg) {
		cfg.hidden = !b
	})
}

type FrameBufferSizeHandler interface {
	OnFrameBufferSize(w Window, width, height int)
}
