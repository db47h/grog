package grog

import (
	"image"
)

// FrameBuffer represents a render target framebuffer.
//
type FrameBuffer interface {
	Size() image.Point
	View() *View
}

// FbToGL converts framebuffer pixel coordinates to GL coordinates in range [-1, 1].
//
func FbToGL(fb FrameBuffer, p Point) Point {
	sz := fb.Size()
	return Point{
		X: 2.0*float32(p.X)/float32(sz.X) - 1.0,
		Y: -2.0*float32(p.Y)/float32(sz.Y) + 1.0,
	}
}

// GLToFb converts GL coordinates in range [-1, 1] to framebuffer pixel coordinates.
//
func GLToFb(fb FrameBuffer, p Point) Point {
	sz := fb.Size()
	return Point{(p.X + 1) * float32(sz.X) / 2.0,
		(1 - p.Y) * float32(sz.Y) / 2.0}
}

// A Screen is a FrameBuffer implementation for a physical display screen.
//
type Screen struct {
	v View
}

// NewScreen returns a new screen of the requested size. The size should be
// updated whenever the size of the associated frame buffer changes.
//
func NewScreen(sz image.Point) *Screen {
	s := new(Screen)
	s.v = View{Fb: s, Rect: image.Rectangle{Max: sz}, Scale: 1}
	return s
}

// SetSize sets the Screen size to sz.
//
func (s *Screen) SetSize(sz image.Point) {
	s.v.Rect.Max = sz
}

// Size returns the screen size.
//
func (s *Screen) Size() image.Point {
	return s.v.Size()
}

// View returns the fullscreen view for that screen. The parent screen
// changes the view Rect whenever the Screen size is changed. Client code is
// free to adjust the view Origin, Angle and Scale.
//
func (s *Screen) View() *View {
	return &s.v
}
