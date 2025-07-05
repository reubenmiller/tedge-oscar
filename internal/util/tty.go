package util

import "golang.org/x/term"

// Isatty returns true if the given file descriptor is a terminal.
func Isatty(fd uintptr) bool {
	return term.IsTerminal(int(fd))
}
