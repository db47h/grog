package debug

import (
	"image"
	"image/color"
	"time"

	"github.com/db47h/grog"
	"github.com/db47h/grog/text"
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

func (t *Timer) PerSecond() float64 {
	return float64(time.Second) / float64(t.Average())
}

func InfoBox(b grog.Drawer, td *text.Drawer, v *grog.View, pos int, s string) {
	dbgView := grog.View{Fb: v.Fb, Scale: 1}
	p, sz, _ := td.BoundString(s)
	sz = sz.Add(image.Pt(2, 2))
	p = p.Add(image.Pt(1, 1))
	switch pos {
	case 0:
		dbgView.Viewport(v.Rect.Min.X, v.Rect.Min.Y, sz.X, sz.Y, grog.OrgTopLeft)
	case 1:
		dbgView.Viewport(v.Rect.Max.X-sz.X, v.Rect.Min.Y, sz.X, sz.Y, grog.OrgTopLeft)
	}
	b.SetView(&dbgView)
	b.Clear(color.NRGBA{A: 1})
	td.DrawString(b, s, grog.PtPt(p), grog.Pt(1, 1), color.White)
}