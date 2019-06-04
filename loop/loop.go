// The loop package provides a simple fixed-timestep event loop.
//
package loop

import (
	"time"
)

// EventProcessor wraps the ProcessEvents method.
//
// It is up to the implementation to either poll events or wait for events.
// Applications using a wait-for-event model should however only use the Simple
// event loop.
//
// Graphical applications that need to swap buffers should swap their buffers in
// their ProcessEvents method, before actually processing events.
//
type EventProcessor interface {
	ProcessEvents() (quit bool)
}

type FixedStepUpdater interface {
	EventProcessor
	Update(timestep time.Duration)
	Draw(frameTime, partialTimestep time.Duration)
}

// FrameStarter is the interface implemented by any App that wants the time
// stamp at the beginning of each loop iteration.
//
type FrameStarter interface {
	FrameStart(time.Time)
}

type FixedStep struct {
	ticker *time.Ticker
	minFT  time.Duration
	MaxFT  time.Duration // maximum frame time
	DT     time.Duration // timestep
}

// MinFT sets the minimum frame time.
//
// If the minFrameTime value is greater than 0, the frame rate will be clamped
// to time.Second/minFrameTime.
//
func (l *FixedStep) MinFT(minFrameTime time.Duration) {
	if minFrameTime == l.minFT {
		return
	}
	if l.ticker != nil {
		l.ticker.Stop()
		l.ticker = nil
	}
	l.minFT = minFrameTime
	if l.minFT > 0 {
		l.ticker = time.NewTicker(l.minFT)
	}
}

func (l *FixedStep) Run(a FixedStepUpdater) {
	var (
		tPrev  = time.Now()
		tAcc   time.Duration
		now    time.Time
		fStart FrameStarter
	)

	fStart, _ = a.(FrameStarter)

	if l.DT == 0 {
		l.DT = time.Second / 240
	}
	if l.MaxFT == 0 {
		l.MaxFT = time.Second
	}

	for !a.ProcessEvents() {
		if l.ticker != nil {
			now = <-l.ticker.C
		} else {
			now = time.Now()
		}
		ft := now.Sub(tPrev)
		if ft > l.MaxFT {
			ft = l.MaxFT
		}
		tAcc += ft
		tPrev = now
		if fStart != nil {
			fStart.FrameStart(now)
		}
		for dt := l.DT; tAcc >= dt; tAcc -= dt {
			a.Update(dt)
		}
		a.Draw(ft, tAcc)
	}
	if l.ticker != nil {
		l.ticker.Stop()
		l.ticker = nil
	}
}

type SimpleUpdater interface {
	EventProcessor
	Update()
	Draw()
}

// Simple provides a very simple event loop suited for applications that use a
// wait-for-event model.
//
// TODO: implement frame rate clamping.
//
type Simple struct{}

func (l Simple) Run(a SimpleUpdater) {
	fStart, _ := a.(FrameStarter)
	for !a.ProcessEvents() {
		if fStart != nil {
			fStart.FrameStart(time.Now())
		}
		a.Update()
		a.Draw()
	}
}
