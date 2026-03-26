//go:build darwin

package input

/*
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation
#include <CoreGraphics/CoreGraphics.h>

void sh_move_mouse(int x, int y) {
    CGEventRef event = CGEventCreateMouseEvent(NULL, kCGEventMouseMoved,
        CGPointMake(x, y), kCGMouseButtonLeft);
    CGEventPost(kCGHIDEventTap, event);
    CFRelease(event);
}

void sh_mouse_event(int x, int y, int eventType, int button) {
    CGEventRef event = CGEventCreateMouseEvent(NULL, eventType,
        CGPointMake(x, y), button);
    CGEventPost(kCGHIDEventTap, event);
    CFRelease(event);
}

void sh_scroll(int deltaX, int deltaY) {
    CGEventRef event = CGEventCreateScrollWheelEvent(NULL,
        kCGScrollEventUnitPixel, 2, deltaY, deltaX);
    CGEventPost(kCGHIDEventTap, event);
    CFRelease(event);
}

void sh_key_event(int keycode, int down) {
    CGEventRef event = CGEventCreateKeyboardEvent(NULL, keycode, down);
    CGEventPost(kCGHIDEventTap, event);
    CFRelease(event);
}
*/
import "C"

type darwinHandler struct{}

func newPlatformHandler() Handler {
	return &darwinHandler{}
}

func (h *darwinHandler) MoveMouse(x, y int) {
	C.sh_move_mouse(C.int(x), C.int(y))
}

func (h *darwinHandler) MouseDown(x, y int, button int) {
	eventType, cgButton := mapMouseButton(button, true)
	C.sh_mouse_event(C.int(x), C.int(y), C.int(eventType), C.int(cgButton))
}

func (h *darwinHandler) MouseUp(x, y int, button int) {
	eventType, cgButton := mapMouseButton(button, false)
	C.sh_mouse_event(C.int(x), C.int(y), C.int(eventType), C.int(cgButton))
}

func (h *darwinHandler) Scroll(x, y int, deltaX, deltaY int) {
	h.MoveMouse(x, y)
	C.sh_scroll(C.int(deltaX), C.int(deltaY))
}

func (h *darwinHandler) KeyDown(code string) {
	kc := mapKeyCodeDarwin(code)
	if kc >= 0 {
		C.sh_key_event(C.int(kc), 1)
	}
}

func (h *darwinHandler) KeyUp(code string) {
	kc := mapKeyCodeDarwin(code)
	if kc >= 0 {
		C.sh_key_event(C.int(kc), 0)
	}
}

func mapMouseButton(jsButton int, down bool) (eventType, cgButton int) {
	// CGEventType values: LeftDown=1, LeftUp=2, RightDown=3, RightUp=4, OtherDown=25, OtherUp=26
	switch jsButton {
	case 0:
		if down {
			return 1, 0
		}
		return 2, 0
	case 2:
		if down {
			return 3, 1
		}
		return 4, 1
	default:
		if down {
			return 25, 2
		}
		return 26, 2
	}
}

// mapKeyCodeDarwin maps JS KeyboardEvent.code to macOS virtual keycode.
func mapKeyCodeDarwin(code string) int {
	m := map[string]int{
		"KeyA": 0x00, "KeyB": 0x0B, "KeyC": 0x08, "KeyD": 0x02,
		"KeyE": 0x0E, "KeyF": 0x03, "KeyG": 0x05, "KeyH": 0x04,
		"KeyI": 0x22, "KeyJ": 0x26, "KeyK": 0x28, "KeyL": 0x25,
		"KeyM": 0x2E, "KeyN": 0x2D, "KeyO": 0x1F, "KeyP": 0x23,
		"KeyQ": 0x0C, "KeyR": 0x0F, "KeyS": 0x01, "KeyT": 0x11,
		"KeyU": 0x20, "KeyV": 0x09, "KeyW": 0x0D, "KeyX": 0x07,
		"KeyY": 0x10, "KeyZ": 0x06,
		"Digit0": 0x1D, "Digit1": 0x12, "Digit2": 0x13, "Digit3": 0x14,
		"Digit4": 0x15, "Digit5": 0x17, "Digit6": 0x16, "Digit7": 0x1A,
		"Digit8": 0x1C, "Digit9": 0x19,
		"Enter": 0x24, "Backspace": 0x33, "Tab": 0x30, "Escape": 0x35, "Space": 0x31,
		"Delete": 0x75, "Home": 0x73, "End": 0x77, "PageUp": 0x74, "PageDown": 0x79,
		"ArrowUp": 0x7E, "ArrowDown": 0x7D, "ArrowLeft": 0x7B, "ArrowRight": 0x7C,
		"ShiftLeft": 0x38, "ShiftRight": 0x3C,
		"ControlLeft": 0x3B, "ControlRight": 0x3E,
		"AltLeft": 0x3A, "AltRight": 0x3D,
		"MetaLeft": 0x37, "MetaRight": 0x36,
		"CapsLock": 0x39, "Insert": 0x72,
		"F1": 0x7A, "F2": 0x78, "F3": 0x63, "F4": 0x76,
		"F5": 0x60, "F6": 0x61, "F7": 0x62, "F8": 0x64,
		"F9": 0x65, "F10": 0x6D, "F11": 0x67, "F12": 0x6F,
		"Minus": 0x1B, "Equal": 0x18,
		"BracketLeft": 0x21, "BracketRight": 0x1E,
		"Backslash": 0x2A, "Semicolon": 0x29, "Quote": 0x27,
		"Backquote": 0x32, "Comma": 0x2B, "Period": 0x2F, "Slash": 0x2C,
	}
	if v, ok := m[code]; ok {
		return v
	}
	return -1
}
