//go:build windows

package download

import (
	"net"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestWinsockResetClassification checks the actual wrapped errors returned by net on Windows.
func TestWinsockResetClassification(t *testing.T) {
	for _, cause := range []error{syscall.WSAECONNRESET, syscall.WSAECONNABORTED} {
		err := &net.OpError{Op: "read", Net: "tcp", Err: &os.SyscallError{Syscall: "wsarecv", Err: cause}}
		require.True(t, isTransient(err))
	}
	require.False(t, isTransient(syscall.WSAEACCES))
}
