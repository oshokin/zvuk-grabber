Zvuk Grabber 🎵
===============

[Zvuk (Звук)](https://zvuk.com/) grabber written in Go.\
This tool allows you to download artists, albums, tracks, and playlists from Zvuk.

* * *

### Quick Start 🚀

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

Installation 🛠️
------------

### Download Pre-built Binaries

Pre-built binaries for **macOS**, **Windows**, and **Linux** (for both `arm64` and `amd64` architectures) are available on the [Releases page](https://github.com/oshokin/zvuk-grabber/releases).

1.  **Download the Correct Binary**:  
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

2.  **Extract the Bundle**:  
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

3.  **Set Up Authentication Token**:  
    Open the `.zvuk-grabber.yaml` file and set your authentication token in the `auth_token` field.\
    You can obtain the token by logging into [Zvuk's API](https://zvuk.com/api/v2/tiny/profile) and locating the token using the JSON path `$.result.profile.token`.

4.  **Run the Binary**:  
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

1.  **Install Go**:  
    Download and install Go from the [official website](https://go.dev/dl/).

2.  **Install Task**:  
    [Task](https://taskfile.dev/) is a task runner/build tool that simplifies the build process.\
    Install it by following the instructions on their [official website](https://taskfile.dev/installation/).

3.  **Clone the Repository**:  
    Clone the repository to your local machine:
    ```bash
    git clone https://github.com/oshokin/zvuk-grabber.git
    cd zvuk-grabber
    ```

4.  **Build the Binary**:  
    Use Task to build the binary:
    ```bash
    task build
    ```
    The compiled binary will be stored in the `bin/` directory.

* * *

Usage 🎧
-----

### Downloading Content

1.  **Download Albums**:  
    To download one or more albums, provide the album URLs as arguments:
    ```bash
    zvuk-grabber https://zvuk.com/release/36599795 https://zvuk.com/release/37212880
    ```

2.  **Download Tracks**:  
    To download individual tracks, provide the track URLs.\
    The tracks will be organized as if they were part of an album, with a folder and cover art:
    ```bash
    zvuk-grabber https://zvuk.com/track/67856297 https://zvuk.com/track/51397074 https://zvuk.com/track/63391919 https://zvuk.com/track/106773860 https://zvuk.com/track/114947212
    ```

3.  **Download Playlists**:  
    To download a playlist, provide the playlist URL:
    ```bash
    zvuk-grabber https://zvuk.com/playlist/9037842
    ```

4.  **Download Artists**:  
    To download an artist's entire discography, provide the artist URL:
    ```bash
    zvuk-grabber https://zvuk.com/artist/3196437
    ```

5.  **Using Text Files**:  
    You can also provide text files containing URLs (one per line):
    ```bash
    zvuk-grabber 1.txt 2.txt
    ```

* * *

Configuration ⚙️
-------------

The default configuration is already set in the `.zvuk-grabber.yaml` file.\
You only need to modify it if you want to customize the behavior.\
Key options include:

### Authentication

*   **`auth_token`**: Your Zvuk API authentication token.\
    To obtain your token, log in to [Zvuk's API](https://zvuk.com/api/v2/tiny/profile) and locate the token using the JSON path `$.result.profile.token`.  
    Example:

    ```yaml
    auth_token: "a3f8e7b2c5d946f1a0b9e8d7c6f5e4a2"
    ```

### Audio Format

*   **`download_format`**: Preferred audio format for downloaded files.\
    Available options:
    *   `1` = MP3, 128 Kbps (standard quality)
    *   `2` = MP3, 320 Kbps (high quality)
    *   `3` = FLAC, 16/24-bit (lossless quality)
    Example:
    ```yaml
    download_format: 3
    ```

### Output Settings

*   **`output_path`**: Directory where downloaded files will be saved.\
    You can specify either a relative path (e.g., `"zvuk downloads"`) or an absolute path (e.g., `"C:/Music"`).\
    Example:
    ```yaml
    output_path: "zvuk downloads"
    ```

*   **`create_folder_for_singles`**: Whether to create a separate folder for single tracks (tracks not part of an album).\
    If set to `false`, single tracks will be saved directly in the output directory.\
    Example:
    ```yaml
    create_folder_for_singles: false
    ```

*   **`max_folder_name_length`**: Maximum length for folder names created by the application.\
    This ensures folder names remain readable and compatible across different operating systems.\
    Set to `0` to avoid cutting folder names.\
    Example:
    ```yaml
    max_folder_name_length: 100
    ```

### File Naming Templates

*   **`track_filename_template`**: Track file naming format.\
    Available placeholders:
    *   `{{.albumArtist}}`: Primary artist(s) of the album.
    *   `{{.albumID}}`: Unique identifier for the album.
    *   `{{.albumTitle}}`: Title of the album.
    *   `{{.albumTrackCount}}`: Total number of tracks in the album.
    *   `{{.collectionTitle}}`: Title of the album.
    *   `{{.recordLabel}}`: Name of the record label.
    *   `{{.releaseDate}}`: Full release date of the album (YYYY-MM-DD format).
    *   `{{.releaseYear}}`: Year the album was released.
    *   `{{.trackArtist}}`: Artist(s) of the track.
    *   `{{.trackCount}}`: Total number of tracks in the album.
    *   `{{.trackGenre}}`: Genre(s) of the track.
    *   `{{.trackID}}`: Unique identifier for the track.
    *   `{{.trackNumber}}`: Track number within the album (without leading zeros).
    *   `{{.trackNumberPad}}`: Track number with two-digit padding (e.g., 01, 02).
    *   `{{.trackTitle}}`: Track title.
    *   `{{.type}}`: "album" (used to differentiate album tracks).

    Example:
    ```yaml
    track_filename_template: "{{.trackNumberPad}} - {{.trackTitle}}"
    ```

*   **`album_folder_template`**: Album folder naming format.\
    Available placeholders:
    *   `{{.albumArtist}}`: Primary artist(s) of the album.
    *   `{{.albumID}}`: Unique identifier for the album.
    *   `{{.albumTitle}}`: Title of the album.
    *   `{{.albumTrackCount}}`: Total number of tracks in the album.
    *   `{{.releaseDate}}`: Full release date of the album (YYYY-MM-DD format).
    *   `{{.releaseYear}}`: Year the album was released.
    *   `{{.type}}`: "album" (used to differentiate albums from playlists).

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

*   **`playlist_filename_template`**: Playlist file naming format.\
    Available placeholders:
    *   `{{.albumArtist}}`: Primary artist(s) of the album containing the track.
    *   `{{.albumID}}`: Unique identifier for the album containing the track.
    *   `{{.albumTitle}}`: Title of the album containing the track.
    *   `{{.albumTrackCount}}`: Total number of tracks in the album.
    *   `{{.collectionTitle}}`: Title of the playlist.
    *   `{{.playlistID}}`: Unique identifier for the playlist.
    *   `{{.playlistTitle}}`: Title of the playlist.
    *   `{{.playlistTrackCount}}`: Total number of tracks in the playlist.
    *   `{{.recordLabel}}`: Name of the record label.
    *   `{{.releaseDate}}`: Full release date of the album containing the track (YYYY-MM-DD format).
    *   `{{.releaseYear}}`: Year the album containing the track was released.
    *   `{{.trackArtist}}`: Artist(s) of the track.
    *   `{{.trackCount}}`: Total number of tracks in the playlist.
    *   `{{.trackGenre}}`: Genre(s) of the track.
    *   `{{.trackID}}`: Unique identifier for the track.
    *   `{{.trackNumber}}`: Track number within the playlist (without leading zeros).
    *   `{{.trackNumberPad}}`: Track number with two-digit padding (e.g., 01, 02).
    *   `{{.trackTitle}}`: Track title.
    *   `{{.type}}`: "playlist" (used to differentiate playlists from albums).

    Example:
    ```yaml
    playlist_filename_template: "{{.trackNumberPad}} - {{.trackArtist}} - {{.trackTitle}}"
    ```

### Download Behavior

*   **`download_lyrics`**: Whether to download lyrics for tracks (if available).\
    Example:
    ```yaml
    download_lyrics: true
    ```

*   **`replace_tracks`**: Whether to overwrite existing track files.\
    Example:
    ```yaml
    replace_tracks: false
    ```

*   **`replace_covers`**: Whether to overwrite existing cover art files.\
    Example:
    ```yaml
    replace_covers: false
    ```

*   **`replace_lyrics`**: Whether to overwrite existing lyric files.\
    Example:
    ```yaml
    replace_lyrics: false
    ```

*   **`download_speed_limit`**: Limit download speed (e.g., `"1MB"` for 1 MB/s).\
    Set to empty or `0` for unlimited speed.\
    Example:
    ```yaml
    download_speed_limit: ""
    ```

### Retry and Pause Settings

*   **`retry_attempts_count`**: Number of retry attempts before giving up on a failed download.\
    Example:
    ```yaml
    retry_attempts_count: 5
    ```

*   **`max_download_pause`**: Maximum pause duration between track downloads (to mimic human behavior).\
    This value is in Go duration format (e.g., `"2s"` for 2 seconds).\
    Example:
    ```yaml
    max_download_pause: "2s"
    ```

*   **`min_retry_pause`**: Minimum pause duration before retrying a failed download attempt.\
    Helps avoid hitting API rate limits or temporary failures.\
    Example:
    ```yaml
    min_retry_pause: "3s"
    ```

*   **`max_retry_pause`**: Maximum pause duration before retrying a failed download attempt.\
    Randomized between `min_retry_pause` and `max_retry_pause` for better resilience.\
    Example:
    ```yaml
    max_retry_pause: "7s"
    ```

### Logging

*   **`log_level`**: Logging level for the application.\
    Available options: `debug`, `info`, `warn`, `error`, `fatal`.\
    Default: `info`.\
    Example:
    ```yaml
    log_level: "debug"
    ```

* * *

Troubleshooting 🐛
------------------

Having trouble? Follow these steps:

1. **Enable Debug Logging**:\
   Set the `log_level` to `debug` in the `.zvuk-grabber.yaml` file:
   ```yaml
   log_level: "debug"
   ```
   Attach the logs when reporting issues.

2. **Check Your Token**:\
    Ensure your `auth_token` is valid and properly set in the `.zvuk-grabber.yaml` file.\
    If it's not working, log in again and obtain a new token.

3. **Check Your Internet Connection**:\
    A stable connection is essential. If downloads are failing, wait a moment and try again.

4. **Check Zvuk's API Status**:\
    If Zvuk's API is down, check their website or API status page for updates.

* * *

Support the Project 💖
----------------------

If you find Zvuk Grabber useful and want to support its development, here's how you can help:

1. **Create a PR**:\
   If you're a developer, create a Pull Request with improvements or bug fixes.\
   Contributions are always welcome!

* * *

Bug Fixes and Updates 🛠️
-------------------------

I add new features and fix bugs when the stars align, the moon's in the right phase, and my cat's purring just right.\
If you're waiting for a fix, feel free to open an issue or create a PR.

* * *

Disclaimer ⚠️
----------

* Use Zvuk Grabber responsibly and in compliance with the laws of your country.

* Zvuk’s brand and name are trademarks of their respective owners.

* Zvuk Grabber is not affiliated, sponsored, or endorsed by Zvuk.

* * *
