package grog

import (
	"image"
	"image/color"

	"github.com/go-gl/mathgl/mgl32"
)

// Drawable wraps the methods for drawable objects like texture.Texture and
// texture.Region.
//
type Drawable interface {
	Bind()               // Bind calls gl.BindTexture(gl.GL_TEXTURE_2D, ...)
	NativeID() uint32    // OpenGL handle of the associated texture
	Origin() image.Point // Point of origin
	Size() image.Point   // Drawable size
	UV() [4]float32      // UV coordinates of the drawable in the associated texture
}

type Drawer interface {
	Draw(d Drawable, dp, scale Point, rot float32, c color.Color)
	SetView(v *View)
	Clear(color.Color)
}

// Screen keeps track of a screen's width and height.
//
type Screen struct {
	W, H int
}

// ScreenToGL converts screen coordinates to GL coordinates in range [-1, 1].
//
func (s *Screen) ScreenToGL(p Point) Point {
	return Point{
		X: 2.0*float32(p.X)/float32(s.W) - 1.0,
		Y: -2.0*float32(p.Y)/float32(s.H) + 1.0,
	}
}

// GLToScreen converts GL coordinates in range [-1, 1] to screen coordinates.
//
func (s *Screen) GLToScreen(p Point) Point {
	return Point{(p.X + 1) * float32(s.W) / 2.0,
		(1 - p.Y) * float32(s.H) / 2.0}
}

// A View converts world coordinates to screen coordinates.
//
// The Center point is however in world coordinates (Y axis orientated downwards).
//
//	// keep track of frame buffer size in event handlers
//	fb := Screen{fbWidth, fbHeight}
//	// v is a full screen view
//	v := &grog.View{S: fb, Scale: 1.0}
//	// mapView will display a minimap in the bottom right corner
//	mv := &grog.View{S: &fb, Scale: 1.0}
//
//	// draw loop
//	for {
//		batch.Begin()
//		v.CenterOn(player.X, player.Y)
//		v.Viewport(0, 0, fb.W, fb.H, grog.OrgUnchanged)
//		batch.SetView(v)
//		// draw commands
//		// ...
//
//		// switch to minimap view with point (0,0) in the top left corner
//		mv.Viewport(fb.W-200, fb.H-200, 200, 200, grog.OrgTopLeft)
//		batch.SetView(mv)
//		// draw minimap
//		// ...
//	}
//
// Screen and View coordinates have the y axis pointing downwards (y = 0 is the top line of the screen).
//
type View struct {
	S *Screen
	// View rectangle in screen coordinates
	Rect image.Rectangle
	// World coordinates of the center point
	Center Point
	Scale  float32
	Angle  float32
}

// OrgPosition sets the position of the point of origin (world coordinates 0, 0)
// when calling view.Viewport.
//
type OrgPosition int

const (
	OrgUnchanged OrgPosition = iota
	OrgTopLeft
	OrgCenter
)

// Viewport sets the view position and size. The originPos parameter determines if the position of the point of origin needs to be adjusted:
//
//	OrgUnchanged: do not change the view's CenterX/Y.
//	OrgTopLeft  : update CenterX/Y so that the point with world coordinates 0,0 is aligned with the top left of the view.
//	OrgCenter   : set CenterX/Y to 0, 0. The point with world coordinates 0,0 will be at the center of the view.
//
func (v *View) Viewport(x, y, w, h int, originPos OrgPosition) {
	v.Rect = image.Rect(x, y, x+w, y+h)
	switch originPos {
	case OrgTopLeft:
		v.Center = Pt(float32(w)/2, float32(h)/2)
	case OrgCenter:
		v.Center = Point{}
	}
}

// Size returns the view size in pixels.
//
func (v *View) Size() image.Point {
	return v.Rect.Size()
}

// GLRect returns an image.Rectangle for gl.Viewport (y axis upwards, 0,0 in the bottom left corner).
//
//	r := v.GLRect()
//	gl.Viewport(int32(r.Min.X), int32(r.Min.Y), int32(r.Dx()), int32(r.Dy()))
//
func (v *View) GLRect() image.Rectangle {
	r := v.Rect
	r.Min.Y, r.Max.Y = v.S.H-r.Max.Y, v.S.H-r.Min.Y
	return r
}

func (v *View) projection() (x0, y0, x1, y1, tx, ty float32) {
	s2 := 2. * v.Scale
	// t := Point{float32(v.Rect.Min.X+v.Rect.Max.X-v.S.W) / s2,
	// 	float32(v.Rect.Min.Y+v.Rect.Max.Y-v.S.H) / s2}

	t := Point{float32(v.Rect.Min.X+v.Rect.Max.X-v.S.W) / s2,
		float32(v.Rect.Min.Y+v.Rect.Max.Y-v.S.H) / s2}

	m := mgl32.Scale2D(s2/float32(v.S.W), -s2/float32(v.S.H))
	m = m.Mul3(mgl32.Translate2D(t.X, t.Y))
	if v.Angle != 0 {
		m = m.Mul3(mgl32.HomogRotate2D(v.Angle))
	}
	m = m.Mul3(mgl32.Translate2D(-v.Center.X, -v.Center.Y))
	return m[0], m[4], m[1], m[3], m[6], m[7]

	// m := mgl32.Scale2D(2.*v.Scale/float32(v.Rect.Dx()), -2.*v.Scale/float32(v.Rect.Dy()))
	// if v.Angle != 0 {
	// 	// don't translate before and after rotate, we rotate around the view center
	// 	m = m.Mul3(mgl32.HomogRotate2D(v.Angle))
	// }
	// m = m.Mul3(mgl32.Translate2D(-v.Center.X, -v.Center.Y))
	// return m[0], m[4], m[1], m[3], m[6], m[7]

	// sX, sY := float32(v.Rect.Dx()), float32(v.Rect.Dy())
	// z2 := v.Scale * 2
	// x0 = z2 / sX
	// y0 = -z2 / sY
	// if v.Angle != 0 {
	// 	sin, cos := float32(math.Sin(float64(v.Angle))), float32(math.Cos(float64(v.Angle)))
	// 	x0, y1 = x0*cos, x0*-sin
	// 	x1, y0 = y0*sin, y0*cos
	// }
	// tx, ty = x0*-v.Center.X+y1*-v.Center.Y, x1*-v.Center.X+y0*-v.Center.Y
	// return
}

// ProjectionMatrix returns a 4x4 projection matrix suitable for Batch.SetProjectionMatrix.
//
func (v *View) ProjectionMatrix() [16]float32 {
	x0, y0, x1, y1, tx, ty := v.projection()
	return [16]float32{
		x0, x1, 0, 0,
		y1, y0, 0, 0,
		0, 0, 1, 0,
		tx, ty, 0, 1,
	}
}

// ViewToScreen converts view coordinates to screen coordinates.
// It is equivalent to p.Add(v.Rect.Min).
//
func (v *View) ViewToScreen(p Point) Point {
	return p.Add(PtPt(v.Rect.Min))
}

// ScreenToView converts screen coordinates to view coordinates.
// It is equivalent to p.Sub(v.Rect.Min).
//
func (v *View) ScreenToView(p Point) Point {
	return p.Sub(PtPt(v.Rect.Min))
}

// ViewToWorld converts view coordinates to world coordinates.
//
func (v *View) ViewToWorld(p Point) Point {
	return v.ScreenToWorld(v.ViewToScreen(p))
}

// WorldToView converts world coordinates to view coordinates.
//
func (v *View) WorldToView(p Point) Point {
	return v.ScreenToView(v.WorldToScreen(p))
}

// ScreenToWorld is a shorthand for
//
//	v.ViewToWorld(v.ScreenToView(p))
//
func (v *View) ScreenToWorld(p Point) Point {
	g := v.S.ScreenToGL(p)
	// simplification of
	// vec := mgl32.Mat4(v.ProjectionMatrix()).Inv().Mul4x1(mgl32.Vec4{x, y, 0, 1})
	x0, y0, x1, y1, tx, ty := v.projection()
	det := x1*y1 - x0*y0
	return Point{
		X: (y0*(tx-g.X) - y1*(ty-g.Y)) / det,
		Y: (x0*(ty-g.Y) - x1*(tx-g.X)) / det,
	}
}

// WorldToScreen is a shorthand for
//
//	v.ViewToScreen(v.WorldToView(p))
//
func (v *View) WorldToScreen(p Point) Point {
	x0, y0, x1, y1, tx, ty := v.projection()
	return v.S.GLToScreen(Point{x0*p.X + y1*p.Y + tx, x1*p.X + y0*p.Y + ty})
}

// Pan pans the view by p.X, p.Y screen pixels.
//
// This is an optimized version of
//
//	v.Center = v.Center.Add(v.ViewToWorld(p).Sub(v.ViewToWorld(image.ZP)))
//
func (v *View) Pan(p Point) {
	g := v.S.ScreenToGL(p)
	x0, y0, x1, y1, _, _ := v.projection()
	det := x1*y1 - x0*y0
	v.Center.X += (y1*(g.Y-1) - y0*(g.X+1)) / det
	v.Center.Y += (x0*(1-g.Y) + x1*(g.X+1)) / det
}
