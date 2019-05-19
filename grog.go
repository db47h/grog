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
}

// A View converts world coordinates to screen coordinates.
//
// With OpenGL the view fields map directly to the OpenGL viewport (Y axis orientated upwards).
//
// The Center point is however in world coordinates (Y axis orientated downwards).
//
//	// v is a full screen view
//	v := &grog.View{Scale: 1.0}
//
//	// draw loop
//	for {
//		batch.Begin()
//		v.CenterOn(player.X, player.Y)
//		v.Viewport(0, 0, frameBufWidth, frameBufHeight)
//		batch.SetView(v)
//		// draw commands
//		// ...
//
//		// switch to another view (to render a minimap in the bottom right corner for example)
//		mapV := &grog.View{
//			X: image.Rect(frameBufWidth-200, Y: 0,
//			W: 200, H: 200,
//			Zoom: 1.0,
//		}
//		// mapV is centered by default on the origin point (0, 0)
//		batch.SetView(mapV)
//		// draw minimap
//		// ...
//	}
//
type View struct {
	X, Y int // bounds are orientated upwards: (0, 0) is the lower left corner on the screen
	W, H int
	// World coordinates of the center point
	CenterX float32
	CenterY float32
	Scale   float32
	Angle   float32
}

type OrgPosition int

const (
	OrgUnchanged OrgPosition = iota
	OrgTopLeft
	OrgCenter
)

func (v *View) Viewport(x, y, w, h int, originPos OrgPosition) {
	v.X, v.Y, v.W, v.H = x, y, w, h
	switch originPos {
	case OrgTopLeft:
		v.CenterX, v.CenterY = float32(w)/2, float32(h)/2
	case OrgCenter:
		v.CenterX, v.CenterY = 0, 0
	}
}

func (v *View) CenterOn(x, y float32) {
	v.CenterX, v.CenterY = x, y
}

func (v *View) Rect() image.Rectangle {
	return image.Rect(v.X, v.Y, v.X+v.W, v.Y+v.H)
}

func (v *View) Size() image.Point {
	return image.Pt(v.W, v.H)
}

// ProjectionMatrix returns a 4x4 projection matrix suitable for Batch.SetProjectionMatrix.
//
func (v *View) ProjectionMatrix() [16]float32 {
	sX, sY := float32(v.W), float32(v.H)
	z2 := v.Scale * 2
	p := [16]float32{
		z2 / sX, 0, 0, 0,
		0, z2 / -sY, 0, 0,
		0, 0, -1, 0,
		-v.CenterX * z2 / sX, v.CenterY * z2 / sY, 0, 1,
	}
	if v.Angle != 0 {
		sin, cos := float32(math.Sin(float64(v.Angle))), float32(math.Cos(float64(v.Angle)))
		p[0], p[4] = p[0]*cos, p[0]*-sin
		p[1], p[5] = p[5]*sin, p[5]*cos
	}
	return p
}
