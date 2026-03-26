//go:build windows

package input

import (
	"syscall"
	"unsafe"
)

var (
	user32         = syscall.NewLazyDLL("user32.dll")
	procSetCursorPos = user32.NewProc("SetCursorPos")
	procSendInput    = user32.NewProc("SendInput")
	procMapVirtualKey = user32.NewProc("MapVirtualKeyW")
)

const (
	inputMouse    = 0
	inputKeyboard = 1

	mousefAbsolute       = 0x8000
	mousefMove           = 0x0001
	mousefLeftDown       = 0x0002
	mousefLeftUp         = 0x0004
	mousefRightDown      = 0x0008
	mousefRightUp        = 0x0010
	mousefMiddleDown     = 0x0020
	mousefMiddleUp       = 0x0040
	mousefWheel          = 0x0800
	mousefHWheel         = 0x1000

	keyfExtendedKey = 0x0001
	keyfKeyUp       = 0x0002
	keyfScanCode    = 0x0008
)

type mouseInput struct {
	dx, dy    int32
	mouseData uint32
	flags     uint32
	time      uint32
	extraInfo uintptr
}

type keybdInput struct {
	wVk         uint16
	wScan       uint16
	dwFlags     uint32
	time        uint32
	dwExtraInfo uintptr
}

// winINPUT matches the Windows INPUT struct layout.
// Using mouseInput (the largest union member) ensures correct size and alignment.
// On x64: 4 (type) + 4 (padding) + 32 (mouseInput) = 40 bytes.
type winINPUT struct {
	inputType uint32
	mi        mouseInput
}

type windowsHandler struct{}

func newPlatformHandler() Handler {
	return &windowsHandler{}
}

func (h *windowsHandler) MoveMouse(x, y int) {
	procSetCursorPos.Call(uintptr(x), uintptr(y))
}

func (h *windowsHandler) MouseDown(x, y int, button int) {
	h.MoveMouse(x, y)
	var flags uint32
	switch button {
	case 0:
		flags = mousefLeftDown
	case 1:
		flags = mousefMiddleDown
	case 2:
		flags = mousefRightDown
	default:
		return
	}
	h.sendMouseInput(flags, 0)
}

func (h *windowsHandler) MouseUp(x, y int, button int) {
	h.MoveMouse(x, y)
	var flags uint32
	switch button {
	case 0:
		flags = mousefLeftUp
	case 1:
		flags = mousefMiddleUp
	case 2:
		flags = mousefRightUp
	default:
		return
	}
	h.sendMouseInput(flags, 0)
}

func (h *windowsHandler) Scroll(x, y int, deltaX, deltaY int) {
	h.MoveMouse(x, y)
	if deltaY != 0 {
		h.sendMouseInput(mousefWheel, uint32(deltaY))
	}
	if deltaX != 0 {
		h.sendMouseInput(mousefHWheel, uint32(deltaX))
	}
}

func (h *windowsHandler) sendMouseInput(flags, mouseData uint32) {
	var inp winINPUT
	inp.inputType = inputMouse
	inp.mi.flags = flags
	inp.mi.mouseData = mouseData
	procSendInput.Call(1, uintptr(unsafe.Pointer(&inp)), unsafe.Sizeof(inp))
}

func (h *windowsHandler) KeyDown(code string) {
	vk, extended := mapKeyCode(code)
	if vk == 0 {
		return
	}
	scan, _, _ := procMapVirtualKey.Call(uintptr(vk), 0)
	flags := uint32(keyfScanCode)
	if extended {
		flags |= keyfExtendedKey
	}
	h.sendKeyInput(uint16(vk), uint16(scan), flags)
}

func (h *windowsHandler) KeyUp(code string) {
	vk, extended := mapKeyCode(code)
	if vk == 0 {
		return
	}
	scan, _, _ := procMapVirtualKey.Call(uintptr(vk), 0)
	flags := uint32(keyfScanCode | keyfKeyUp)
	if extended {
		flags |= keyfExtendedKey
	}
	h.sendKeyInput(uint16(vk), uint16(scan), flags)
}

func (h *windowsHandler) sendKeyInput(vk, scan uint16, flags uint32) {
	var inp winINPUT
	inp.inputType = inputKeyboard
	// Reinterpret the mouseInput area as keybdInput (it fits within the larger mouseInput)
	ki := (*keybdInput)(unsafe.Pointer(&inp.mi))
	ki.wVk = vk
	ki.wScan = scan
	ki.dwFlags = flags
	procSendInput.Call(1, uintptr(unsafe.Pointer(&inp)), unsafe.Sizeof(inp))
}

// mapKeyCode maps JavaScript KeyboardEvent.code to Windows virtual key code.
// Returns (vk, isExtended).
func mapKeyCode(code string) (uint16, bool) {
	switch code {
	// Letters
	case "KeyA":
		return 0x41, false
	case "KeyB":
		return 0x42, false
	case "KeyC":
		return 0x43, false
	case "KeyD":
		return 0x44, false
	case "KeyE":
		return 0x45, false
	case "KeyF":
		return 0x46, false
	case "KeyG":
		return 0x47, false
	case "KeyH":
		return 0x48, false
	case "KeyI":
		return 0x49, false
	case "KeyJ":
		return 0x4A, false
	case "KeyK":
		return 0x4B, false
	case "KeyL":
		return 0x4C, false
	case "KeyM":
		return 0x4D, false
	case "KeyN":
		return 0x4E, false
	case "KeyO":
		return 0x4F, false
	case "KeyP":
		return 0x50, false
	case "KeyQ":
		return 0x51, false
	case "KeyR":
		return 0x52, false
	case "KeyS":
		return 0x53, false
	case "KeyT":
		return 0x54, false
	case "KeyU":
		return 0x55, false
	case "KeyV":
		return 0x56, false
	case "KeyW":
		return 0x57, false
	case "KeyX":
		return 0x58, false
	case "KeyY":
		return 0x59, false
	case "KeyZ":
		return 0x5A, false
	// Numbers
	case "Digit0":
		return 0x30, false
	case "Digit1":
		return 0x31, false
	case "Digit2":
		return 0x32, false
	case "Digit3":
		return 0x33, false
	case "Digit4":
		return 0x34, false
	case "Digit5":
		return 0x35, false
	case "Digit6":
		return 0x36, false
	case "Digit7":
		return 0x37, false
	case "Digit8":
		return 0x38, false
	case "Digit9":
		return 0x39, false
	// Special keys
	case "Enter":
		return 0x0D, false
	case "Backspace":
		return 0x08, false
	case "Tab":
		return 0x09, false
	case "Escape":
		return 0x1B, false
	case "Space":
		return 0x20, false
	case "Delete":
		return 0x2E, true
	case "Home":
		return 0x24, true
	case "End":
		return 0x23, true
	case "PageUp":
		return 0x21, true
	case "PageDown":
		return 0x22, true
	// Arrow keys
	case "ArrowUp":
		return 0x26, true
	case "ArrowDown":
		return 0x28, true
	case "ArrowLeft":
		return 0x25, true
	case "ArrowRight":
		return 0x27, true
	// Modifiers
	case "ShiftLeft":
		return 0xA0, false
	case "ShiftRight":
		return 0xA1, false
	case "ControlLeft":
		return 0xA2, false
	case "ControlRight":
		return 0xA3, true
	case "AltLeft":
		return 0xA4, false
	case "AltRight":
		return 0xA5, true
	case "MetaLeft":
		return 0x5B, true
	case "MetaRight":
		return 0x5C, true
	// F-keys
	case "F1":
		return 0x70, false
	case "F2":
		return 0x71, false
	case "F3":
		return 0x72, false
	case "F4":
		return 0x73, false
	case "F5":
		return 0x74, false
	case "F6":
		return 0x75, false
	case "F7":
		return 0x76, false
	case "F8":
		return 0x77, false
	case "F9":
		return 0x78, false
	case "F10":
		return 0x79, false
	case "F11":
		return 0x7A, false
	case "F12":
		return 0x7B, false
	// Punctuation
	case "Minus":
		return 0xBD, false
	case "Equal":
		return 0xBB, false
	case "BracketLeft":
		return 0xDB, false
	case "BracketRight":
		return 0xDD, false
	case "Backslash":
		return 0xDC, false
	case "Semicolon":
		return 0xBA, false
	case "Quote":
		return 0xDE, false
	case "Backquote":
		return 0xC0, false
	case "Comma":
		return 0xBC, false
	case "Period":
		return 0xBE, false
	case "Slash":
		return 0xBF, false
	case "CapsLock":
		return 0x14, false
	case "Insert":
		return 0x2D, true
	default:
		return 0, false
	}
}
