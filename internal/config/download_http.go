package config

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/spf13/viper"
)

// DownloadHTTPConfig separates long audio transfers from bounded API calls.
// ReceiveBufferBytes and HTTP1Only are applied only when a speed limit is active.
type DownloadHTTPConfig struct {
	// Timeout is the optional overall client timeout; 0 means no total deadline.
	Timeout time.Duration `mapstructure:"timeout"`
	// DialTimeout bounds TCP connect time.
	DialTimeout time.Duration `mapstructure:"dial_timeout"`
	// TLSHandshakeTimeout bounds TLS setup.
	TLSHandshakeTimeout time.Duration `mapstructure:"tls_handshake_timeout"`
	// ResponseHeaderTimeout bounds waiting for response headers.
	ResponseHeaderTimeout time.Duration `mapstructure:"response_header_timeout"`
	// ReadIdleTimeout bounds stalled socket reads; 0 disables idle deadlines.
	ReadIdleTimeout time.Duration `mapstructure:"read_idle_timeout"`
	// ReceiveBufferBytes is SO_RCVBUF for paced downloads; 0 leaves the OS default.
	ReceiveBufferBytes int `mapstructure:"receive_buffer_bytes"`
	// HTTP1Only disables HTTP/2 when a download speed limit is active.
	HTTP1Only bool `mapstructure:"http1_only"`
}

const (
	// DefaultDownloadHTTPDialTimeout bounds TCP connect time for audio downloads.
	DefaultDownloadHTTPDialTimeout = 30 * time.Second
	// DefaultDownloadHTTPTLSHandshakeTimeout bounds TLS setup for audio downloads.
	DefaultDownloadHTTPTLSHandshakeTimeout = 10 * time.Second
	// DefaultDownloadHTTPResponseHeaderTimeout bounds waiting for response headers.
	DefaultDownloadHTTPResponseHeaderTimeout = 60 * time.Second
	// DefaultDownloadHTTPReadIdleTimeout bounds stalled socket reads during transfer.
	DefaultDownloadHTTPReadIdleTimeout = 60 * time.Second
	// DefaultDownloadHTTPReceiveBufferBytes is the paced-download socket receive buffer.
	DefaultDownloadHTTPReceiveBufferBytes = 256 * 1024
	// DefaultDownloadHTTPHTTP1Only disables HTTP/2 when a download speed limit is active.
	DefaultDownloadHTTPHTTP1Only = true

	// keyTotalTimeout is the mapstructure/viper key for the overall download timeout.
	keyTotalTimeout = "timeout"
	// keyDialTimeout is the mapstructure/viper key for TCP connect timeout.
	keyDialTimeout = "dial_timeout"
	// keyTLSHandshakeTimeout is the mapstructure/viper key for TLS handshake timeout.
	keyTLSHandshakeTimeout = "tls_handshake_timeout"
	// keyResponseHeaderTimeout is the mapstructure/viper key for response-header wait.
	keyResponseHeaderTimeout = "response_header_timeout"
	// keyReadIdleTimeout is the mapstructure/viper key for stalled-read timeout.
	keyReadIdleTimeout = "read_idle_timeout"
	// keyReceiveBufferBytes is the mapstructure/viper key for SO_RCVBUF.
	keyReceiveBufferBytes = "receive_buffer_bytes"
	// keyHTTP1Only is the mapstructure/viper key that disables HTTP/2 when paced.
	keyHTTP1Only = "http1_only"

	// maxDownloadHTTPReceiveBufferBytes is the signed 32-bit SO_RCVBUF upper bound.
	maxDownloadHTTPReceiveBufferBytes int64 = math.MaxInt32

	// viperPrefixDownloadHTTP is the YAML/viper prefix for download HTTP keys.
	viperPrefixDownloadHTTP = "zvuk_download_http."
)

// errDownloadHTTPConfig is returned when zvuk_download_http fails validation.
var errDownloadHTTPConfig = errors.New("invalid zvuk_download_http")

// DefaultDownloadHTTPConfig returns download HTTP settings used when the YAML block is omitted.
func DefaultDownloadHTTPConfig() *DownloadHTTPConfig {
	return &DownloadHTTPConfig{
		DialTimeout:           DefaultDownloadHTTPDialTimeout,
		TLSHandshakeTimeout:   DefaultDownloadHTTPTLSHandshakeTimeout,
		ResponseHeaderTimeout: DefaultDownloadHTTPResponseHeaderTimeout,
		ReadIdleTimeout:       DefaultDownloadHTTPReadIdleTimeout,
		ReceiveBufferBytes:    DefaultDownloadHTTPReceiveBufferBytes,
		HTTP1Only:             DefaultDownloadHTTPHTTP1Only,
	}
}

// Validate reports whether download HTTP settings are in range.
func (c *DownloadHTTPConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("%w: config is missing", errDownloadHTTPConfig)
	}

	if c.Timeout < 0 {
		return fmt.Errorf("%w: %s cannot be negative", errDownloadHTTPConfig, keyTotalTimeout)
	}

	if c.DialTimeout < 0 {
		return fmt.Errorf("%w: %s cannot be negative", errDownloadHTTPConfig, keyDialTimeout)
	}

	if c.TLSHandshakeTimeout < 0 {
		return fmt.Errorf("%w: %s cannot be negative", errDownloadHTTPConfig, keyTLSHandshakeTimeout)
	}

	if c.ResponseHeaderTimeout < 0 {
		return fmt.Errorf("%w: %s cannot be negative", errDownloadHTTPConfig, keyResponseHeaderTimeout)
	}

	if c.ReadIdleTimeout < 0 {
		return fmt.Errorf("%w: %s cannot be negative", errDownloadHTTPConfig, keyReadIdleTimeout)
	}

	// Bound the conversion to the signed 32-bit socket option on every target OS.
	if c.ReceiveBufferBytes < 0 || int64(c.ReceiveBufferBytes) > maxDownloadHTTPReceiveBufferBytes {
		return fmt.Errorf(
			"%w: %s must be between 0 and %d",
			errDownloadHTTPConfig,
			keyReceiveBufferBytes,
			maxDownloadHTTPReceiveBufferBytes,
		)
	}

	return nil
}

// setDownloadHTTPDefaults registers zvuk_download_http keys with viper.
func setDownloadHTTPDefaults(c *DownloadHTTPConfig) {
	if c == nil {
		c = DefaultDownloadHTTPConfig()
	}

	viper.SetDefault(viperPrefixDownloadHTTP+keyTotalTimeout, c.Timeout)
	viper.SetDefault(viperPrefixDownloadHTTP+keyDialTimeout, c.DialTimeout)
	viper.SetDefault(viperPrefixDownloadHTTP+keyTLSHandshakeTimeout, c.TLSHandshakeTimeout)
	viper.SetDefault(viperPrefixDownloadHTTP+keyResponseHeaderTimeout, c.ResponseHeaderTimeout)
	viper.SetDefault(viperPrefixDownloadHTTP+keyReadIdleTimeout, c.ReadIdleTimeout)
	viper.SetDefault(viperPrefixDownloadHTTP+keyReceiveBufferBytes, c.ReceiveBufferBytes)
	viper.SetDefault(viperPrefixDownloadHTTP+keyHTTP1Only, c.HTTP1Only)
}
