//nolint:gocritic,err113,gosec // Yandex API client keeps provider flow stages together; md5 is required by Yandex direct-link signature.
package yandex

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/oshokin/zvuk-grabber/internal/client/yandex/model"
	httptransport "github.com/oshokin/zvuk-grabber/internal/transport/http"
)

// RemoteFile represents an opened remote media stream.
type RemoteFile struct {
	// Body is the readable response stream.
	Body io.ReadCloser
	// ContentLength is the reported content length, or -1 if unknown.
	ContentLength int64
	// ContentType is the HTTP Content-Type header value.
	ContentType string
}

// Client is the Yandex Music API client.
type Client struct {
	// httpClient performs authenticated API and download requests.
	httpClient *HttpClient
	// losslessDownloader resolves and downloads lossless audio streams.
	losslessDownloader losslessTrackDownloader

	// accountMu protects cached account fields.
	accountMu sync.Mutex
	// userUID is the cached Yandex account UID.
	userUID int
	// username is the cached Yandex account login.
	username string
}

// losslessTrackDownloader resolves and downloads lossless track audio.
type losslessTrackDownloader interface {
	// GetDownloadInfo fetches lossless download metadata for a track.
	GetDownloadInfo(reqCtx *httptransport.RequestLogContext, trackID string, userUID int) (*DownloadInfo, error)
	// DownloadAudio downloads lossless audio using resolved download info.
	DownloadAudio(reqCtx *httptransport.RequestLogContext, info *DownloadInfo) ([]byte, error)
}

const (
	// baseURL is the Yandex Music API base URL.
	baseURL = "https://api.music.yandex.net"
	// signSalt is the salt used to sign direct MP3 download links.
	signSalt = "XGRlBW9FXlekgbPrRHuSiA"
	// lyricsSignKey is the HMAC key used to sign track lyrics requests.
	lyricsSignKey = "p93jhgh689SBReK6ghtw62"
	// lyricsFormatLRC requests time-synced lyrics payload from track lyrics endpoint.
	lyricsFormatLRC = "LRC"
	// sensitiveFieldToken is the key for generic token values.
	sensitiveFieldToken = "token"
	// sensitiveFieldAccessToken is the key for OAuth access tokens.
	sensitiveFieldAccessToken = "access_token"
	// sensitiveFieldRefreshToken is the key for OAuth refresh tokens.
	sensitiveFieldRefreshToken = "refresh_token"
	// sensitiveFieldClientSecret is the key for OAuth client secrets.
	sensitiveFieldClientSecret = "client_secret"
	// sensitiveFieldSessionID is the key for sessionid values.
	sensitiveFieldSessionID = "sessionid"
	// sensitiveFieldSessionIDAlt is the key for session_id values.
	sensitiveFieldSessionIDAlt = "session_id"
	// sensitiveFieldPassport is the key for passport values.
	sensitiveFieldPassport = "passport"
	// Bitrate64 is the 64 Kbps MP3 bitrate constant.
	Bitrate64 = 64
	// Bitrate128 is the 128 Kbps MP3 bitrate constant.
	Bitrate128 = 128
	// Bitrate192 is the 192 Kbps MP3 bitrate constant.
	Bitrate192 = 192
	// Bitrate320 is the 320 Kbps MP3 bitrate constant.
	Bitrate320 = 320
	// CodecMP3 is the MP3 codec identifier used in download-info responses.
	CodecMP3 = "mp3"
)

var (
	// requestSensitiveFieldKeys defines Yandex-specific structured log fields that must be redacted.
	requestSensitiveFieldKeys = []string{
		"authorization",
		"cookie",
		"set-cookie",
		"proxy-authorization",
		sensitiveFieldToken,
		sensitiveFieldAccessToken,
		sensitiveFieldRefreshToken,
		sensitiveFieldClientSecret,
		sensitiveFieldSessionID,
		sensitiveFieldSessionIDAlt,
		sensitiveFieldPassport,
		"secret",
	}
	// requestSensitiveQueryKeys defines Yandex-specific URL query params that must be redacted.
	requestSensitiveQueryKeys = []string{
		sensitiveFieldToken,
		sensitiveFieldAccessToken,
		sensitiveFieldRefreshToken,
		sensitiveFieldClientSecret,
		sensitiveFieldSessionID,
		sensitiveFieldSessionIDAlt,
		sensitiveFieldPassport,
		"sig",
		"sign",
		"signature",
		"x-amz-signature",
		"x-amz-credential",
		"x-amz-security-token",
		"x-goog-signature",
		"x-goog-credential",
		"x-goog-security-token",
	}
)

// NewClient creates a Yandex Music client with the given HTTP client.
func NewClient(httpClient *HttpClient) *Client {
	if httpClient == nil {
		httpClient = NewHttpClient(nil)
	}

	return &Client{
		httpClient:         httpClient,
		losslessDownloader: newLosslessDownloader(httpClient),
	}
}

// SetToken configures the OAuth bearer token for subsequent requests.
func (c *Client) SetToken(token string) {
	c.httpClient.SetToken(token)
}

// Cancel cancels in-flight requests on the underlying HTTP client.
func (c *Client) Cancel() {
	if c == nil || c.httpClient == nil {
		return
	}

	c.httpClient.Cancel()
}

// CloseIdleConnections closes idle API and audio keep-alives so the process can exit.
func (c *Client) CloseIdleConnections() {
	if c == nil || c.httpClient == nil {
		return
	}

	c.httpClient.CloseIdleConnections()
}

// ResetCancel recreates the base request context on the underlying HTTP client.
func (c *Client) ResetCancel() {
	if c == nil || c.httpClient == nil {
		return
	}

	c.httpClient.ResetCancel()
}

// AccountStatus fetches the authenticated account and caches UID and login.
func (c *Client) AccountStatus(ctx context.Context) (*model.Account, error) {
	endpoint := baseURL + "/account/status"

	response, err := c.httpClient.GetWithContext(
		c.requestContext(ctx, "account_status", "fetch_account_status"),
		endpoint,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get account status: %w", err)
	}

	var data model.AccountStatusResponse

	if err = parseResponse(response, &data); err != nil {
		return nil, err
	}

	c.accountMu.Lock()
	if uid := data.Result.Account.Uid; uid != 0 {
		c.userUID = uid
		c.username = data.Result.Account.Login
	}

	c.accountMu.Unlock()

	return &data.Result.Account, nil
}

// TrackInfo fetches metadata for a single track by ID.
func (c *Client) TrackInfo(ctx context.Context, id string) (*model.Track, error) {
	data, err := getJSON[model.TracksResponse](
		c,
		c.requestContext(ctx, "track_info", "fetch_track_info"),
		fmt.Sprintf("%s/tracks/%s", baseURL, id),
	)
	if err != nil {
		return nil, err
	}

	if len(data.Result) == 0 {
		return nil, errors.New("track not found")
	}

	return &data.Result[0], nil
}

// TrackLyrics fetches lyrics metadata and text for a single track.
func (c *Client) TrackLyrics(ctx context.Context, id string) (*model.TrackLyrics, error) {
	trackID := normalizeTrackID(id)
	if trackID == "" {
		return nil, errors.New("track id is empty")
	}

	endpoint := buildTrackLyricsURL(trackID, lyricsFormatLRC, time.Now().Unix())

	response, err := c.httpClient.GetWithContext(c.requestContext(ctx, "track_lyrics", "fetch_track_lyrics"), endpoint)
	if err != nil {
		return nil, err
	}

	var wrapped model.TrackLyricsResponse

	if err = parseResponse(response, &wrapped); err == nil && wrapped.Result != nil {
		return c.hydrateLyricsText(ctx, wrapped.Result)
	}

	var direct model.TrackLyrics

	if err = parseResponse(response, &direct); err != nil {
		return nil, err
	}

	if direct.Text() == "" && direct.DownloadLink() == "" {
		return nil, errors.New("track lyrics not found")
	}

	return c.hydrateLyricsText(ctx, &direct)
}

// AlbumWithTracks fetches album metadata including track volumes.
func (c *Client) AlbumWithTracks(ctx context.Context, id string) (*model.Album, error) {
	data, err := getJSON[model.AlbumResponse](
		c,
		c.requestContext(ctx, "album_with_tracks", "fetch_album_with_tracks"),
		fmt.Sprintf("%s/albums/%s/with-tracks", baseURL, id),
	)
	if err != nil {
		return nil, err
	}

	if data.Result.ID.String() == "" && len(data.Result.Volumes) == 0 {
		return nil, errors.New("album not found")
	}

	return &data.Result, nil
}

// UsersPlaylist fetches a user-owned playlist by username and playlist ID.
func (c *Client) UsersPlaylist(ctx context.Context, id string, username string) (*model.Playlist, error) {
	return c.fetchPlaylist(
		c.requestContext(ctx, "playlist", "fetch_users_playlist"),
		fmt.Sprintf("%s/users/%s/playlists/%s", baseURL, username, id),
	)
}

// PlaylistByUUID fetches a playlist by its UUID.
func (c *Client) PlaylistByUUID(ctx context.Context, id string) (*model.Playlist, error) {
	return c.fetchPlaylist(
		c.requestContext(ctx, "playlist", "fetch_uuid_playlist"),
		fmt.Sprintf("%s/playlist/%s", baseURL, id),
	)
}

// TracksDownloadInfo fetches available download options for a track.
func (c *Client) TracksDownloadInfo(ctx context.Context, trackID string) ([]model.DownloadInfo, error) {
	data, err := getJSON[model.DownloadInfoResponse](
		c,
		c.requestContext(ctx, "download_info", "fetch_download_info"),
		fmt.Sprintf("%s/tracks/%s/download-info", baseURL, trackID),
	)
	if err != nil {
		return nil, err
	}

	return data.Result, nil
}

// TrackDownloadLink resolves a direct MP3 download URL from a download-info URL.
func (c *Client) TrackDownloadLink(ctx context.Context, url string) (string, error) {
	response, err := c.httpClient.GetWithContext(
		c.requestContext(ctx, "resolve_download_link", "fetch_direct_link"),
		url,
	)
	if err != nil {
		return "", err
	}

	var info model.TrackDownloadInfo

	if err = xml.Unmarshal(response, &info); err != nil {
		return "", err
	}

	return buildDirectLink(&info), nil
}

// hydrateLyricsText downloads lyrics text when the API response only contains a link.
func (c *Client) hydrateLyricsText(ctx context.Context, lyrics *model.TrackLyrics) (*model.TrackLyrics, error) {
	if lyrics == nil || lyrics.Text() != "" {
		return lyrics, nil
	}

	downloadURL := lyrics.DownloadLink()
	if downloadURL == "" {
		return lyrics, nil
	}

	body, err := c.httpClient.DownloadBytesWithContext(
		c.requestContext(ctx, "lyrics_download", "download_lyrics"),
		downloadURL,
	)
	if err != nil {
		return nil, err
	}

	lyrics.FullLyrics = strings.TrimSpace(string(body))
	if lyrics.FullLyrics == "" {
		return nil, errors.New("track lyrics text is empty")
	}

	return lyrics, nil
}

// buildTrackLyricsURL constructs a signed lyrics API URL for the given track.
func buildTrackLyricsURL(trackID, format string, timestamp int64) string {
	normalizedTrackID := normalizeLyricsSignTrackID(trackID)

	values := url.Values{}
	values.Set("timeStamp", strconv.FormatInt(timestamp, 10))
	values.Set("sign", signLyricsRequest(normalizedTrackID, timestamp))

	lyricsFormat := strings.TrimSpace(format)
	if lyricsFormat != "" {
		values.Set("format", lyricsFormat)
	}

	return fmt.Sprintf("%s/tracks/%s/lyrics?%s", baseURL, url.PathEscape(normalizedTrackID), values.Encode())
}

// signLyricsRequest computes the HMAC signature required by the lyrics endpoint.
func signLyricsRequest(trackID string, timestamp int64) string {
	normalizedTrackID := normalizeLyricsSignTrackID(trackID)

	mac := hmac.New(sha256.New, []byte(lyricsSignKey))
	_, _ = mac.Write([]byte(normalizedTrackID + strconv.FormatInt(timestamp, 10)))

	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// fetchPlaylist fetches playlist metadata from the given API URL.
func (c *Client) fetchPlaylist(reqCtx *httptransport.RequestLogContext, url string) (*model.Playlist, error) {
	data, err := getJSON[model.PlaylistResponse](c, reqCtx, url)
	if err != nil {
		return nil, err
	}

	return &data.Result, nil
}

// ensureUserUID returns the cached account UID, fetching account status if needed.
func (c *Client) ensureUserUID(ctx context.Context) (int, error) {
	c.accountMu.Lock()
	userUID := c.userUID
	c.accountMu.Unlock()

	if userUID != 0 {
		return userUID, nil
	}

	if _, err := c.AccountStatus(ctx); err != nil {
		return 0, err
	}

	c.accountMu.Lock()
	userUID = c.userUID
	c.accountMu.Unlock()

	return userUID, nil
}

// requestContext builds a transport request log context for an operation.
func (c *Client) requestContext(ctx context.Context, stage, operation string) *httptransport.RequestLogContext {
	return &httptransport.RequestLogContext{
		Ctx:                ctx,
		Stage:              stage,
		Operation:          operation,
		SensitiveFieldKeys: requestSensitiveFieldKeys,
		SensitiveQueryKeys: requestSensitiveQueryKeys,
	}
}

// getJSON sends a GET request and decodes a JSON API response.
func getJSON[T any](c *Client, reqCtx *httptransport.RequestLogContext, url string) (T, error) {
	var result T

	response, err := c.httpClient.GetWithContext(reqCtx, url)
	if err != nil {
		return result, err
	}

	if err = parseResponse(response, &result); err != nil {
		return result, err
	}

	return result, nil
}

// parseResponse unmarshals a JSON API response body into the target value.
func parseResponse(responseBody []byte, response any) error {
	if err := json.Unmarshal(responseBody, response); err != nil {
		return fmt.Errorf("error parsing response: %w", err)
	}

	return nil
}

// buildDirectLink constructs a signed direct MP3 download URL from track download info.
func buildDirectLink(info *model.TrackDownloadInfo) string {
	if info == nil || info.Path == "" {
		return ""
	}

	pathWithoutFirstChar := info.Path[1:]
	signData := signSalt + pathWithoutFirstChar + info.S
	hash := md5.Sum([]byte(signData))
	sign := hex.EncodeToString(hash[:])

	return fmt.Sprintf("https://%s/get-mp3/%s/%s%s", info.Host, sign, info.Ts, info.Path)
}

// normalizeTrackID trims whitespace from a track identifier.
func normalizeTrackID(trackID string) string {
	return strings.TrimSpace(trackID)
}

// normalizeLyricsSignTrackID strips album suffixes before lyrics request signing.
func normalizeLyricsSignTrackID(trackID string) string {
	normalized := normalizeTrackID(trackID)
	if normalized == "" {
		return ""
	}

	parts := strings.SplitN(normalized, ":", 2)
	if len(parts) == 0 {
		return normalized
	}

	return strings.TrimSpace(parts[0])
}
