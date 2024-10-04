//go:build windows
// +build windows

package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32                 = syscall.NewLazyDLL("kernel32.dll")
	getConsoleCursorInfoProc = kernel32.NewProc("GetConsoleCursorInfo")
	setConsoleCursorInfoProc = kernel32.NewProc("SetConsoleCursorInfo")
	procGetConsoleMode       = kernel32.NewProc("GetConsoleMode")
)

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

// cursorMoveUp moves the cursor up by a number of lines in Windows.
func cursorMoveUp(lines int) {
	if lines > 0 {
		fmt.Printf("\033[%dA", lines) // Use ANSI escape sequence for cursor movement
	}
}

// cursorMoveUp moves the cursor up by a number of lines in Windows.
func cursorMoveDown(lines int) {
	if lines > 0 {
		fmt.Printf("\033[%dB", lines) // Use ANSI escape sequence for cursor movement
	}
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
