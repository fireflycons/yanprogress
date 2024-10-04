//go:build (darwin || freebsd || openbsd || netbsd || dragonfly || hurd) && !appengine && !tinygo
// +build darwin freebsd openbsd netbsd dragonfly hurd
// +build !appengine
// +build !tinygo

package main

import "golang.org/x/sys/unix"

// IsTerminal return true if the file descriptor is terminal.
func isatty(fd uintptr) bool {
	_, err := unix.IoctlGetTermios(int(fd), unix.TIOCGETA)
	return err == nil
}
