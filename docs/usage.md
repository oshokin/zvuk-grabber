# Usage

Install, sign in, then pass URLs. YAML keys: [Configuration](configuration.md).

Zvuk URLs cover tracks, releases (albums), playlists, artists, audiobooks, and podcasts. Yandex Music URLs cover tracks, albums, legacy user playlists, and UUID playlists. Artist, audiobook, and podcast links on Yandex are rejected.

## Install a binary

Pre-built binaries for macOS, Windows, and Linux (`amd64` and `arm64`) are on the [Releases page](https://github.com/oshokin/zvuk-grabber/releases).

1. Download the archive for your OS. Replace `1.0.0` with the latest version:

    - macOS: `zvuk-grabber_1.0.0_darwin_amd64.tar.gz` (Intel) or `zvuk-grabber_1.0.0_darwin_arm64.tar.gz` (Apple Silicon)
    - Windows: `zvuk-grabber_1.0.0_windows_amd64.zip` or `zvuk-grabber_1.0.0_windows_arm64.zip`
    - Linux: `zvuk-grabber_1.0.0_linux_amd64.tar.gz` or `zvuk-grabber_1.0.0_linux_arm64.tar.gz`

2. Extract it. Inside you get:

    - `zvuk-grabber` (`zvuk-grabber.exe` on Windows)
    - `.zvuk-grabber.example.yaml`
    - `LICENSE`
    - `README.md`
    - `docs/`

    ```bash
    tar -xvzf zvuk-grabber_1.0.0_darwin_amd64.tar.gz  # macOS/Linux
    unzip zvuk-grabber_1.0.0_windows_amd64.zip        # Windows
    ```

3. Copy `.zvuk-grabber.example.yaml` to `.zvuk-grabber.yaml` in the directory you will run from. The tool looks for `.zvuk-grabber.yaml` in the current working directory unless you pass `-c`. If the file is missing, defaults are used and tokens stay empty.

4. On Linux and macOS: `chmod +x zvuk-grabber`.

## Sign in

You only need a token for the provider you actually download from.

### Zvuk browser login (the easy way)

I've wanted to automate the authentication cookie extraction for ages. UI automation is usually the part where you poke a website like a black box and hope. Like a blind chicken in the dark.

It works. Mostly.

```bash
zvuk-grabber auth zvuk login
```

That command:

1. Opens Chrome/Chromium with stealth mode
2. Goes to the Zvuk homepage first (OAuth needs that origin)
3. Waits while you log in (phone number + SMS)
4. Wiggles the mouse, scrolls, and inserts random delays so it looks less like a robot
5. Notices when login and OAuth finish
6. Reads the `auth` cookie
7. Writes it to `zvuk_auth_token` in `.zvuk-grabber.yaml`
8. Closes the browser

Anti-bot stack:

1. Stealth mode ([go-rod/stealth](https://github.com/go-rod/stealth)): hides `navigator.webdriver`, patches automation fingerprints, spoofs plugins/permissions, hides CDP
2. Human-ish waiting: random mouse moves, occasional scroll, 500ms-2s gaps, random pauses
3. Fresh incognito profile each login, no leftover cookies
4. OAuth starts on `zvuk.com`, not the login page. If the callback page dies on CORS, the tool sends you back to the homepage and reads the cookie instead of hammering rate-limited APIs

On Windows 10 with ESET you might get a "virus" popup. It is not a virus. This repo and `go-rod` are source-available. Ignore it or whitelist the binary.

If login hangs:

1. Set `log_level: debug` in `.zvuk-grabber.yaml`
2. Run `zvuk-grabber auth zvuk login` again
3. Open an issue and paste the debug log (not the token)

And if the moon phase is in the right wavelength of light and Mercury's retrograde isn't too retrograde, I might take a look.

Known issues:

- CORS on Zvuk's OAuth callback: the tool redirects to the main page
- Rate limits if you retry too often
- Chrome/Chromium is the tested browser. Firefox is untested
- Windows cleanup warnings about the temp profile: Chrome still holding file locks. Harmless

### Yandex Music browser login

```bash
zvuk-grabber auth yandex login
```

Opens a visible go-rod window at `https://music.yandex.ru/`. Log in there. The command watches the session and writes the OAuth token to `yandex_music_token`. The token is never printed to logs.

### Manual tokens

If the browser path fails, or you already have tokens:

1. `zvuk_auth_token`: log in to Zvuk, open [https://zvuk.com/api/v2/tiny/profile](https://zvuk.com/api/v2/tiny/profile), copy `$.result.profile.token`
2. `yandex_music_token`: paste an existing Yandex Music OAuth token
3. Save both in `.zvuk-grabber.yaml`:

   ```yaml
   zvuk_auth_token: "your_token_here"
   yandex_music_token: "your_token_here"
   ```

There is no `auth_token` key. Old configs that still use it are ignored.

## Build from source

Need this only to change the code or skip release binaries. Go is `1.27.1` in `go.mod`.

1. Install [Go](https://go.dev/dl/)
2. Install [Task](https://taskfile.dev/installation/)
3. Clone and build:

    ```bash
    git clone https://github.com/oshokin/zvuk-grabber.git
    cd zvuk-grabber
    task build
    ```

The binary lands in `bin/`. `task test` and `task lint` are there if you are about to send a PR.

## Download from Zvuk

Hosts: `zvuk.com` and `*.zvuk.com`. Paths must end at the numeric id. Query strings (`?utm_source=...`) do not match, so the URL is treated as unknown. Strip the junk after `?` before you paste.

```bash
# albums (releases)
zvuk-grabber https://zvuk.com/release/38858441 https://zvuk.com/release/52524287

# tracks (saved like album tracks, with a folder and cover when the metadata has one)
zvuk-grabber https://zvuk.com/track/61318527 https://zvuk.com/track/77437343

# playlist
zvuk-grabber https://zvuk.com/playlist/9037842

# artist discography
zvuk-grabber https://zvuk.com/artist/3196437

# audiobook
zvuk-grabber https://zvuk.com/abook/37364537

# podcast
zvuk-grabber https://zvuk.com/podcast/12891594
```

A Zvuk URL with extra path (`/track/123/details`) is unknown. Trailing slash after the id is also unknown.

## Download from Yandex Music

Hosts: `music.yandex.*` (for example `music.yandex.ru`). Query strings are allowed. A host/path without `https://` is accepted.

```bash
# track
zvuk-grabber https://music.yandex.ru/album/12460810/track/72443010

# album
zvuk-grabber https://music.yandex.ru/album/2985370

# legacy playlist
zvuk-grabber https://music.yandex.ru/users/yamusic-daily/playlists/1000

# UUID playlist (optional two-letter prefix before the UUID)
zvuk-grabber https://music.yandex.ru/playlists/018f7f8a-90fb-7f72-89f1-cd5f6c8d4cb1
```

Artist / radio / other Yandex paths return `unsupported Yandex Music URL` and fail the run.

## Mix providers

```bash
zvuk-grabber \
  https://zvuk.com/release/38858441 \
  https://music.yandex.ru/album/2985370 \
  https://zvuk.com/track/61318527
```

Zvuk URLs run first, then Yandex, in the order they appeared. An unknown URL is a failure even if the rest downloaded. If you Ctrl+C during Zvuk, Yandex does not start.

## URL lists

If an argument is a file on disk, it is read as a URL list. It does not have to end in `.txt`. Empty lines and lines starting with `#` are skipped. Duplicates are dropped. You can mix files and URLs:

```bash
zvuk-grabber urls.txt another-list.txt https://zvuk.com/release/38858441
```

If the path does not exist, it is treated as a URL.

## Flags

Flags override YAML for that run. They are applied only when you actually pass them. The Cobra default for `-q` is `1`, but that value is unused unless you set `-q`. Without the flag, YAML `quality` wins (default `3`).

```bash
zvuk-grabber [flags] {urls}
```

- `-c, --config <path>`: config file (default `.zvuk-grabber.yaml` in the current directory)
- `-q, --quality <1-3>`: preferred quality. `1` = MP3 128 Kbps, `2` = MP3 320 Kbps, `3` = FLAC lossless, then MP3 if FLAC is missing
- `-m, --min-quality <0-3>`: skip tracks below this. `0` = no filter. Must be `<= quality`
- `-o, --output <path>`: download root (created if needed)
- `-l, --lyrics`: set `download_lyrics` to true. YAML already defaults to true, so this matters when the file has `false`
- `-s, --speed-limit <speed>`: per-track cap in bytes/second (`115 KiB`, `500KB`, `1MB`). Empty YAML / omitted flag = unlimited
- `-n, --dry-run`: resolve metadata and print what would be written, without writing audio

```bash
zvuk-grabber -q 3 https://zvuk.com/release/38858441
zvuk-grabber -o "/Music/Zvuk" -l https://zvuk.com/release/52524287
zvuk-grabber -s 1MB https://zvuk.com/track/61318527
zvuk-grabber -n https://zvuk.com/release/38858441 https://music.yandex.ru/album/2985370
zvuk-grabber -q 3 -o "/Music" -l -s 2MB https://zvuk.com/release/38858441
```

## Commands

- `zvuk-grabber {urls}`: download
- `zvuk-grabber auth zvuk login`
- `zvuk-grabber auth yandex login`
- `zvuk-grabber version`
- `zvuk-grabber help`

## Stop a run

`SIGINT` (Ctrl+C), `SIGTERM`, and `SIGHUP` cancel the context. Partial audio for the current transfer is deleted after cancel. Completed files stay. Exit status is `1`.

## Exit status

- `0`: no error. Intentional skips and dry-run previews still count as success
- `1`: invalid config, missing token, unsupported URL, recorded download failure, or interrupt

A batch can finish remaining items and still return `1`. Cover/lyrics warnings are not always enough to fail a run. A summary is printed either way.
