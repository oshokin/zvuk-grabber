//go:build !unix && !windows

package download

import (
	"fmt"
	"syscall"
)

// receiveBufferControl is a no-op error on platforms without SO_RCVBUF control.
func receiveBufferControl(_ int) func(string, string, syscall.RawConn) error {
	return func(_, _ string, _ syscall.RawConn) error {
		return fmt.Errorf(
			"receive buffer control is unsupported on this OS; set receive_buffer to 0 in the provider download HTTP block",
		)
	}
}
