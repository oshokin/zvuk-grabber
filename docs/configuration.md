# Configuration

Copy [`.zvuk-grabber.example.yaml`](../.zvuk-grabber.example.yaml) to `.zvuk-grabber.yaml`. Omitted keys use the defaults below. Settings apply to both providers unless a key says otherwise.

Durations use Go syntax (`30s`, `5m`, `1h`). Nested HTTP timeouts accept `0s` to disable that deadline. Login: [Usage](usage.md#sign-in).

## Tokens

- **`zvuk_auth_token`**: Zvuk API token. Easiest: `zvuk-grabber auth zvuk login`. Manual: [Zvuk profile JSON](https://zvuk.com/api/v2/tiny/profile), path `$.result.profile.token`.

    ```yaml
    zvuk_auth_token: "your_token_here"
    ```

- **`yandex_music_token`**: Yandex Music OAuth token. Easiest: `zvuk-grabber auth yandex login`.

    ```yaml
    yandex_music_token: "your_token_here"
    ```

There is no `auth_token` key and no fallback from old configs.

## Quality and duration

- **`quality`**: preferred audio. Default `3`.

  - `1` = MP3, 128 Kbps
  - `2` = MP3, 320 Kbps
  - `3` = FLAC lossless first, MP3 if FLAC is missing

    Native FLAC is saved as-is. If a `.flac` body starts with an MP4 `ftyp` box, the program remuxes supported unencrypted, non-fragmented FLAC into a temp file before tagging. Audio is copied, not decoded. Extra disk is needed until conversion succeeds; the original download stays until then; temps are removed on failure. Multi-track, encrypted, fragmented, and malformed containers are rejected.

    ```yaml
    quality: 3
    ```

- **`min_quality`**: skip tracks below this. Default `0` (no filter). `1`/`2`/`3` match `quality`. Must be `<= quality`.

    ```yaml
    quality: 3
    min_quality: 2
    ```

    FLAC only:

    ```yaml
    quality: 3
    min_quality: 3
    ```

- **`min_duration`**: skip shorter tracks (`30s`, `1m`, `1m30s`). Empty = no filter. Useful for intros, skits, and 3-second "samples".

    ```yaml
    min_duration: "30s"
    ```

- **`max_duration`**: skip longer tracks (`10m`, `15m`, `1h`). Empty = no filter. If both are set, `max_duration` must be greater than `min_duration`.

    ```yaml
    min_duration: "30s"
    max_duration: "10m"
    ```

## Output

- **`output_path`**: download root, relative or absolute. Default `zvuk-grabber-downloads`. Empty string becomes that default.

    ```yaml
    output_path: "zvuk-grabber-downloads"
    ```

- **`group_by_provider`**: default `true`. Files go under `output_path/zvuk` and `output_path/yandex`. Set `false` for one shared folder.

    ```yaml
    group_by_provider: true
    ```

- **`create_folder_for_singles`**: default `false`. `true` puts standalone tracks in their own folder. `false` writes them in the output directory.

    ```yaml
    create_folder_for_singles: false
    ```

- **`max_folder_name_length`**: truncate generated folder names. `0` means do not truncate. Default `100`.

    ```yaml
    max_folder_name_length: 100
    ```

## File names

Templates are Go `text/template`. Unknown placeholders render empty. `/` or `\` in a folder template creates nested folders and is converted to the host separator.

Examples below were read from the live catalog for the URLs already used in [Usage](usage.md). Catalog data moves. If a placeholder is empty for that URL, the template just drops it.

Sources:

- Zvuk album: [https://zvuk.com/release/36599795](https://zvuk.com/release/36599795) (Benjah, *Heal*, track 1 *From the Root*)
- Zvuk playlist: [https://zvuk.com/playlist/9037842](https://zvuk.com/playlist/9037842) (*Зимние этюды*, first track is Tchaikovsky / Dudamel)
- Zvuk audiobook: [https://zvuk.com/abook/37364537](https://zvuk.com/abook/37364537) (Стивен Кинг, *Темная Башня III: Бесплодные земли*)
- Zvuk podcast: [https://zvuk.com/podcast/12891594](https://zvuk.com/podcast/12891594) (*Молодые и Глупые*)
- Yandex album: [https://music.yandex.ru/album/2176030](https://music.yandex.ru/album/2176030) (Tremonti, *All I Was*, track 1 *Leave It Alone*, id `19326613`)
- Yandex track id `17588871` from the Usage URL currently belongs to [https://music.yandex.ru/album/2063889](https://music.yandex.ru/album/2063889) (The Kiboomers, *Favorite Preschool Songs*, track 17 *Mary Had a Little Lamb*), not album `2176030`

The Yandex playlist URLs in Usage (`yamusic-daily/playlists/1000` and the sample UUID) currently 404. Playlist placeholder examples therefore use the Zvuk playlist above. Same keys apply when a Yandex playlist URL still exists.

- **`track_filename_template`**: album track file name. Default `{{.trackNumberPad}} - {{.trackTitle}}`.

  - `{{.albumArtist}}`: album artist(s). Zvuk `36599795`: `Benjah`. Yandex `2176030`: `Tremonti`
  - `{{.albumID}}`: album id. Zvuk: `36599795`. Yandex: `2176030`
  - `{{.albumTitle}}`: album title. Zvuk: `Heal`. Yandex: `All I Was`
  - `{{.albumTrackCount}}`: tracks on the album. Zvuk: `12`. Yandex: `12`
  - `{{.collectionTitle}}`: album title. Same as `albumTitle` here: `Heal` / `All I Was`
  - `{{.recordLabel}}`: label. Empty on these two albums. Filled for Zvuk audiobooks (see below)
  - `{{.releaseDate}}`: album date `YYYY-MM-DD`. Zvuk: `2025-01-24`. Yandex: `2012-07-17`
  - `{{.releaseYear}}`: album year. Zvuk: `2025`. Yandex: `2012`
  - `{{.trackArtist}}`: track artist(s). Zvuk track 1: `Benjah`. Yandex track 1: `Tremonti`
  - `{{.trackCount}}`: tracks on the album. `12` / `12`
  - `{{.trackGenre}}`: genre(s). Zvuk: `Hip-Hop, Rap`. Yandex: `alternativemetal`
  - `{{.trackID}}`: track id. Zvuk: `141865883`. Yandex: `19326613`
  - `{{.trackNumber}}`: track number, no padding. `1` / `1`
  - `{{.trackNumberPad}}`: two-digit track number (`01`, `02`). `01` / `01`
  - `{{.trackTitle}}`: title. Zvuk: `From the Root`. Yandex: `Leave It Alone`
  - `{{.type}}`: `album`

    ```yaml
    track_filename_template: "{{.trackNumberPad}} - {{.trackTitle}}"
    ```

    Renders as `01 - From the Root` (Zvuk) or `01 - Leave It Alone` (Yandex). Same template on track `17588871`: `17 - Mary Had a Little Lamb`.

- **`album_folder_template`**: album folder. Default `{{.releaseYear}} - {{.albumArtist}} - {{.albumTitle}}`.

  - `{{.albumArtist}}`: `Benjah` / `Tremonti`
  - `{{.albumID}}`: `36599795` / `2176030`
  - `{{.albumTitle}}`: `Heal` / `All I Was`
  - `{{.albumTrackCount}}`: `12` / `12`
  - `{{.releaseDate}}`: `2025-01-24` / `2012-07-17`
  - `{{.releaseYear}}`: `2025` / `2012`
  - `{{.type}}`: `album`

    ```yaml
    album_folder_template: "Artists/{{.albumArtist}}/{{.releaseYear}} - {{.albumTitle}}"
    ```

    Renders as `Artists/Benjah/2025 - Heal` or `Artists/Tremonti/2012 - All I Was`.

    Windows-style separators work too (`Music\\{{.albumArtist}}\\...`). Flat default:

    ```yaml
    album_folder_template: "{{.releaseYear}} - {{.albumArtist}} - {{.albumTitle}}"
    ```

    Renders as `2025 - Benjah - Heal` or `2012 - Tremonti - All I Was`.

- **`playlist_filename_template`**: playlist track file name. Default `{{.trackNumberPad}} - {{.trackArtist}} - {{.trackTitle}}`.

  Values from Zvuk [playlist 9037842](https://zvuk.com/playlist/9037842), first track (also tagged from its source album `7399068`):

  - `{{.albumArtist}}`: album that owns the track. `Los Angeles Philharmonic, Gustavo Dudamel`
  - `{{.albumID}}`: `7399068`
  - `{{.albumTitle}}`: `Tchaikovsky: The Nutcracker, Op. 71, TH 14 (Live at Walt Disney Concert Hall, Los Angeles / 2013)`
  - `{{.albumTrackCount}}`: tracks on that album. `24`
  - `{{.collectionTitle}}`: playlist title. `Зимние этюды`
  - `{{.playlistID}}`: `9037842`
  - `{{.playlistTitle}}`: `Зимние этюды`
  - `{{.playlistTrackCount}}`: `28`
  - `{{.recordLabel}}`: empty on this playlist track
  - `{{.releaseDate}}`: album date. `2018-11-16`
  - `{{.releaseYear}}`: `2018`
  - `{{.trackArtist}}`: `Los Angeles Philharmonic, Gustavo Dudamel`
  - `{{.trackCount}}`: tracks in the playlist. `28`
  - `{{.trackGenre}}`: `Classical`
  - `{{.trackID}}`: `58217111`
  - `{{.trackNumber}}`: position in the playlist. `1`
  - `{{.trackNumberPad}}`: `01`
  - `{{.trackTitle}}`: `Tchaikovsky: The Nutcracker, Op. 71, TH 14 / Act 1 - No. 2 March (Live at Walt Disney Concert Hall, Los Angeles / 2013)`
  - `{{.type}}`: `playlist`

    ```yaml
    playlist_filename_template: "{{.trackNumberPad}} - {{.trackArtist}} - {{.trackTitle}}"
    ```

    Renders as `01 - Los Angeles Philharmonic, Gustavo Dudamel - Tchaikovsky: The Nutcracker, Op. 71, TH 14 / Act 1 - No. 2 March (Live at Walt Disney Concert Hall, Los Angeles / 2013)`.

- **`audiobook_folder_template`**: Zvuk audiobook folder. Default `{{.publishYear}} - {{.audiobookAuthors}} - {{.audiobookTitle}}`.

  Values from [https://zvuk.com/abook/37364537](https://zvuk.com/abook/37364537):

  - `{{.audiobookID}}`: `37364537`
  - `{{.audiobookTitle}}`: `Темная Башня III: Бесплодные земли`
  - `{{.audiobookAuthors}}`: comma-separated. `Стивен Кинг`
  - `{{.audiobookTrackCount}}`: chapter count. `92`
  - `{{.audiobookPublisher}}`: brand name. `Vargtroms Studio`
  - `{{.audiobookPublisherName}}`: internal name. `ugcbooks`
  - `{{.audiobookCopyright}}`: `Vargtroms Studio`
  - `{{.audiobookDescription}}`: starts with `Действие книги начинается спустя семь недель после событий, описанных в романе «Извлечение троих».` (long; truncated here)
  - `{{.audiobookPerformers}}`: narrators, comma-separated. `Роман Волков`
  - `{{.audiobookGenres}}`: `Художественные произведения`
  - `{{.audiobookAgeLimit}}`: e.g. `12`, `16`, `18`. Here `18`
  - `{{.audiobookDuration}}`: seconds. `74967`
  - `{{.audiobookPublicationDate}}`: ISO 8601. `2025-02-01T09:21:20.21035+00:00`
  - `{{.publishYear}}`: year from publication date. `2025`
  - `{{.releaseDate}}`: `YYYY-MM-DD`. `2025-02-01`
  - `{{.releaseYear}}`: same as `publishYear`. `2025`
  - `{{.type}}`: `audiobook`

    ```yaml
    audiobook_folder_template: "{{.publishYear}} - {{.audiobookAuthors}} - {{.audiobookTitle}}"
    ```

    Renders as `2025 - Стивен Кинг - Темная Башня III: Бесплодные земли`.

- **`audiobook_chapter_filename_template`**: chapter file name. All audiobook-folder placeholders plus:

  First chapter of the same book:

  - `{{.trackTitle}}`: chapter title. `Книга первая. Джейк. Ужас в пригоршне праха. Часть 1. Медведь и кость 13`
  - `{{.trackID}}`: chapter id. `142497316`
  - `{{.trackNumber}}`: `1`
  - `{{.trackNumberPad}}`: `01`
  - `{{.trackCount}}`: chapter count. `92`
  - `{{.collectionTitle}}`: audiobook title. `Темная Башня III: Бесплодные земли`
  - `{{.trackArtist}}`: usually the authors. `Стивен Кинг`
  - `{{.trackGenre}}`: `Художественные произведения`

    ```yaml
    audiobook_chapter_filename_template: "{{.trackNumberPad}} - {{.trackTitle}}"
    ```

    Renders as `01 - Книга первая. Джейк. Ужас в пригоршне праха. Часть 1. Медведь и кость 13`.

- **`podcast_folder_template`**: Zvuk podcast folder. Default `{{.podcastAuthors}} - {{.podcastTitle}}`.

  Values from [https://zvuk.com/podcast/12891594](https://zvuk.com/podcast/12891594):

  - `{{.podcastID}}`: `12891594`
  - `{{.podcastTitle}}`: `Молодые и Глупые`
  - `{{.podcastAuthors}}`: hosts, comma-separated. `Молодые и Глупые`
  - `{{.podcastTrackCount}}`: episode count. `40`
  - `{{.podcastDescription}}`: starts with `Привет!` then a long about-text (site, telegram, etc.)
  - `{{.podcastCategory}}`: `Общество и культура`
  - `{{.podcastExplicit}}`: `true` when marked explicit. Here `true`
  - `{{.type}}`: `podcast`

    ```yaml
    podcast_folder_template: "{{.podcastAuthors}} - {{.podcastTitle}}"
    ```

    Renders as `Молодые и Глупые - Молодые и Глупые`.

- **`podcast_episode_filename_template`**: episode file name. All podcast-folder placeholders plus:

  First episode of that podcast:

  - `{{.episodePublicationDate}}`: `YYYY-MM-DD`. `2020-05-04`
  - `{{.episodeID}}`: `78203209`
  - `{{.episodeTitle}}`: `🖥 УДАЛЕНКА! (очень кстати)`
  - `{{.episodeNumber}}`: `1`
  - `{{.episodeNumberPad}}`: `01`
  - `{{.episodeDuration}}`: seconds. `5297`
  - `{{.trackTitle}}`: alias of `episodeTitle`. `🖥 УДАЛЕНКА! (очень кстати)`
  - `{{.trackID}}`: alias of `episodeID`. `78203209`
  - `{{.trackNumber}}`: alias of `episodeNumber`. `1`
  - `{{.trackNumberPad}}`: alias of `episodeNumberPad`. `01`
  - `{{.trackDuration}}`: alias of `episodeDuration`. `5297`

    ```yaml
    podcast_episode_filename_template: "{{.episodePublicationDate}} - {{.trackTitle}}"
    ```

    Renders as `2020-05-04 - 🖥 УДАЛЕНКА! (очень кстати)`.

## Download behavior

- **`download_lyrics`**: default `true`. Zvuk and Yandex write sidecar `.lrc` files when lyrics exist. Yandex also embeds the same text in tags.

    ```yaml
    download_lyrics: true
    ```

- **`replace_tracks`**: overwrite completed track files. Default `false`. Not the same as `resume` (that is byte-range continuation inside one run).

    ```yaml
    replace_tracks: false
    ```

- **`replace_covers`**: overwrite existing covers. Default `false`.

    ```yaml
    replace_covers: false
    ```

- **`replace_descriptions`**: overwrite audiobook/podcast description files. Default `false`.

    ```yaml
    replace_descriptions: false
    ```

- **`replace_lyrics`**: overwrite existing lyric files. Default `false`.

    ```yaml
    replace_lyrics: false
    ```

- **`log_level`**: `debug`, `info` (default), `warn`, `error`, `fatal`.

    ```yaml
    log_level: "info"
    ```

- **`download_speed_limit`**: per-track cap in **bytes per second**, not bits. `"1MB"`, `"115 KiB"`, `"500KB"`. Empty or `0` is unlimited. Concurrent jobs can exceed it in aggregate. TCP, TLS, the token-bucket burst, and proxies can still buffer. Pair with the nested HTTP keys when pacing lossless.

    ```yaml
    download_speed_limit: ""
    ```

    Unattended lossless near listening speed (example value, not a bitrate detector):

    ```yaml
    quality: 3
    max_concurrent_downloads: 1
    download_speed_limit: "115 KiB"
    ```

## Retries and concurrency

Two layers. Do not mix them up.

1. **API** (`api_retry_*`): Yandex metadata, covers, lyrics, MP3 link resolution; Zvuk stream metadata, including HTTP 418. Delay is random in `[api_min_retry_pause, api_max_retry_pause)`.
2. **Audio HTTP** (`zvuk_download_http` / `yandex_music_download_http`): the file body. Exponential backoff with equal jitter. `max_retries: 0` means no extra attempts.

`max_download_pause` is a pause between finished track jobs (`RandomPause(0, max)`). It is not a retry.

- **`api_retry_attempts_count`**: total API attempts including the first. Must be a positive integer. Default `5`. `1` means do not retry.

    ```yaml
    api_retry_attempts_count: 5
    ```

- **`api_min_retry_pause`**: lower bound of the API retry wait. Default `3s`.

    ```yaml
    api_min_retry_pause: "3s"
    ```

- **`api_max_retry_pause`**: upper bound (exclusive) of the API retry wait. Default `7s`.

    ```yaml
    api_max_retry_pause: "7s"
    ```

- **`max_download_pause`**: max pause between track jobs. Default `2s`.

    ```yaml
    max_download_pause: "2s"
    ```

- **`max_concurrent_downloads`**: tracks in flight. Default `1` (sequential, progress bars on). Values above `1` can trip rate limits or get the account slapped, and they turn progress bars off to avoid two bars fighting in the terminal.

    Use at your own risk. Sequential is what I actually test. If Zvuk bans you because you set this to 12, that is between you and Zvuk.

    ```yaml
    max_concurrent_downloads: 1
    ```

## Audio HTTP

`zvuk_download_http` and `yandex_music_download_http` are separate clients for audio bodies. API, covers, and lyrics do not use them. Same nested keys and defaults.

Interrupted audio is retried on `unexpected EOF`, connection resets, timeouts, and HTTP 408/429/500/502/503/504. Auth errors, cancellation, and local file errors are not retried.

`max_retries` must be 0-100. `retry_initial_delay` must be positive and `<= retry_max_delay`. `receive_buffer` uses the same size syntax as `download_speed_limit` (`256KiB`, `115 KiB`, `0`) and must parse to 0-2147483647 bytes. Negative durations are rejected.

`http1_only` and `receive_buffer` apply only when `download_speed_limit` or `--speed-limit` is positive. There is no `ZVUK_GRABBER_RCVBUF` environment variable.

Nested keys:

- **`timeout`**: total deadline per HTTP attempt including the body. Default `0s` (disabled), so a paced FLAC can last longer than 60s
- **`dial_timeout`**: TCP connect. Default `30s`
- **`tls_handshake_timeout`**: TLS handshake. Default `10s`
- **`response_header_timeout`**: wait for headers. Default `60s`
- **`read_idle_timeout`**: stalled socket read, not time spent in the speed limiter. Default `60s`. `0s` disables it
- **`receive_buffer`**: requested `SO_RCVBUF` when paced. Same size syntax as `download_speed_limit`. Default `"256KiB"` (262144 bytes). `0` or empty keeps the OS default. Linux may double or clamp it. Windows and other Unix kernels treat it differently. An unsupported socket option shows up as a connection error
- **`http1_only`**: HTTP/1.1 when paced, so HTTP/2 cannot read-ahead past the limiter. Default `true`. Set `false` to allow HTTP/2
- **`max_retries`**: extra attempts per audio URL, shared by header and body failures. `0` disables retries. Default `3`
- **`retry_initial_delay`**: first exponential backoff. Equal jitter picks `[delay/2, delay)`. Default `1s`
- **`retry_max_delay`**: cap on backoff and accepted `Retry-After`. If the server asks for longer, the transfer fails. Default `30s`
- **`resume`**: `Range` / `If-Range` continuation inside the current run. Needs a known size and a strong ETag, or Last-Modified at least 60s older than Date. The next response must be `206` with matching range, size, and validator. If the CDN ignores Range, rejects it, or changes the file, the same retry budget restarts from byte 0. Partial files are removed after cancel or exhausted retries. Completed files are skipped on the next run unless `replace_tracks` is true. Yandex may try another API URL with a fresh budget and file. A failed transfer is a failure, not an MP3 downgrade. Yandex FLAC is paced/resumed on the encrypted source, spooled, then decrypted. Default `true`

```yaml
zvuk_download_http:
  timeout: "0s"
  dial_timeout: "30s"
  tls_handshake_timeout: "10s"
  response_header_timeout: "60s"
  read_idle_timeout: "60s"
  receive_buffer: "256KiB"
  http1_only: true
  max_retries: 3
  retry_initial_delay: "1s"
  retry_max_delay: "30s"
  resume: true

yandex_music_download_http:
  timeout: "0s"
  dial_timeout: "30s"
  tls_handshake_timeout: "10s"
  response_header_timeout: "60s"
  read_idle_timeout: "60s"
  receive_buffer: "256KiB"
  http1_only: true
  max_retries: 3
  retry_initial_delay: "1s"
  retry_max_delay: "30s"
  resume: true
```
