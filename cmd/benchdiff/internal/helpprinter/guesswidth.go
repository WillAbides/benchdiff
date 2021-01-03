// +build appengine !linux,!freebsd,!darwin,!dragonfly,!netbsd,!openbsd

package helpprinter

import "io"

func guessWidth(w io.Writer) int {
	return 80
}
