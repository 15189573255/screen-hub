package capture

import (
	"image"

	"github.com/kbinani/screenshot"
)

func CaptureScreen(displayIndex int) (*image.RGBA, error) {
	bounds := screenshot.GetDisplayBounds(displayIndex)
	return screenshot.CaptureRect(bounds)
}

func NumDisplays() int {
	return screenshot.NumActiveDisplays()
}

func DisplayBounds(displayIndex int) image.Rectangle {
	return screenshot.GetDisplayBounds(displayIndex)
}
