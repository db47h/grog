package app

import (
	"runtime"
	"time"

	"github.com/db47h/grog"
)

func init() {
	runtime.LockOSThread()
}

func Main(a Interface, opts ...WindowOption) error {
	if err := drv.init(a, opts...); err != nil {
		return err
	}
	defer drv.terminate()
	if err := a.Init(drv.window()); err != nil {
		return err
	}
	drv.run(a)
	return a.Terminate()
}

type Window interface {
	NativeHandle() interface{}
	FrameBuffer() *grog.FrameBuffer
	Destroy()
}

type driver interface {
	init(Interface, ...WindowOption) error
	terminate()
	run(Interface)
	window() Window
}

type Interface interface {
	Init(Window) error
	Terminate() error

	OnUpdate(time.Duration)
	OnDraw(Window, time.Duration)
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
