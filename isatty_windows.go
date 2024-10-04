//go:build windows && !appengine
// +build windows,!appengine

package main

import "unsafe"

// isatty return true if the file descriptor is a terminal.
func isatty(fd uintptr) bool {
	var st uint32
	// r1, _, e := syscall.Syscall(procGetConsoleMode.Addr(), 2, fd, uintptr(unsafe.Pointer(&st)), 0)
	r1, r2, _ := procGetConsoleMode.Call(2, fd, uintptr(unsafe.Pointer(&st)), 0)
	return r1 != 0 && r2 == 0
}
