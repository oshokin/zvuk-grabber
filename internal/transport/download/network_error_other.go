//go:build !windows

package download

import (
	"errors"
	"syscall"
)

// isConnectionReset recognizes interrupted sockets on Unix-like systems.
func isConnectionReset(err error) bool {
	return errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.ECONNABORTED) || errors.Is(err, syscall.EPIPE)
}
