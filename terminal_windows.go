//go:build windows
// +build windows

package main

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32                   = syscall.NewLazyDLL("kernel32.dll")
	getConsoleCursorInfoProc   = kernel32.NewProc("GetConsoleCursorInfo")
	setConsoleCursorInfoProc   = kernel32.NewProc("SetConsoleCursorInfo")
	procGetConsoleMode         = kernel32.NewProc("GetConsoleMode")
	getConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
	setConsoleCursorPosition   = kernel32.NewProc("SetConsoleCursorPosition")
)

// Define the CONSOLE_SCREEN_BUFFER_INFO struct for storing console information.
type _COORD struct {
	X int16
	Y int16
}

type _CONSOLE_SCREEN_BUFFER_INFO struct {
	Size              _COORD
	CursorPosition    _COORD
	Attributes        uint16
	Window            struct{ Left, Top, Right, Bottom int16 }
	MaximumWindowSize _COORD
}

func cursorShowHide(show bool) {
	handle := getConsoleHandle()
	var info consoleCursorInfo
	getConsoleCursorInfoProc.Call(uintptr(handle), uintptr(unsafe.Pointer(&info)))

	info.visible = func() int32 {
		if show {
			return 1
		}

		return 0
	}()

	setConsoleCursorInfoProc.Call(uintptr(handle), uintptr(unsafe.Pointer(&info)))
}

// cursorHide hides the cursor in Windows.
func cursorHide() {
	cursorShowHide(false)
}

// cursorShow shows the cursor in Windows.
func cursorShow() {
	cursorShowHide(true)
}

// moveCursorVertically moves the console cursor up or down by the specified number of lines.
// Negative values move the cursor up, positive values move it down.
func moveCursorVertically(lines int) error {

	// Get the handle to the standard output (stdout).
	handle := syscall.Handle(os.Stdout.Fd())

	// Create a buffer to store console screen buffer information.
	var csbi _CONSOLE_SCREEN_BUFFER_INFO

	// Call GetConsoleScreenBufferInfo to get the current console information.
	ret, _, err := getConsoleScreenBufferInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&csbi)))
	if ret == 0 { // If the call failed, return the error.
		return err
	}

	// Move the cursor vertically (up or down) by adjusting the Y coordinate.
	newCursorPosition := csbi.CursorPosition
	newCursorPosition.Y += int16(lines)

	// Make sure the cursor doesn't move above the top of the console.
	if newCursorPosition.Y < 0 {
		newCursorPosition.Y = 0
	}

	// Also ensure the cursor doesn't move beyond the bottom of the console window.
	if newCursorPosition.Y >= csbi.Size.Y {
		newCursorPosition.Y = csbi.Size.Y - 1
	}

	// Calculate the position to move the cursor to.
	position := (uintptr(newCursorPosition.Y) << 16) | uintptr(newCursorPosition.X)

	// Call SetConsoleCursorPosition to update the cursor position.
	ret, _, err = setConsoleCursorPosition.Call(uintptr(handle), position)
	if ret == 0 { // If the call failed, return the error.
		return err
	}

	return nil
}

// cursorMoveUp moves the cursor up by a number of lines in Windows.
func cursorMoveUp(lines int) {
	if lines == 0 {
		return
	}

	moveCursorVertically(0 - lines)
}

// cursorMoveUp moves the cursor up by a number of lines in Windows.
func cursorMoveDown(lines int) {
	if lines == 0 {
		return
	}

	moveCursorVertically(lines)
}

// getConsoleHandle gets the handle for the console output.
func getConsoleHandle() syscall.Handle {
	handle := syscall.Handle(os.Stdout.Fd())
	return handle
}

func getTerminalWidth() int {
	return 80
}

// consoleCursorInfo represents the cursor info on Windows.
type consoleCursorInfo struct {
	size    uint32
	visible int32
}
