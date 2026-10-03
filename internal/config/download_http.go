package config

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/spf13/viper"
)

// DownloadHTTPConfig separates long audio transfers from bounded API calls.
// ReceiveBufferBytes and HTTP1Only are applied only when a speed limit is active.
type DownloadHTTPConfig struct {
	// Timeout bounds one HTTP attempt including its body; 0 disables the total deadline.
	Timeout time.Duration `mapstructure:"timeout"`
	// DialTimeout bounds TCP connect time.
	DialTimeout time.Duration `mapstructure:"dial_timeout"`
	// TLSHandshakeTimeout bounds TLS setup.
	TLSHandshakeTimeout time.Duration `mapstructure:"tls_handshake_timeout"`
	// ResponseHeaderTimeout bounds waiting for response headers.
	ResponseHeaderTimeout time.Duration `mapstructure:"response_header_timeout"`
	// ReadIdleTimeout bounds stalled socket reads; 0 disables idle deadlines.
	ReadIdleTimeout time.Duration `mapstructure:"read_idle_timeout"`
	// ReceiveBuffer is the YAML size for SO_RCVBUF (`256KiB`, `0`). Empty or `0` leaves the OS default.
	ReceiveBuffer string `mapstructure:"receive_buffer"`
	// ReceiveBufferBytes is the parsed SO_RCVBUF in bytes; 0 leaves the OS default.
	ReceiveBufferBytes int `mapstructure:"-"`
	// MaxRetries bounds additional HTTP attempts for one audio URL, including body failures.
	MaxRetries int `mapstructure:"max_retries"`
	// RetryInitialDelay is the first backoff interval before retrying.
	RetryInitialDelay time.Duration `mapstructure:"retry_initial_delay"`
	// RetryMaxDelay caps exponential backoff and accepted Retry-After delays.
	RetryMaxDelay time.Duration `mapstructure:"retry_max_delay"`
	// Resume permits validated byte-range continuation within the current run.
	Resume bool `mapstructure:"resume"`
	// HTTP1Only disables HTTP/2 when a download speed limit is active.
	HTTP1Only bool `mapstructure:"http1_only"`
}

const (
	// DefaultDownloadHTTPResume enables validated byte-range continuation.
	DefaultDownloadHTTPResume = true
	// keyMaxRetries configures the additional request budget.
	keyMaxRetries = "max_retries"
	// keyRetryInitialDelay configures the first backoff interval.
	keyRetryInitialDelay = "retry_initial_delay"
	// keyRetryMaxDelay caps retry waits.
	keyRetryMaxDelay = "retry_max_delay"
	// keyResume enables byte-range continuation.
	keyResume = "resume"
	// DefaultDownloadHTTPMaxRetries allows three retries after the initial request.
	DefaultDownloadHTTPMaxRetries = 3
	// DefaultDownloadHTTPRetryInitialDelay avoids immediately hammering a failing CDN.
	DefaultDownloadHTTPRetryInitialDelay = time.Second
	// DefaultDownloadHTTPRetryMaxDelay bounds waiting between attempts.
	DefaultDownloadHTTPRetryMaxDelay = 30 * time.Second
	// MaxDownloadHTTPRetries bounds configuration mistakes and retry arithmetic.
	MaxDownloadHTTPRetries = 100
	// viperPrefixYandexDownloadHTTP identifies Yandex audio settings.
	viperPrefixYandexDownloadHTTP = "yandex_music_download_http."

	// DefaultDownloadHTTPDialTimeout bounds TCP connect time for audio downloads.
	DefaultDownloadHTTPDialTimeout = 30 * time.Second
	// DefaultDownloadHTTPTLSHandshakeTimeout bounds TLS setup for audio downloads.
	DefaultDownloadHTTPTLSHandshakeTimeout = 10 * time.Second
	// DefaultDownloadHTTPResponseHeaderTimeout bounds waiting for response headers.
	DefaultDownloadHTTPResponseHeaderTimeout = 60 * time.Second
	// DefaultDownloadHTTPReadIdleTimeout bounds stalled socket reads during transfer.
	DefaultDownloadHTTPReadIdleTimeout = 60 * time.Second
	// DefaultDownloadHTTPReceiveBuffer is the paced-download socket receive buffer size.
	DefaultDownloadHTTPReceiveBuffer = "256KiB"
	// DefaultDownloadHTTPReceiveBufferBytes is DefaultDownloadHTTPReceiveBuffer in bytes.
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
	// keyReceiveBuffer is the mapstructure/viper key for SO_RCVBUF.
	keyReceiveBuffer = "receive_buffer"
	// keyHTTP1Only is the mapstructure/viper key that disables HTTP/2 when paced.
	keyHTTP1Only = "http1_only"

	// maxDownloadHTTPReceiveBufferBytes is the signed 32-bit SO_RCVBUF upper bound.
	maxDownloadHTTPReceiveBufferBytes int64 = math.MaxInt32

	// viperPrefixDownloadHTTP is the YAML/viper prefix for download HTTP keys.
	viperPrefixDownloadHTTP = "zvuk_download_http."
)

// errDownloadHTTPConfig is returned when either provider’s audio settings fail validation.
var errDownloadHTTPConfig = errors.New("invalid download HTTP settings")

// DefaultDownloadHTTPConfig returns download HTTP settings used when the YAML block is omitted.
func DefaultDownloadHTTPConfig() *DownloadHTTPConfig {
	return &DownloadHTTPConfig{
		DialTimeout:           DefaultDownloadHTTPDialTimeout,
		TLSHandshakeTimeout:   DefaultDownloadHTTPTLSHandshakeTimeout,
		ResponseHeaderTimeout: DefaultDownloadHTTPResponseHeaderTimeout,
		ReadIdleTimeout:       DefaultDownloadHTTPReadIdleTimeout,
		ReceiveBuffer:         DefaultDownloadHTTPReceiveBuffer,
		ReceiveBufferBytes:    DefaultDownloadHTTPReceiveBufferBytes,
		HTTP1Only:             DefaultDownloadHTTPHTTP1Only,
		MaxRetries:            DefaultDownloadHTTPMaxRetries,
		RetryInitialDelay:     DefaultDownloadHTTPRetryInitialDelay,
		RetryMaxDelay:         DefaultDownloadHTTPRetryMaxDelay,
		Resume:                DefaultDownloadHTTPResume,
	}
}

// Clone returns a detached copy of download HTTP settings.
func (c *DownloadHTTPConfig) Clone() *DownloadHTTPConfig {
	if c == nil {
		return nil
	}

	return &DownloadHTTPConfig{
		Timeout:               c.Timeout,
		DialTimeout:           c.DialTimeout,
		TLSHandshakeTimeout:   c.TLSHandshakeTimeout,
		ResponseHeaderTimeout: c.ResponseHeaderTimeout,
		ReadIdleTimeout:       c.ReadIdleTimeout,
		ReceiveBuffer:         c.ReceiveBuffer,
		ReceiveBufferBytes:    c.ReceiveBufferBytes,
		MaxRetries:            c.MaxRetries,
		RetryInitialDelay:     c.RetryInitialDelay,
		RetryMaxDelay:         c.RetryMaxDelay,
		Resume:                c.Resume,
		HTTP1Only:             c.HTTP1Only,
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

	if err := c.parseReceiveBuffer(); err != nil {
		return err
	}

	if c.MaxRetries < 0 || c.MaxRetries > MaxDownloadHTTPRetries {
		return fmt.Errorf("%w: max_retries must be between 0 and %d", errDownloadHTTPConfig, MaxDownloadHTTPRetries)
	}

	if c.RetryInitialDelay <= 0 || c.RetryMaxDelay < c.RetryInitialDelay {
		return fmt.Errorf(
			"%w: retry_initial_delay must be positive and no greater than retry_max_delay",
			errDownloadHTTPConfig,
		)
	}

	return nil
}

// parseReceiveBuffer turns receive_buffer into ReceiveBufferBytes.
func (c *DownloadHTTPConfig) parseReceiveBuffer() error {
	raw := strings.TrimSpace(c.ReceiveBuffer)
	if raw == "" || raw == "0" {
		c.ReceiveBufferBytes = 0

		return nil
	}

	parsed, err := humanize.ParseBytes(raw)
	if err != nil {
		return fmt.Errorf("%w: %s: %w", errDownloadHTTPConfig, keyReceiveBuffer, err)
	}

	if parsed > uint64(maxDownloadHTTPReceiveBufferBytes) {
		return fmt.Errorf(
			"%w: %s must be between 0 and %d",
			errDownloadHTTPConfig,
			keyReceiveBuffer,
			maxDownloadHTTPReceiveBufferBytes,
		)
	}

	c.ReceiveBufferBytes = int(parsed)

	return nil
}

// setDownloadHTTPDefaults registers one provider’s audio transport defaults.
func setDownloadHTTPDefaults(prefix string, c *DownloadHTTPConfig) {
	if c == nil {
		c = DefaultDownloadHTTPConfig()
	}

	receiveBuffer := strings.TrimSpace(c.ReceiveBuffer)
	if receiveBuffer == "" {
		receiveBuffer = DefaultDownloadHTTPReceiveBuffer
	}

	viper.SetDefault(prefix+keyMaxRetries, c.MaxRetries)
	viper.SetDefault(prefix+keyRetryInitialDelay, c.RetryInitialDelay)
	viper.SetDefault(prefix+keyRetryMaxDelay, c.RetryMaxDelay)
	viper.SetDefault(prefix+keyResume, c.Resume)
	viper.SetDefault(prefix+keyTotalTimeout, c.Timeout)
	viper.SetDefault(prefix+keyDialTimeout, c.DialTimeout)
	viper.SetDefault(prefix+keyTLSHandshakeTimeout, c.TLSHandshakeTimeout)
	viper.SetDefault(prefix+keyResponseHeaderTimeout, c.ResponseHeaderTimeout)
	viper.SetDefault(prefix+keyReadIdleTimeout, c.ReadIdleTimeout)
	viper.SetDefault(prefix+keyReceiveBuffer, receiveBuffer)
	viper.SetDefault(prefix+keyHTTP1Only, c.HTTP1Only)
}
