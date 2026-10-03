//go:build windows

package download

import (
	"errors"
	"syscall"
)

// isConnectionReset recognizes Winsock errors; Unix errno values are different on Windows.
func isConnectionReset(err error) bool {
	return errors.Is(err, syscall.WSAECONNRESET) || errors.Is(err, syscall.WSAECONNABORTED)
}
