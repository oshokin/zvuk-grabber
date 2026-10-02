//nolint:err113 // Lossless flow mirrors Yandex API contract and keeps request structs close to codec/decrypt logic.
package yandex

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/oshokin/zvuk-grabber/internal/media"
	httptransport "github.com/oshokin/zvuk-grabber/internal/transport/http"
)

// DownloadInfo contains resolved lossless download metadata.
type DownloadInfo struct {
	// Quality is the requested stream quality label.
	Quality string
	// Codec is the audio codec reported by the API.
	Codec string
	// URLs are candidate direct download URLs.
	URLs []string
	// Key is the hex-encoded AES key for encrypted streams.
	Key string
	// Bitrate is the reported stream bitrate in Kbps.
	Bitrate int
}

// losslessHTTPClient performs lossless download HTTP operations.
type losslessHTTPClient interface {
	// GetWithContextAndHeaders sends a GET request with extra headers.
	GetWithContextAndHeaders(
		reqCtx *httptransport.RequestLogContext,
		url string,
		headers map[string]string,
	) ([]byte, error)
	// DownloadBytesWithContext downloads a remote resource as bytes.
	DownloadBytesWithContext(reqCtx *httptransport.RequestLogContext, url string) ([]byte, error)
}

// losslessDownloader resolves and downloads lossless audio from Yandex Music.
type losslessDownloader struct {
	// httpClient sends authenticated file-info and download requests.
	httpClient losslessHTTPClient
	// baseURL is the Yandex Music API base URL.
	baseURL string
	// now returns the current time for request signing.
	now func() time.Time
}

// fileInfoResponse mirrors the get-file-info API payload shape.
type fileInfoResponse struct {
	// DownloadInfo is the snake_case download info block.
	DownloadInfo *downloadInfoPayload `json:"download_info,omitempty"`
	// DownloadInfoCamel is the camelCase download info block.
	DownloadInfoCamel *downloadInfoPayload `json:"downloadInfo,omitempty"`
	// Result wraps nested download info variants.
	Result *struct {
		// DownloadInfo is the snake_case download info block inside result.
		DownloadInfo *downloadInfoPayload `json:"download_info,omitempty"`
		// DownloadInfoCamel is the camelCase download info block inside result.
		DownloadInfoCamel *downloadInfoPayload `json:"downloadInfo,omitempty"`
	} `json:"result,omitempty"`
}

// downloadInfoPayload is the nested download info object in file-info responses.
type downloadInfoPayload struct {
	// Quality is the stream quality label.
	Quality string `json:"quality"`
	// Codec is the audio codec name.
	Codec string `json:"codec"`
	// URLs are direct download URLs.
	URLs []string `json:"urls"`
	// Key is the hex-encoded decryption key.
	Key string `json:"key"`
	// Bitrate is the stream bitrate in Kbps.
	Bitrate int `json:"bitrate"`
}

const (
	// defaultSignKey is the HMAC key used to sign lossless file-info requests.
	defaultSignKey = "7tvSmFbyf5hJnIHhCimDDD"
	// transportEncRaw is the encrypted raw transport used by the current Yandex web player.
	transportEncRaw = "encraw"
	// transportRaw is the legacy unencrypted lossless transport.
	transportRaw = "raw"
)

var (
	// ErrNoFLACDownloadInfo is returned when the API provides no FLAC download info.
	ErrNoFLACDownloadInfo = errors.New("no flac download info available")
	// ErrNoDownloadURLs is returned when download info contains no URLs.
	ErrNoDownloadURLs = errors.New("no lossless download urls available")
)

// BuildFileInfoURL builds a signed get-file-info endpoint URL for a track.
func BuildFileInfoURL(baseURL, trackID, transport string, timestamp int64) string {
	if transport == "" {
		transport = transportEncRaw
	}

	values := url.Values{}
	values.Set("ts", strconv.FormatInt(timestamp, 10))
	values.Set("trackId", trackID)
	values.Set("quality", "lossless")
	values.Set("codecs", strings.Join(supportedCodecs(), ","))
	values.Set("transports", transport)
	values.Set("sign", SignRequest(timestamp, trackID, transport))

	return strings.TrimRight(baseURL, "/") + "/get-file-info?" + values.Encode()
}

// SignRequest computes the HMAC signature for a lossless file-info request.
func SignRequest(timestamp int64, trackID, transport string) string {
	if transport == "" {
		transport = transportEncRaw
	}

	signData := fmt.Sprintf(
		"%d%slossless%s%s",
		timestamp,
		trackID,
		strings.Join(supportedCodecs(), ""),
		transport,
	)
	mac := hmac.New(sha256.New, []byte(defaultSignKey))
	mac.Write([]byte(signData))
	sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return strings.TrimRight(sign, "=")
}

// ParseDownloadInfo extracts DownloadInfo from a get-file-info response body.
func ParseDownloadInfo(body []byte) (*DownloadInfo, error) {
	var response fileInfoResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse lossless download info: %w", err)
	}

	payload := response.DownloadInfo
	if payload == nil {
		payload = response.DownloadInfoCamel
	}

	if payload == nil && response.Result != nil {
		payload = response.Result.DownloadInfo
		if payload == nil {
			payload = response.Result.DownloadInfoCamel
		}
	}

	if payload == nil {
		return nil, ErrNoFLACDownloadInfo
	}

	return &DownloadInfo{
		Quality: payload.Quality,
		Codec:   payload.Codec,
		URLs:    payload.URLs,
		Key:     payload.Key,
		Bitrate: payload.Bitrate,
	}, nil
}

// DecryptData decrypts AES-CTR encrypted lossless audio using a hex-encoded key.
func DecryptData(data []byte, key string) ([]byte, error) {
	decodedKey, err := hex.DecodeString(strings.TrimSpace(key))
	if err != nil {
		return nil, fmt.Errorf("failed to decode lossless decryption key: %w", err)
	}

	block, err := aes.NewCipher(decodedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize lossless decryptor: %w", err)
	}

	// Yandex encraw is AES-CTR with an all-zero IV; the web player uses the same nonce.
	var (
		iv        [aes.BlockSize]byte
		stream    = cipher.NewCTR(block, iv[:])
		decrypted = make([]byte, len(data))
	)

	stream.XORKeyStream(decrypted, data)

	return decrypted, nil
}

// newLosslessDownloader creates a lossless downloader backed by the given HTTP client.
func newLosslessDownloader(httpClient losslessHTTPClient) *losslessDownloader {
	return &losslessDownloader{
		httpClient: httpClient,
		baseURL:    baseURL,
		now:        time.Now,
	}
}

// GetDownloadInfo fetches and parses lossless download metadata for a track.
func (d *losslessDownloader) GetDownloadInfo(
	reqCtx *httptransport.RequestLogContext,
	trackID string,
	userUID int,
) (*DownloadInfo, error) {
	if d == nil || d.httpClient == nil {
		return nil, errors.New("lossless downloader is not configured")
	}

	var errs []error

	for _, transport := range losslessTransports() {
		info, err := d.getDownloadInfo(reqCtx, trackID, userUID, transport)
		if err == nil {
			return info, nil
		}

		errs = append(errs, err)
	}

	return nil, errors.Join(errs...)
}

// DownloadAudio downloads and optionally decrypts lossless audio from the given info.
func (d *losslessDownloader) DownloadAudio(
	reqCtx *httptransport.RequestLogContext,
	info *DownloadInfo,
) ([]byte, error) {
	if d == nil || d.httpClient == nil {
		return nil, errors.New("lossless downloader is not configured")
	}

	if info == nil {
		return nil, ErrNoDownloadURLs
	}

	if len(info.URLs) == 0 {
		return nil, ErrNoDownloadURLs
	}

	var errs []error

	for _, rawURL := range info.URLs {
		data, err := d.httpClient.DownloadBytesWithContext(reqCtx, rawURL)
		if err != nil {
			errs = append(
				errs,
				fmt.Errorf(
					"%s: %w",
					httptransport.SanitizeURLWithKeys(rawURL, requestSensitiveQueryKeys, requestSensitiveFieldKeys),
					err,
				),
			)

			continue
		}

		if strings.TrimSpace(info.Key) != "" {
			data, err = DecryptData(data, info.Key)
			if err != nil {
				return nil, err
			}
		}

		return data, nil
	}

	return nil, errors.Join(errs...)
}

// getDownloadInfo fetches lossless metadata using a single transport.
func (d *losslessDownloader) getDownloadInfo(
	reqCtx *httptransport.RequestLogContext,
	trackID string,
	userUID int,
	transport string,
) (*DownloadInfo, error) {
	endpoint := BuildFileInfoURL(d.baseURL, trackID, transport, d.now().Unix())

	body, err := d.httpClient.GetWithContextAndHeaders(reqCtx, endpoint, buildFileInfoHeaders(userUID))
	if err != nil {
		return nil, err
	}

	info, err := ParseDownloadInfo(body)
	if err != nil {
		return nil, err
	}

	codec := media.ParseCodec(info.Codec)
	if !codec.IsFLACFamily() {
		return nil, fmt.Errorf(
			"%w: expected FLAC lossless, got %s",
			ErrNoFLACDownloadInfo,
			codec.Description(),
		)
	}

	if len(info.URLs) == 0 {
		return nil, ErrNoDownloadURLs
	}

	return info, nil
}

// supportedCodecs returns codec names accepted by the lossless file-info endpoint.
func supportedCodecs() []string {
	return []string{
		"flac",
		"aac",
		"he-aac",
		"mp3",
		"flac-mp4",
		"aac-mp4",
		"he-aac-mp4",
	}
}

// buildFileInfoHeaders builds HTTP headers required by the get-file-info endpoint.
func buildFileInfoHeaders(userUID int) map[string]string {
	headers := map[string]string{
		headerUserAgent:                          "YandexMusicAPI/1.0.0",
		"x-yandex-music-client":                  "YandexMusicWebNext/1.0.0",
		"x-yandex-music-without-invocation-info": "1",
		"Referer":                                "https://music.yandex.ru/",
		"Origin":                                 "https://music.yandex.ru",
	}
	if userUID > 0 {
		headers["x-yandex-music-multi-auth-user-id"] = strconv.Itoa(userUID)
	}

	return headers
}

// losslessTransports returns file-info transports in preference order.
func losslessTransports() []string {
	return []string{transportEncRaw, transportRaw}
}
