//go:build linux || darwin || freebsd
// +build linux darwin freebsd

package main

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// cursorHide hides the cursor in Unix-based systems.
func cursorHide() {
	fmt.Printf("\033[?25l")
}

// cursorShow shows the cursor in Unix-based systems.
func cursorShow() {
	fmt.Printf("\033[?25h")
}

// cursorMoveUp moves the cursor up by a number of lines in Unix-based systems.
func cursorMoveUp(lines int) {
	if lines > 0 {
		fmt.Printf("\033[%dA", lines)
	}
}

// cursorMoveDown moves the cursor down by a number of lines in Unix-based systems.
func cursorMoveDown(lines int) {
	if lines > 0 {
		fmt.Printf("\033[%dB", lines)
	}
}

// Get the current terminal width
func getTerminalWidth() int {
	ws, err := unix.IoctlGetWinsize(0, unix.TIOCGWINSZ)
	if err != nil {
		// Fallback to a default width if we can't get terminal size
		return 80
	}
	return int(ws.Col)
}
