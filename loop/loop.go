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

type SimpleUpdater interface {
	EventProcessor
	Update()
	Draw()
}

// Simple provides a very simple event loop suited for applications that use a
// wait-for-event model.
//
type Simple struct {
	ticker *time.Ticker
	minFT  time.Duration
}

// MinFrameTime sets the minimum frame time.
//
// If the t value is greater than 0, the frame rate will be clamped
// to time.Second/t.
//
func (l *Simple) MinFrameTime(t time.Duration) {
	if t == l.minFT {
		return
	}
	l.stopTicker()
	l.minFT = t
	if l.minFT > 0 {
		l.ticker = time.NewTicker(l.minFT)
	}
}

func (l *Simple) now() time.Time {
	if l.ticker != nil {
		return <-l.ticker.C
	}
	return time.Now()
}

func (l *Simple) stopTicker() {
	if l.ticker != nil {
		l.ticker.Stop()
		l.ticker = nil
	}
}

func (l *Simple) Run(a SimpleUpdater) {
	fStart, _ := a.(FrameStarter)
	for !a.ProcessEvents() {
		now := l.now()
		if fStart != nil {
			fStart.FrameStart(now)
		}
		a.Update()
		a.Draw()
	}
	l.stopTicker()
}

type FixedStep struct {
	Simple
	MaxFT time.Duration // maximum frame time
	DT    time.Duration // timestep
}

// Default timings for RunFixedStep
const (
	DefaultDT    time.Duration = time.Second / 240
	DefaultMaxFT time.Duration = time.Second
)

func (l *FixedStep) Run(a FixedStepUpdater) {
	var (
		tPrev  = time.Now()
		tAcc   time.Duration
		fStart FrameStarter
	)

	fStart, _ = a.(FrameStarter)

	if l.DT == 0 {
		l.DT = DefaultDT
	}
	if l.MaxFT == 0 {
		l.MaxFT = DefaultMaxFT
	}

	for !a.ProcessEvents() {
		now := l.now()
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
	l.stopTicker()
}
