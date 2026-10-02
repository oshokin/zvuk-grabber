//go:build !unix && !windows

package zvuk

import (
	"fmt"
	"syscall"
)

// receiveBufferControl is a no-op error on platforms without SO_RCVBUF control.
func receiveBufferControl(size int) func(string, string, syscall.RawConn) error {
	return func(_, _ string, _ syscall.RawConn) error {
		return fmt.Errorf(
			"receive buffer control is unsupported on this OS; set zvuk_download_http.receive_buffer_bytes to 0",
		)
	}
}
