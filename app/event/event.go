package event

type Interface interface{}

type Quit struct{}

type WindowClose struct{}

type FrameBufferSize struct {
	Width, Height int
}

type KeyUp struct {
}

type KeyDown struct {
}
