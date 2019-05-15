package grog

import (
	"image"
)

// Drawable wraps the methods for drawable objects like texture.Texture and
// texture.Region.
//
type Drawable interface {
	Origin() image.Point // Point of origin
	Size() image.Point   // Drawable size
	UV() [4]float32      // UV coordinates of the drawable in the associated texture
	NativeID() uint32    // OpenGL handle of the associated texture
}

// Binder is implemented by any Drawable that needs to perform some action just after
// a batch calls gl.BindTexture (like regenerating mapmaps).
//
type Binder interface {
	OnBind()
}

// A View converts world coordinates to screen coordinates.
//
// With OpenGL the view Rectangle maps directly to the OpenGL viewport:
//
//	// v is a full screen view
//	v := &grog.View{
//		Rectangle: image.Rect(0, 0, frameBufWidth, frameBufHeight),
//		Zoom: 1.0,
//	}
//
//	// draw loop
//	for {
//		batch.Begin()
//		v.CenterOn(player.X, player.Y)
//		batch.SetView(v)
//		// draw commands
//		// ...
//
//		// switch to another view (to render a minimap in the bottom right corner for example)
//		mapV := &View{
//			Rectangle: image.Rect(frameBufWidth-200, frameBufHeight-200, frameBufWidth, frameBufHeight),
//			Zoom: 1.0,
//		}
//		batch.SetView(mapV)
//		// draw minimap
//		// ...
//	}
//
type View struct {
	image.Rectangle            // bounds are orientated upwards: (0, 0) is the lower left corner on the screen
	Origin          [2]float32 // World coordinates of the top-left point
	Zoom            float32
}

// CenterOn adjusts the view origin so that the point (x, y) in world
// coordinates will be at the center of the view. Note that the resulting Origin
// point depends on the view's Zoom value. If the view needs to be kept
// centered, this function must be called after updating Zoom.
//
func (v *View) CenterOn(x, y float32) {
	v.Origin[0] = x - float32(v.Dx())/(2*v.Zoom)
	v.Origin[1] = y - float32(v.Dy())/(2*v.Zoom)
}

// ProjectionMatrix returns a 4x4 projection matrix suitable for Batch.SetProjectionMatrix.
//
//	[ 2*zoom/width  0              0 -(width+Origin.X*2*zoom)/width  ]
//	[ 0            -2*zoom/height  0  (height+Origin.Y*2*zoom)/height]
//  [ 0             0              1  0                              ]
//  [ 0             0              0  1                              ]
//
func (v *View) ProjectionMatrix() [16]float32 {
	sX, sY := float32(v.Dx()), float32(v.Dy())
	z2 := v.Zoom * 2
	return [16]float32{
		z2 / sX, 0, 0, 0,
		0, -z2 / sY, 0, 0,
		0, 0, 1, 0,
		-(sX + v.Origin[0]*z2) / sX, (sY + v.Origin[1]*z2) / sY, 0, 1,
	}
}
