package download

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/oshokin/zvuk-grabber/internal/config"
)

// idleReadConn times out a blocked network read, not time spent in the rate limiter.
// Request contexts still cancel transfers and TLS verification remains enabled.
type idleReadConn struct {
	// Conn is the underlying network connection.
	net.Conn

	// timeout is the idle read deadline applied before each Read.
	timeout time.Duration
}

// defaultKeepAlive probes otherwise idle audio connections.
const defaultKeepAlive = 30 * time.Second

// Read refreshes the idle deadline, then reads from the wrapped connection.
func (c *idleReadConn) Read(p []byte) (int, error) {
	if err := c.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return 0, fmt.Errorf("set download read deadline: %w", err)
	}

	return c.Conn.Read(p)
}

// NewTransport keeps connection setup bounded without putting a total
// deadline on the response body. Each client owns its transport and socket policy.
func NewTransport(cfg *config.DownloadHTTPConfig, paced bool) *http.Transport {
	if cfg == nil {
		cfg = config.DefaultDownloadHTTPConfig()
	}

	transport := http.DefaultTransport.(*http.Transport).Clone() //nolint:errcheck // net/http initializes DefaultTransport to *http.Transport.
	transport.TLSHandshakeTimeout = cfg.TLSHandshakeTimeout

	transport.ResponseHeaderTimeout = cfg.ResponseHeaderTimeout
	if paced && cfg.HTTP1Only {
		// Protocols is the Go 1.24+ API; it also controls TLS ALPN negotiation.
		transport.Protocols = new(http.Protocols)
		transport.Protocols.SetHTTP1(true)

		transport.ForceAttemptHTTP2 = false
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		} else {
			transport.TLSClientConfig = transport.TLSClientConfig.Clone()
		}

		transport.TLSClientConfig.NextProtos = []string{"http/1.1"}
	}

	dialer := &net.Dialer{Timeout: cfg.DialTimeout, KeepAlive: defaultKeepAlive}
	if paced && cfg.ReceiveBufferBytes > 0 {
		dialer.Control = receiveBufferControl(cfg.ReceiveBufferBytes)
	}

	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := dialer.DialContext(ctx, network, address)
		if err != nil {
			return nil, err
		}

		if cfg.ReadIdleTimeout > 0 {
			return &idleReadConn{Conn: conn, timeout: cfg.ReadIdleTimeout}, nil
		}

		return conn, nil
	}

	return transport
}
