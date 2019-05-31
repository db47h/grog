package debug

import (
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

type Debug struct {
	TD *grog.TextDrawer
}

func (dbg *Debug) InfoBox(b grog.Renderer, v *grog.View, pos int, s string) {
	dbgView := grog.View{Fb: v.Fb, Scale: 1}
	p, sz, _ := dbg.TD.BoundString(s)
	sz = sz.Add(image.Pt(2, 2))
	p = p.Add(image.Pt(1, 1))
	switch pos {
	case 0:
		dbgView.Rect = image.Rectangle{Min: v.Rect.Min, Max: v.Rect.Min.Add(sz)}
	case 1:
		dbgView.Rect = image.Rect(v.Rect.Max.X-sz.X, v.Rect.Min.Y, v.Rect.Max.X, v.Rect.Min.Y+sz.Y)
	}
	b.Camera(&dbgView)
	b.Clear(color.RGBA{A: 255})
	dbg.TD.DrawString(b, s, grog.PtPt(p), grog.Pt(1, 1), color.White)
}
