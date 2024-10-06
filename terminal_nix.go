//go:build linux
// +build linux

package yanprogress

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

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

// GetCursorPosition retrieves the cursor position
func GetCursorPosition() (int, int, error) {
	// Switch terminal to raw mode to read response directly
	fd := int(os.Stdin.Fd())
	var oldState syscall.Termios
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), syscall.TCGETS, uintptr(unsafe.Pointer(&oldState)), 0, 0, 0); err != 0 {
		return 0, 0, fmt.Errorf("failed to get terminal attributes: %v", err)
	}

	newState := oldState
	newState.Lflag &^= syscall.ICANON | syscall.ECHO
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), syscall.TCSETS, uintptr(unsafe.Pointer(&newState)), 0, 0, 0); err != 0 {
		return 0, 0, fmt.Errorf("failed to set terminal to raw mode: %v", err)
	}
	defer syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), syscall.TCSETS, uintptr(unsafe.Pointer(&oldState)), 0, 0, 0)

	// Write ANSI escape sequence to request cursor position
	fmt.Print("\x1b[6n")

	// Read response
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('R')
	if err != nil {
		return 0, 0, err
	}

	// Parse response: ESC [ rows ; cols R
	response = strings.Trim(response, "\x1b[")
	response = strings.TrimSuffix(response, "R")
	parts := strings.Split(response, ";")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid response: %s", response)
	}

	row, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}

	col, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}

	return col, row, nil
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
