//go:build windows && !appengine
// +build windows,!appengine

package main

import (
	"syscall"
	"unsafe"
)

// isatty return true if the file descriptor is a terminal.
func isatty(fd uintptr) bool {
	var st uint32
	handle := uintptr(syscall.Handle(fd))
	r1, r2, err := procGetConsoleMode.Call(handle, uintptr(unsafe.Pointer(&st)))
	_ = err
	return r1 != 0 && r2 == 0
}
