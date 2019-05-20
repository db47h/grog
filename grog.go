package grog

import (
	"image"
	"image/color"
	"math"
)

// Drawable wraps the methods for drawable objects like texture.Texture and
// texture.Region.
//
type Drawable interface {
	Bind()               // Bind calls gl.BindTexture(gl.GL_TEXTURE_2D, ...)
	Origin() image.Point // Point of origin
	Size() image.Point   // Drawable size
	UV() [4]float32      // UV coordinates of the drawable in the associated texture
	NativeID() uint32    // OpenGL handle of the associated texture
}

type Drawer interface {
	Draw(d Drawable, x, y, scaleX, scaleY, rot float32, c color.Color)
	SetView(v *View)
}

// Screen keeps track of a screen's width and height.
//
type Screen struct {
	W, H int
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
	CenterX float32
	CenterY float32
	Scale   float32
	Angle   float32
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
		v.CenterX, v.CenterY = float32(w)/2, float32(h)/2
	case OrgCenter:
		v.CenterX, v.CenterY = 0, 0
	}
}

// CenterOn is a shorthand for v.CenterX, v.CenterY = x, y
//
func (v *View) CenterOn(x, y float32) {
	v.CenterX, v.CenterY = x, y
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
	sX, sY := float32(v.Rect.Dx()), float32(v.Rect.Dy())
	z2 := v.Scale * 2
	x0 = z2 / sX
	y0 = z2 / -sY
	tx = -v.CenterX * z2 / sX
	ty = v.CenterY * z2 / sY
	if v.Angle != 0 {
		sin, cos := float32(math.Sin(float64(v.Angle))), float32(math.Cos(float64(v.Angle)))
		x0, y1 = x0*cos, x0*-sin
		x1, y0 = y0*sin, y0*cos
	}
	return

}

// ProjectionMatrix returns a 4x4 projection matrix suitable for Batch.SetProjectionMatrix.
//
func (v *View) ProjectionMatrix() [16]float32 {
	x0, y0, x1, y1, tx, ty := v.projection()
	return [16]float32{
		x0, x1, 0, 0,
		y1, y0, 0, 0,
		0, 0, -1, 0,
		tx, ty, 0, 1,
	}
}

// ScreenToView converts screen coordinates to view coordinates.
// It is equivalent to p.Sub(v.Rect.Min).
//
func (v *View) ScreenToView(p image.Point) image.Point {
	return p.Sub(v.Rect.Min)
}

// ViewToGL converts view coordinates to GL coordinates.
// v.Rect.Min maps to (-1, -1) and v.Rect.Max maps to (1,1)
//
func (v *View) ViewToGL(p image.Point) (x, y float32) {
	return 2.0*float32(p.X)/float32(v.Rect.Dx()) - 1.0,
		-2.0*float32(p.Y)/float32(v.Rect.Dy()) + 1.0
}

// ViewToWorld converts view coordinates to world coordinates.
//
// The following example shows how to get the world coordinate under the mouse
// cursor:
//
//	wX, wY := v.ViewToWorld(v.ScreenToView(image.Pt(mouseX, mouseY)))
//
func (v *View) ViewToWorld(p image.Point) (wX, wY float32) {
	x, y := v.ViewToGL(p)
	// simplification of
	// vec := mgl32.Mat4(v.ProjectionMatrix()).Inv().Mul4x1(mgl32.Vec4{x, y, 0, 1})
	x0, y0, x1, y1, tx, ty := v.projection()
	det := x1*y1 - x0*y0
	wX = (y0*(tx-x) - y1*(ty-y)) / det
	wY = (x0*(ty-y) - x1*(tx-x)) / det
	return
}
