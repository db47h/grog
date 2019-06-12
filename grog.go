package grog

import (
	"image"
	"image/color"
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

type Renderer interface {
	Draw(d Drawable, dp, scale Point, rot float32, c color.Color)
	Camera(Camera)
	Clear(color.Color)
}

type BatchRenderer interface {
	Renderer
	Begin()
	Flush()
	End()
	Close()
}

type Camera interface {
	ProjectionMatrix() [16]float32
	GLRect() image.Rectangle
}
