//go:build windows

package download

import (
	"fmt"
	"syscall"
)

// receiveBufferControl sets SO_RCVBUF before connect, before window negotiation.
// Linux may double/clamp this request; other kernels have different accounting.
func receiveBufferControl(size int) func(string, string, syscall.RawConn) error {
	return func(_, _ string, raw syscall.RawConn) error {
		var socketErr error
		if err := raw.Control(func(fd uintptr) {
			socketErr = syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, size)
		}); err != nil {
			return fmt.Errorf("control download socket: %w", err)
		}
		if socketErr != nil {
			return fmt.Errorf("set download receive buffer: %w", socketErr)
		}
		return nil
	}
}
