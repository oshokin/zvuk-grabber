# Zvuk Grabber 🎵

[Zvuk (Звук)](https://zvuk.com/) grabber written in Go.\
This tool allows you to download artists, albums, tracks, and playlists from Zvuk.

* * *

## Quick Start 🚀

1. **Download the Latest Release**:  
   Grab the pre-built binary for your OS from the [Releases page](https://github.com/oshokin/zvuk-grabber/releases).

2. **Extract the Archive**:  
   Just extract the archive! It already has everything you need inside.
   - For macOS/Linux:

     ```bash
     tar -xvzf zvuk-grabber_1.0.0_darwin_amd64.tar.gz
     ```

   - For Windows:

     ```bash
     unzip zvuk-grabber_1.0.0_windows_amd64.zip
     ```

3. **Set Up Authentication Token**:  

   **Option 1: Automatic Browser Login (Recommended)**

   Run the interactive login command:

   ```bash
   zvuk-grabber auth login
   ```

   This will:
   - Open a browser window
   - Let you log in manually (phone number + SMS code)
   - Automatically extract and save your auth token
   - Update your `.zvuk-grabber.yaml` configuration

   **Option 2: Manual Token Extraction**

   Open the `.zvuk-grabber.yaml` file and set your `auth_token`.\
   You can obtain it by logging into [Zvuk's API](https://zvuk.com/api/v2/tiny/profile) and locating the token in the JSON response using the JSON path `$.result.profile.token`.

4. **Run the Tool**:  
   - **Linux/macOS**:

     ```bash
     chmod +x zvuk-grabber  # Make it executable
     ./zvuk-grabber         # Let it rip!
     ```

   - **Windows**:

     ```bash
     zvuk-grabber           # Run the executable
     ```

5. **Enjoy Your Music!** 🎶  
   Start downloading your favorite tracks, albums, and playlists.

* * *

## Installation 🛠️

### Download Pre-built Binaries

Pre-built binaries for **macOS**, **Windows**, and **Linux** (for both `arm64` and `amd64` architectures) are available on the [Releases page](https://github.com/oshokin/zvuk-grabber/releases).

1. **Download the Correct Binary**:  
    Go to the [Releases page](https://github.com/oshokin/zvuk-grabber/releases) and download the appropriate binary for your operating system and architecture.\
    Replace `1.0.0` in the filenames below with the latest version number:

    - **macOS**:
      - `zvuk-grabber_1.0.0_darwin_amd64.tar.gz` for Intel-based Macs (`amd64`).
      - `zvuk-grabber_1.0.0_darwin_arm64.tar.gz` for Apple Silicon Macs (`arm64`).
    - **Windows**:
      - `zvuk-grabber_1.0.0_windows_amd64.zip` for 64-bit Windows (`amd64`).
      - `zvuk-grabber_1.0.0_windows_arm64.zip` for ARM-based Windows (`arm64`).
    - **Linux**:
      - `zvuk-grabber_1.0.0_linux_amd64.tar.gz` for 64-bit Linux (`amd64`).
      - `zvuk-grabber_1.0.0_linux_arm64.tar.gz` for ARM-based Linux (`arm64`).

2. **Extract the Bundle**:  
    Each bundle contains the following files:
    - `zvuk-grabber` (or `zvuk-grabber.exe` for Windows): The main executable.
    - `.zvuk-grabber.yaml`: The configuration file.
    - `LICENSE`: The license file.
    - `README.md`: The documentation.

    Extract the bundle to your desired location:

    ```bash
    tar -xvzf zvuk-grabber_1.0.0_darwin_amd64.tar.gz  # For macOS/Linux
    unzip zvuk-grabber_1.0.0_windows_amd64.zip        # For Windows
    ```

3. **Set Up Authentication Token**:  

    **Automatic (Recommended)**:

    ```bash
    zvuk-grabber auth login
    ```

    **Manual**: Open the `.zvuk-grabber.yaml` file and set your authentication token in the `auth_token` field.\
    You can obtain the token by logging into [Zvuk's API](https://zvuk.com/api/v2/tiny/profile) and locating the token using the JSON path `$.result.profile.token`.

4. **Run the Binary**:  
    - **Linux/macOS**:  
      Make the binary executable and run it:

      ```bash
      chmod +x zvuk-grabber  # Make the file executable
      ./zvuk-grabber         # Run the tool
      ```

    - **Windows**:  
      Simply run the executable:

      ```bash
      zvuk-grabber
      ```

### Building from Source (Optional) 🛠️

If you want to modify the code or build the binary yourself, you'll need the following prerequisites:

1. **Install Go**:  
    Download and install Go from the [official website](https://go.dev/dl/).

2. **Install Task**:  
    [Task](https://taskfile.dev/) is a task runner/build tool that simplifies the build process.\
    Install it by following the instructions on their [official website](https://taskfile.dev/installation/).

3. **Clone the Repository**:  
    Clone the repository to your local machine:

    ```bash
    git clone https://github.com/oshokin/zvuk-grabber.git
    cd zvuk-grabber
    ```

4. **Build the Binary**:  
    Use Task to build the binary:

    ```bash
    task build
    ```

    The compiled binary will be stored in the `bin/` directory.

* * *

## Authentication 🔐

### Browser-Based Login (The Easy Way)

**I've wanted to automate the authentication cookie extraction for ages!** But as we all know, UI/UX automation is usually painful because we approach the website like a black box and poke it with a stick hoping to discover the right behavior and side effects. Like a blind chicken in the dark, basically.

**But IT WORKS!** Well... mostly.

#### The Good News

Run this command and watch the magic happen:

```bash
zvuk-grabber auth login
```

This will:

1. Open a browser window (Chrome/Chromium) with **stealth mode** enabled
2. Navigate to the Zvuk homepage (to establish proper origin for OAuth)
3. Wait for you to manually log in (phone number + SMS code)
4. **Simulate human behavior** while waiting (mouse movements, scrolling, random delays)
5. Detect when login completes and OAuth flow finishes
6. Extract the `auth` cookie from your browser
7. Save it to `.zvuk-grabber.yaml`
8. Close the browser and celebrate

#### Anti-Bot Detection Stack

The tool employs multiple techniques to evade bot detection:

1. **Stealth Mode** ([go-rod/stealth](https://github.com/go-rod/stealth))
   - Hides `navigator.webdriver` flag
   - Patches browser automation signatures
   - Spoofs plugin lists and permissions
   - Makes CDP (Chrome DevTools Protocol) invisible

2. **Human Behavior Simulation**
   - Random mouse movements across the page
   - Occasional scrolling (up/down)
   - Variable timing between actions (500ms-2s)
   - Random pauses to mimic reading/thinking
   - Randomized interaction patterns

3. **Fresh Browser Profile**
   - Each login uses a temporary incognito profile
   - No persistent cookies or history between sessions
   - Clean slate helps avoid detection patterns

4. **Smart OAuth Flow**
   - Starts on `zvuk.com` domain (not login page directly)
   - Avoids CORS errors during OAuth callback
   - Bypasses broken automatic redirects manually
   - Detects auth cookie directly without rate-limited API calls

#### The Bad News (Windows Edition)

On **Windows 10 with ESET Security**, you might get a fun notification that our code is infected with some virus. **It's not.** Both this project and `go-rod` have source code available - feel free to audit it yourself.

**TL;DR: It's a false positive. Ignore the warning or whitelist the application.**

#### Troubleshooting Login Issues

If the login process gets stuck or fails:

1. **Enable debug logging** in `.zvuk-grabber.yaml`:

   ```yaml
   log_level: debug
   ```

2. **Run the command again**:

   ```bash
   zvuk-grabber auth login
   ```

3. **Create an issue** with the debug output

And if the moon phase is in the right wavelength of light and Mercury's retrograde isn't too retrograde, I might just take a look at what's going on in your code.

#### Known Issues

- **CORS/API Issues**: Zvuk's OAuth callback sometimes fails with CORS errors. The tool now automatically bypasses this by manually redirecting to the main page.
- **Rate Limiting**: If you try too many times, Zvuk might rate-limit you. The tool now minimizes API calls during login to avoid this.
- **Browser Compatibility**: Works best with Chrome/Chromium. Firefox might work but is untested.
- **Cleanup Warnings**: You might see warnings about temp directory cleanup on Windows. This is normal and non-critical - Chrome takes time to release file locks.

### Manual Token Extraction (The Old-School Way)

If the browser automation fails or you prefer doing things manually:

1. **Log in to Zvuk** in your browser
2. **Navigate to** [https://zvuk.com/api/v2/tiny/profile](https://zvuk.com/api/v2/tiny/profile)
3. **Find the token** in the JSON response at `$.result.profile.token`
4. **Copy it** to `.zvuk-grabber.yaml`:

   ```yaml
   auth_token: "your_token_here"
   ```

* * *

## Usage 🎧

### Downloading Content

1. **Download Albums**:  
    To download one or more albums, provide the album URLs as arguments:

    ```bash
    zvuk-grabber https://zvuk.com/release/36599795 https://zvuk.com/release/37212880
    ```

2. **Download Tracks**:  
    To download individual tracks, provide the track URLs.\
    The tracks will be organized as if they were part of an album, with a folder and cover art:

    ```bash
    zvuk-grabber https://zvuk.com/track/67856297 https://zvuk.com/track/51397074 https://zvuk.com/track/63391919 https://zvuk.com/track/106773860 https://zvuk.com/track/114947212
    ```

3. **Download Playlists**:  
    To download a playlist, provide the playlist URL:

    ```bash
    zvuk-grabber https://zvuk.com/playlist/9037842
    ```

4. **Download Artists**:  
    To download an artist's entire discography, provide the artist URL:

    ```bash
    zvuk-grabber https://zvuk.com/artist/3196437
    ```

5. **Download Audiobooks**:  
    To download audiobooks, provide the audiobook URL:

    ```bash
    zvuk-grabber https://zvuk.com/abook/37364537
    ```

6. **Download Podcasts**:  
    To download podcasts, provide the podcast URL:

    ```bash
    zvuk-grabber https://zvuk.com/podcast/12891594
    ```

7. **Using Text Files**:  
    You can also provide text files containing URLs (one per line):

    ```bash
    zvuk-grabber 1.txt 2.txt
    ```

### Command-Line Flags

You can override configuration settings using command-line flags:

```bash
zvuk-grabber [flags] {urls}
```

**Available flags:**

- `-c, --config <path>` - Path to configuration file (default: `.zvuk-grabber.yaml`)
- `-q, --quality <1-3>` - Preferred audio quality:
  - `1` = MP3, 128 Kbps
  - `2` = MP3, 320 Kbps
  - `3` = FLAC, 16-bit/44.1kHz
- `-m, --min-quality <1-3>` - Minimum acceptable quality (tracks below this will be skipped):
  - `1` = MP3, 128 Kbps
  - `2` = MP3, 320 Kbps
  - `3` = FLAC
  - `0` = No filtering (default)
- `-o, --output <path>` - Output directory for downloads
- `-l, --lyrics` - Download lyrics if available
- `-s, --speed-limit <speed>` - Download speed limit (e.g., `500KB`, `1MB`, `1.5MB`)

**Examples:**

```bash
# Download album in FLAC format
zvuk-grabber -f 3 https://zvuk.com/release/3393328

# Download with custom output directory and lyrics
zvuk-grabber -o "/Music/Zvuk" -l https://zvuk.com/release/5895112

# Download with speed limit
zvuk-grabber -s 1MB https://zvuk.com/release/8045705

# Combine multiple flags
zvuk-grabber -f 3 -o "/Music" -l -s 2MB https://zvuk.com/release/38858441
```

### Available Commands

- `zvuk-grabber {urls}` - Download content from URLs
- `zvuk-grabber auth login` - Interactive browser-based authentication
- `zvuk-grabber version` - Show version information
- `zvuk-grabber help` - Show help information

* * *

## Configuration ⚙️

The default configuration is already set in the `.zvuk-grabber.yaml` file.\
You only need to modify it if you want to customize the behavior.\
Key options include:

### Authentication

- **`auth_token`**: Your Zvuk API authentication token.\
    **Easiest way**: Run `zvuk-grabber auth login` to automatically extract it.\
    **Manual way**: Log in to [Zvuk's API](https://zvuk.com/api/v2/tiny/profile) and locate the token using the JSON path `$.result.profile.token`.  
    Example:

    ```yaml
    auth_token: "a3f8e7b2c5d946f1a0b9e8d7c6f5e4a2"
    ```

### Audio Quality

- **`quality`**: Preferred audio quality for downloaded files.\
    Available options:
  - `1` = MP3, 128 Kbps (standard quality)
  - `2` = MP3, 320 Kbps (high quality)
  - `3` = FLAC, 16/24-bit (lossless quality)
    Example:

    ```yaml
    quality: 3
    ```

- **`min_quality`**: Minimum acceptable quality (tracks below this will be skipped).\
    Available options:
  - `0` = No filtering (accept any quality) - default
  - `1` = Skip tracks only available in MP3 128 Kbps or lower
  - `2` = Skip tracks only available in MP3 320 Kbps or lower (FLAC only)
  - `3` = Skip tracks only available in FLAC or lower (impossible - FLAC is max)

    **Example 1** - Download FLAC preferred, but accept MP3 320 minimum:

    ```yaml
    quality: 3
    min_quality: 2
    ```

    **Example 2** - FLAC only, skip everything else:

    ```yaml
    quality: 3
    min_quality: 3
    ```

    **Note**: `min_quality` must be less than or equal to `quality`.

- **`min_duration`**: Minimum acceptable track duration (tracks shorter than this will be skipped).\
    Use duration strings like `30s`, `1m`, `1m30s`.\
    Empty string = no filtering (default).

    **Example uses**:
  - Skip intros/interludes/skits (common in hip-hop albums)
  - Skip sound effects and ambient noise
  - Filter out incomplete/sample tracks

    ```yaml
    min_duration: "30s"  # Skip tracks shorter than 30 seconds
    ```

- **`max_duration`**: Maximum acceptable track duration (tracks longer than this will be skipped).\
    Use duration strings like `10m`, `15m`, `1h`.\
    Empty string = no filtering (default).

    **Example uses**:
  - Skip DJ mixes and extended live versions
  - Skip very long classical pieces or ambient tracks
  - Filter out podcasts/interviews mistakenly tagged as music

    ```yaml
    max_duration: "10m"  # Skip tracks longer than 10 minutes
    ```

    **Combined Example** - Only download "normal" songs (30s to 10min):

    ```yaml
    min_duration: "30s"
    max_duration: "10m"
    ```

    **Note**: If both are set, `max_duration` must be greater than `min_duration`.

### Output Settings

- **`output_path`**: Directory where downloaded files will be saved.\
    You can specify either a relative path (e.g., `"zvuk downloads"`) or an absolute path (e.g., `"C:/Music"`).\
    Example:

    ```yaml
    output_path: "zvuk downloads"
    ```

- **`create_folder_for_singles`**: Whether to create a separate folder for single tracks (tracks not part of an album).\
    If set to `false`, single tracks will be saved directly in the output directory.\
    Example:

    ```yaml
    create_folder_for_singles: false
    ```

- **`max_folder_name_length`**: Maximum length for folder names created by the application.\
    This ensures folder names remain readable and compatible across different operating systems.\
    Set to `0` to avoid cutting folder names.\
    Example:

    ```yaml
    max_folder_name_length: 100
    ```

### File Naming Templates

- **`track_filename_template`**: Track file naming format.\
    Available placeholders:
  - `{{.albumArtist}}`: Primary artist(s) of the album.
  - `{{.albumID}}`: Unique identifier for the album.
  - `{{.albumTitle}}`: Title of the album.
  - `{{.albumTrackCount}}`: Total number of tracks in the album.
  - `{{.collectionTitle}}`: Title of the album.
  - `{{.recordLabel}}`: Name of the record label.
  - `{{.releaseDate}}`: Full release date of the album (YYYY-MM-DD format).
  - `{{.releaseYear}}`: Year the album was released.
  - `{{.trackArtist}}`: Artist(s) of the track.
  - `{{.trackCount}}`: Total number of tracks in the album.
  - `{{.trackGenre}}`: Genre(s) of the track.
  - `{{.trackID}}`: Unique identifier for the track.
  - `{{.trackNumber}}`: Track number within the album (without leading zeros).
  - `{{.trackNumberPad}}`: Track number with two-digit padding (e.g., 01, 02).
  - `{{.trackTitle}}`: Track title.
  - `{{.type}}`: "album" (used to differentiate album tracks).

    Example:

    ```yaml
    track_filename_template: "{{.trackNumberPad}} - {{.trackTitle}}"
    ```

- **`album_folder_template`**: Album folder naming format.\
    Available placeholders:
  - `{{.albumArtist}}`: Primary artist(s) of the album.
  - `{{.albumID}}`: Unique identifier for the album.
  - `{{.albumTitle}}`: Title of the album.
  - `{{.albumTrackCount}}`: Total number of tracks in the album.
  - `{{.releaseDate}}`: Full release date of the album (YYYY-MM-DD format).
  - `{{.releaseYear}}`: Year the album was released.
  - `{{.type}}`: "album" (used to differentiate albums from playlists).

    **Folder Structure Tip**: Use `/` or `\` in your template to create nested subfolders.\
    Both separators work universally across operating systems - we'll automatically convert them to your system's native format.\
    Examples:

    ```yaml
    # Unix-style path (recommended)
    album_folder_template: "Artists/{{.albumArtist}}/{{.releaseYear}} - {{.albumTitle}}"

    # Windows-style path (escaped backslash)
    album_folder_template: "Music\\{{.albumArtist}}\\{{.releaseYear}} - {{.albumTitle}}"

    # Flat structure alternative
    album_folder_template: "{{.releaseYear}} - {{.albumArtist}} - {{.albumTitle}}"
    ```

- **`playlist_filename_template`**: Playlist file naming format.\
    Available placeholders:
  - `{{.albumArtist}}`: Primary artist(s) of the album containing the track.
  - `{{.albumID}}`: Unique identifier for the album containing the track.
  - `{{.albumTitle}}`: Title of the album containing the track.
  - `{{.albumTrackCount}}`: Total number of tracks in the album.
  - `{{.collectionTitle}}`: Title of the playlist.
  - `{{.playlistID}}`: Unique identifier for the playlist.
  - `{{.playlistTitle}}`: Title of the playlist.
  - `{{.playlistTrackCount}}`: Total number of tracks in the playlist.
  - `{{.recordLabel}}`: Name of the record label.
  - `{{.releaseDate}}`: Full release date of the album containing the track (YYYY-MM-DD format).
  - `{{.releaseYear}}`: Year the album containing the track was released.
  - `{{.trackArtist}}`: Artist(s) of the track.
  - `{{.trackCount}}`: Total number of tracks in the playlist.
  - `{{.trackGenre}}`: Genre(s) of the track.
  - `{{.trackID}}`: Unique identifier for the track.
  - `{{.trackNumber}}`: Track number within the playlist (without leading zeros).
  - `{{.trackNumberPad}}`: Track number with two-digit padding (e.g., 01, 02).
  - `{{.trackTitle}}`: Track title.
  - `{{.type}}`: "playlist" (used to differentiate playlists from albums).

    Example:

    ```yaml
    playlist_filename_template: "{{.trackNumberPad}} - {{.trackArtist}} - {{.trackTitle}}"
    ```

- **`audiobook_folder_template`**: Audiobook folder naming format.\
    Available placeholders:
  - `{{.audiobookID}}`: Unique identifier for the audiobook.
  - `{{.audiobookTitle}}`: Title of the audiobook.
  - `{{.audiobookAuthors}}`: Author(s) of the audiobook (comma-separated).
  - `{{.audiobookTrackCount}}`: Total number of chapters.
  - `{{.audiobookPublisher}}`: Publisher brand name.
  - `{{.audiobookPublisherName}}`: Publisher internal name.
  - `{{.audiobookCopyright}}`: Copyright holder.
  - `{{.audiobookDescription}}`: Audiobook description.
  - `{{.audiobookPerformers}}`: Narrator/performer names (comma-separated).
  - `{{.audiobookGenres}}`: Genre(s) (comma-separated).
  - `{{.audiobookAgeLimit}}`: Age rating (e.g., 12, 16, 18).
  - `{{.audiobookDuration}}`: Total duration in seconds.
  - `{{.audiobookPublicationDate}}`: Full publication date (ISO 8601 format).
  - `{{.publishYear}}`: Year of publication (extracted from publicationDate).
  - `{{.releaseDate}}`: Publication date in YYYY-MM-DD format.
  - `{{.releaseYear}}`: Same as publishYear (for consistency with albums).
  - `{{.type}}`: "audiobook" (used to differentiate audiobooks).

    Example:

    ```yaml
    audiobook_folder_template: "{{.publishYear}} - {{.audiobookAuthors}} - {{.audiobookTitle}}"
    ```

- **`audiobook_chapter_filename_template`**: Audiobook chapter file naming format.\
    Available placeholders (includes all audiobook folder placeholders plus):
  - `{{.trackTitle}}`: Chapter title.
  - `{{.trackID}}`: Unique identifier for the chapter.
  - `{{.trackNumber}}`: Chapter number (without leading zeros).
  - `{{.trackNumberPad}}`: Chapter number with two-digit padding (e.g., 01, 02).
  - `{{.trackCount}}`: Total number of chapters.
  - `{{.collectionTitle}}`: Audiobook title.
  - `{{.trackArtist}}`: Author(s) of the chapter (usually same as audiobook authors).
  - `{{.trackGenre}}`: Genre of the chapter/audiobook.

    Example:

    ```yaml
    audiobook_chapter_filename_template: "{{.trackNumberPad}} - {{.trackTitle}}"
    ```

- **`podcast_folder_template`**: Podcast folder naming format.\
    Available placeholders:
  - `{{.podcastID}}`: Unique identifier for the podcast.
  - `{{.podcastTitle}}`: Title of the podcast.
  - `{{.podcastAuthors}}`: Host/author(s) of the podcast (comma-separated).
  - `{{.podcastTrackCount}}`: Total number of episodes.
  - `{{.podcastDescription}}`: Podcast description.
  - `{{.podcastCategory}}`: Podcast category (e.g., "Общество и культура").
  - `{{.podcastExplicit}}`: "true" if podcast contains explicit content.
  - `{{.type}}`: "podcast" (used to differentiate podcasts).

    Example:

    ```yaml
    podcast_folder_template: "{{.podcastAuthors}} - {{.podcastTitle}}"
    ```

- **`podcast_episode_filename_template`**: Podcast episode file naming format.\
    Available placeholders (includes all podcast folder placeholders plus):
  - `{{.episodePublicationDate}}`: Publication date in YYYY-MM-DD format (e.g., "2020-05-04").
  - `{{.episodeID}}`: Unique identifier for the episode.
  - `{{.episodeTitle}}`: Episode title.
  - `{{.episodeNumber}}`: Episode number (without leading zeros).
  - `{{.episodeNumberPad}}`: Episode number with two-digit padding (e.g., 01, 02).
  - `{{.episodeDuration}}`: Episode duration in seconds.
  - `{{.trackTitle}}`: Episode title (alias for episodeTitle).
  - `{{.trackID}}`: Unique identifier for the episode (alias for episodeID).
  - `{{.trackNumber}}`: Episode number (alias for episodeNumber).
  - `{{.trackNumberPad}}`: Episode number padded (alias for episodeNumberPad).
  - `{{.trackDuration}}`: Episode duration (alias for episodeDuration).

    Example:

    ```yaml
    podcast_episode_filename_template: "{{.episodePublicationDate}} - {{.trackTitle}}"
    ```

### Download Behavior

- **`download_lyrics`**: Whether to download lyrics for tracks (if available).\
    Example:

    ```yaml
    download_lyrics: true
    ```

- **`replace_tracks`**: Whether to overwrite existing track files.\
    Example:

    ```yaml
    replace_tracks: false
    ```

- **`replace_covers`**: Whether to overwrite existing cover art files.\
    Example:

    ```yaml
    replace_covers: false
    ```

- **`replace_lyrics`**: Whether to overwrite existing lyric files.\
    Example:

    ```yaml
    replace_lyrics: false
    ```

- **`download_speed_limit`**: Limit download speed (e.g., `"1MB"` for 1 MB/s).\
    Set to empty or `0` for unlimited speed.\
    Example:

    ```yaml
    download_speed_limit: ""
    ```

### Retry and Pause Settings

- **`retry_attempts_count`**: Number of retry attempts before giving up on a failed download.\
    Example:

    ```yaml
    retry_attempts_count: 5
    ```

- **`max_download_pause`**: Maximum pause duration between track downloads (to mimic human behavior).\
    This value is in Go duration format (e.g., `"2s"` for 2 seconds).\
    Example:

    ```yaml
    max_download_pause: "2s"
    ```

- **`min_retry_pause`**: Minimum pause duration before retrying a failed download attempt.\
    Helps avoid hitting API rate limits or temporary failures.\
    Example:

    ```yaml
    min_retry_pause: "3s"
    ```

- **`max_retry_pause`**: Maximum pause duration before retrying a failed download attempt.\
    Randomized between `min_retry_pause` and `max_retry_pause` for better resilience.\
    Example:

    ```yaml
    max_retry_pause: "7s"
    ```

### Concurrent Downloads

- **`max_concurrent_downloads`**: Maximum number of tracks to download simultaneously.\
    **Default: `1` (sequential downloads - safest and recommended)**\
    \
    ⚠️ **WARNING: Using values greater than 1 may:**
  - Trigger API rate limiting
  - Lead to temporary or permanent account restrictions from Zvuk
  - Disable progress bars (to avoid terminal output conflicts)
    \
    **Use at your own risk.** By increasing this value, you acknowledge that:
  - You are responsible for any consequences, including account blocking
  - This tool's authors are not liable for any service restrictions
  - Sequential downloads (value=1) are the recommended and tested approach
    \
    Example:

    ```yaml
    max_concurrent_downloads: 1  # Recommended default
    ```

    If you want faster downloads despite the risks:

    ```yaml
    max_concurrent_downloads: 3  # Use with caution!
    ```

### Logging

- **`log_level`**: Logging level for the application.\
    Available options: `debug`, `info`, `warn`, `error`, `fatal`.\
    Default: `info`.\
    Example:

    ```yaml
    log_level: "debug"
    ```

* * *

## Troubleshooting 🐛

Having trouble? Follow these steps:

1. **Check Your Configuration File**:\
   Before blaming the code (or me), check if your `.zvuk-grabber.yaml` is up to date.\
   New releases might add new settings with shiny features. If you're using an ancient config file from the Stone Age, weird things might happen.\
   \
   **Quick fix**: Compare your config with the one from the [latest release](https://github.com/oshokin/zvuk-grabber/releases) and add any missing fields.\
   \
   **Pro tip**: If something breaks after an update and you haven't touched your config file in months... yeah, that's probably why.

2. **Enable Debug Logging**:\
   Set the `log_level` to `debug` in the `.zvuk-grabber.yaml` file:

   ```yaml
   log_level: "debug"
   ```

   Attach the logs when reporting issues.

3. **Check Your Token**:\
    Ensure your `auth_token` is valid and properly set in the `.zvuk-grabber.yaml` file.\
    If it's not working, run `zvuk-grabber auth login` to get a fresh token.

4. **Check Your Internet Connection**:\
    A stable connection is essential. If downloads are failing, wait a moment and try again.

5. **Check Zvuk's API Status**:\
    If Zvuk's API is down, check their website or API status page for updates.

* * *

## Support the Project 💖

If you find Zvuk Grabber useful and want to support its development, here's how you can help:

1. **Create a PR**:\
   If you're a developer, create a Pull Request with improvements or bug fixes.\
   Contributions are always welcome!

* * *

## Bug Fixes and Updates 🛠️

I add new features and fix bugs when the stars align, the moon's in the right phase, and my cat's purring just right.\
If you're waiting for a fix, feel free to open an issue or create a PR.

* * *

## Disclaimer ⚠️

- Use Zvuk Grabber responsibly and in compliance with the laws of your country.

- Zvuk’s brand and name are trademarks of their respective owners.

- Zvuk Grabber is not affiliated, sponsored, or endorsed by Zvuk.

* * *
