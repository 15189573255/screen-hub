package input

// Handler provides cross-platform mouse and keyboard simulation.
type Handler interface {
	MoveMouse(x, y int)
	MouseDown(x, y int, button int)
	MouseUp(x, y int, button int)
	Scroll(x, y int, deltaX, deltaY int)
	KeyDown(code string)
	KeyUp(code string)
}

// New returns a platform-specific input handler.
func New() Handler {
	return newPlatformHandler()
}
