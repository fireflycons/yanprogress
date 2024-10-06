//go:build (linux || aix || zos) && !appengine && !tinygo
// +build linux aix zos
// +build !appengine
// +build !tinygo

package yanprogress

import "golang.org/x/sys/unix"

// isatty return true if the file descriptor is terminal.
func isatty(fd uintptr) bool {
	_, err := unix.IoctlGetTermios(int(fd), unix.TCGETS)
	return err == nil
}
