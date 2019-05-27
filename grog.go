package grog

import (
	"image"
	"image/color"
	"math"
)

// Drawable wraps the methods of drawable objects like texture.Texture and
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

// A View converts world coordinates to screen coordinates.
//
//	// keep track of frame buffer size in event handlers
//	s := NewScreen(fbWidth, fbHeight)
//	v := s.RootView()
//	// mapView will display a minimap in the bottom right corner
//	mv :=&grog.View{Fb: s, Scale: 1.0}
//
//	// draw loop
//	for {
//		batch.Begin()
//		v.Origin = player.Pos()
//		batch.SetView(v)
//		// draw commands
//		// ...
//
//		// switch to minimap view
//		mv.Rect = image.Rectangle{Min: s.Size().Sub(image.Pt(200,200), Max: s.Size()}
//		batch.SetView(mv)
//		// draw minimap
//		// ...
//	}
//
// FrameBuffer and View coordinates have the y axis pointing downwards (y = 0 is the top line of the screen).
//
type View struct {
	// Parent FrameBuffer
	Fb FrameBuffer
	// View rectangle in framebuffer pixel coordinates.
	Rect image.Rectangle
	// World coordinates of the point of origin of the view.
	Origin Point
	// On-screen position of the view's point of origin. View rotation and
	// scaling will preserve this property.
	OrgPos OrgPosition
	// View scaling factor.
	Scale float32
	// View angle. Due to Y axis pointing down, positive angles correspond to a
	// clockwise rotation.
	Angle float32
}

// OrgPosition determines the on-screen position of the point of origin of the view.
//
type OrgPosition int

const (
	OrgTopLeft OrgPosition = iota
	OrgCenter
)

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
	sz := v.Fb.Size()
	return image.Rect(v.Rect.Min.X, sz.Y-v.Rect.Max.Y, v.Rect.Max.X, sz.Y-v.Rect.Min.Y)
}

func (v *View) projection() (x0, y0, x1, y1, tx, ty float32) {
	// m := mgl32.Scale2D(s2/float32(sw), -s2/float32(sh))
	// m = m.Mul3(mgl32.Translate2D(float32(v.Rect.Min.X+v.Rect.Max.X-sw) / s2,
	//		float32(v.Rect.Min.Y+v.Rect.Max.Y-sh) / s2))
	// if v.Angle != 0 {
	// 	m = m.Mul3(mgl32.HomogRotate2D(v.Angle))
	// }
	// m = m.Mul3(mgl32.Translate2D(-v.Center.X, -v.Center.Y))
	// return m[0], m[3], m[1], m[4], m[6], m[7]

	var deltaO image.Point
	if v.OrgPos == OrgTopLeft {
		deltaO = v.Size()
	}

	s2 := 2. * v.Scale
	sz := v.Fb.Size()
	sw, sh := float32(sz.X), float32(sz.Y)
	x0, y1 = s2/sw, -s2/sh
	tx = float32(v.Rect.Min.X+v.Rect.Max.X-deltaO.X)/sw - 1.
	ty = -float32(v.Rect.Min.Y+v.Rect.Max.Y-deltaO.Y)/sh + 1.
	if v.Angle != 0 {
		sin, cos := float32(math.Sin(float64(v.Angle))), float32(math.Cos(float64(v.Angle)))
		x0, y0 = x0*cos, x0*-sin
		x1, y1 = y1*sin, y1*cos
	}

	tx -= x0*v.Origin.X + y0*v.Origin.Y
	ty -= x1*v.Origin.X + y1*v.Origin.Y

	return
}

// ProjectionMatrix returns a 4x4 projection matrix suitable for Batch.SetProjectionMatrix.
//
func (v *View) ProjectionMatrix() [16]float32 {
	x0, y0, x1, y1, tx, ty := v.projection()
	return [16]float32{
		x0, x1, 0, 0,
		y0, y1, 0, 0,
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

// ScreenToWorld converts screen coordinates to world coordinates.
//
func (v *View) ScreenToWorld(p Point) Point {
	g := FbToGL(v.Fb, p)
	// simplification of
	// vec := mgl32.Mat4(v.ProjectionMatrix()).Inv().Mul4x1(mgl32.Vec4{x, y, 0, 1})
	x0, y0, x1, y1, tx, ty := v.projection()
	det := x1*y0 - x0*y1
	return Point{
		X: (y1*(tx-g.X) - y0*(ty-g.Y)) / det,
		Y: (x0*(ty-g.Y) - x1*(tx-g.X)) / det,
	}
}

// WorldToScreen converts world coordinates to screen coordinates.
//
func (v *View) WorldToScreen(p Point) Point {
	x0, y0, x1, y1, tx, ty := v.projection()
	return GLToFb(v.Fb, Point{x0*p.X + y0*p.Y + tx, x1*p.X + y1*p.Y + ty})
}

// Pan pans the view by p.X, p.Y screen pixels.
//
// This is an optimized version of
//
//	v.Center = v.Center.Add(v.ScreenToWorld(p).Sub(v.ScreenToWorld(Point{})))
//
func (v *View) Pan(p Point) {
	g := FbToGL(v.Fb, p)
	x0, y0, x1, y1, _, _ := v.projection()
	det := x1*y0 - x0*y1
	v.Origin.X += (y0*(g.Y-1) - y1*(g.X+1)) / det
	v.Origin.Y += (x0*(1-g.Y) + x1*(g.X+1)) / det
}
