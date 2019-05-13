package grog

import (
	"image"
)

type Drawable interface {
	Origin() image.Point
	Size() image.Point
	UV() [4]float32
	NativeID() uint32
}

type View struct {
	Bounds image.Rectangle // bounds are orientated upwards: (0,0) is the lower left corner on the screen
	Origin [2]float32      // World coordinates of the top-left point
	Zoom   float32
}

func (v *View) CenterOn(x, y float32) {
	v.Origin[0] = x - float32(v.Bounds.Dx())/(2*v.Zoom)
	v.Origin[1] = y - float32(v.Bounds.Dy())/(2*v.Zoom)
}

func (v *View) ProjectionMatrix() [16]float32 {
	sX, sY := float32(v.Bounds.Dx()), float32(v.Bounds.Dy())
	z2 := v.Zoom * 2
	return [16]float32{
		z2 / sX, 0, 0, 0,
		0, -z2 / sY, 0, 0,
		0, 0, -1, 0,
		-(sX + v.Origin[0]*z2) / sX, (sY + v.Origin[1]*z2) / sY, 0, 1,
	}
}
