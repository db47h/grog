package debug

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"github.com/db47h/grog"
)

const samples = 32

type Timer struct {
	times [samples]time.Duration
	index int
}

func (t *Timer) Add(dt time.Duration) {
	t.times[t.index] = dt
	t.index = (t.index + 1) & (samples - 1)
}

func (t *Timer) Average() time.Duration {
	var avg time.Duration
	for _, dt := range t.times {
		avg += dt
	}
	return avg / time.Duration(len(t.times))
}

func (t *Timer) AveragePerSecond() float64 {
	return float64(time.Second) / float64(t.Average())
}

type Pos int

const (
	TopLeft Pos = iota
	TopRight
	BottomLeft
	BottomRight
)

type Printer interface {
	Print(v *grog.View, pos Pos, s string)
	Printf(v *grog.View, pos Pos, format string, args ...interface{})
}

func NewPrinter(r grog.Renderer, td *grog.TextDrawer) Printer {
	return &printer{r, td}
}

type printer struct {
	r  grog.Renderer
	td *grog.TextDrawer
}

func (p *printer) Printf(v *grog.View, pos Pos, format string, args ...interface{}) {
	p.Print(v, pos, fmt.Sprintf(format, args...))
}

func (p *printer) Print(v *grog.View, pos Pos, s string) {
	dbgView := grog.View{Fb: v.Fb, Scale: 1}
	pt, sz, _ := p.td.BoundString(s)
	sz = sz.Add(image.Pt(2, 2))
	pt = pt.Add(image.Pt(1, 1))
	switch pos {
	case TopLeft:
		dbgView.Rect = image.Rectangle{Min: v.Rect.Min, Max: v.Rect.Min.Add(sz)}
	case TopRight:
		dbgView.Rect = image.Rect(v.Rect.Max.X-sz.X, v.Rect.Min.Y, v.Rect.Max.X, v.Rect.Min.Y+sz.Y)
	case BottomLeft:
		dbgView.Rect = image.Rect(v.Rect.Min.X, v.Rect.Max.Y-sz.Y, v.Rect.Min.X+sz.X, v.Rect.Max.Y)
	case BottomRight:
		dbgView.Rect = image.Rect(v.Rect.Max.X-sz.X, v.Rect.Max.Y-sz.Y, v.Rect.Max.X, v.Rect.Max.Y)
	}
	p.r.Camera(&dbgView)
	p.r.Clear(color.RGBA{A: 255})
	p.td.DrawString(p.r, s, grog.PtPt(pt), grog.Pt(1, 1), color.White)
}
