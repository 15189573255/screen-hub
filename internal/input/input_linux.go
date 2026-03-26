//go:build linux

package input

import (
	"fmt"
	"os/exec"
	"strconv"
)

type linuxHandler struct{}

func newPlatformHandler() Handler {
	return &linuxHandler{}
}

func (h *linuxHandler) MoveMouse(x, y int) {
	exec.Command("xdotool", "mousemove", strconv.Itoa(x), strconv.Itoa(y)).Run()
}

func (h *linuxHandler) MouseDown(x, y int, button int) {
	h.MoveMouse(x, y)
	exec.Command("xdotool", "mousedown", strconv.Itoa(xButton(button))).Run()
}

func (h *linuxHandler) MouseUp(x, y int, button int) {
	h.MoveMouse(x, y)
	exec.Command("xdotool", "mouseup", strconv.Itoa(xButton(button))).Run()
}

func (h *linuxHandler) Scroll(x, y int, deltaX, deltaY int) {
	h.MoveMouse(x, y)
	if deltaY > 0 {
		for i := 0; i < deltaY/120; i++ {
			exec.Command("xdotool", "click", "4").Run() // scroll up
		}
	} else if deltaY < 0 {
		for i := 0; i < -deltaY/120; i++ {
			exec.Command("xdotool", "click", "5").Run() // scroll down
		}
	}
}

func (h *linuxHandler) KeyDown(code string) {
	key := mapKeyCodeLinux(code)
	if key != "" {
		exec.Command("xdotool", "keydown", key).Run()
	}
}

func (h *linuxHandler) KeyUp(code string) {
	key := mapKeyCodeLinux(code)
	if key != "" {
		exec.Command("xdotool", "keyup", key).Run()
	}
}

func xButton(jsButton int) int {
	switch jsButton {
	case 0:
		return 1 // left
	case 1:
		return 2 // middle
	case 2:
		return 3 // right
	default:
		return 1
	}
}

func mapKeyCodeLinux(code string) string {
	m := map[string]string{
		"Enter": "Return", "Backspace": "BackSpace", "Tab": "Tab",
		"Escape": "Escape", "Space": "space", "Delete": "Delete",
		"Home": "Home", "End": "End", "PageUp": "Prior", "PageDown": "Next",
		"ArrowUp": "Up", "ArrowDown": "Down", "ArrowLeft": "Left", "ArrowRight": "Right",
		"ShiftLeft": "Shift_L", "ShiftRight": "Shift_R",
		"ControlLeft": "Control_L", "ControlRight": "Control_R",
		"AltLeft": "Alt_L", "AltRight": "Alt_R",
		"MetaLeft": "Super_L", "MetaRight": "Super_R",
		"CapsLock": "Caps_Lock", "Insert": "Insert",
		"Minus": "minus", "Equal": "equal",
		"BracketLeft": "bracketleft", "BracketRight": "bracketright",
		"Backslash": "backslash", "Semicolon": "semicolon",
		"Quote": "apostrophe", "Backquote": "grave",
		"Comma": "comma", "Period": "period", "Slash": "slash",
	}
	if v, ok := m[code]; ok {
		return v
	}
	// KeyA..KeyZ
	if len(code) == 4 && code[:3] == "Key" {
		return string(code[3] + 32) // lowercase
	}
	// Digit0..Digit9
	if len(code) == 6 && code[:5] == "Digit" {
		return string(code[5])
	}
	// F1..F12
	if code[0] == 'F' {
		if _, err := strconv.Atoi(code[1:]); err == nil {
			return code
		}
	}
	_ = fmt.Sprintf("unknown key: %s", code)
	return ""
}
