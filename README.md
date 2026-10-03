# Zvuk Grabber 🎵

Downloader for [Zvuk (Звук)](https://zvuk.com/) and [Yandex Music](https://music.yandex.ru/), written in Go.

Mix URLs in one run. Zvuk: tracks, releases, playlists, artists, audiobooks, podcasts. Yandex Music: tracks, albums, legacy user playlists, UUID playlists.

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

3. **Set Up Provider Tokens**:

   You only need tokens for providers you actually download from.

   **Option 1: Automatic Browser Login (Recommended)**

   Run provider-specific interactive login commands:

   ```bash
   zvuk-grabber auth zvuk login
   zvuk-grabber auth yandex login
   ```

   This will:
   - Open a browser window
   - Let you log in manually
   - Automatically extract and save the provider token
   - Update your local `.zvuk-grabber.yaml` configuration (copy from `.zvuk-grabber.example.yaml`)

   **Option 2: Manual Token Setup**

   Open `.zvuk-grabber.yaml` and set token fields manually:
   - `zvuk_auth_token`: obtain it from [Zvuk API profile](https://zvuk.com/api/v2/tiny/profile), JSON path `$.result.profile.token`
   - `yandex_music_token`: set your existing Yandex Music OAuth token

4. **Run the Tool**:
   - **Linux/macOS**:

     ```bash
     chmod +x zvuk-grabber  # Make it executable
     ./zvuk-grabber https://zvuk.com/release/38858441 https://music.yandex.ru/album/2985370
     ```

   - **Windows**:

     ```bash
     zvuk-grabber https://zvuk.com/release/38858441 https://music.yandex.ru/album/2985370
     ```

5. **Enjoy Your Music!** 🎶
   Start downloading your favorite tracks, albums, and playlists.

* * *

## Documentation 📚

- [Usage](docs/usage.md): install, sign in, download
- [Configuration](docs/configuration.md): YAML keys

* * *

## Troubleshooting 🐛

Having trouble? Follow these steps:

1. **Check Your Configuration File**:\
   Before blaming the code (or me), check if your `.zvuk-grabber.yaml` is up to date.\
   New releases might add new settings with shiny features. If you're using an ancient config file from the Stone Age, weird things might happen.\
   \
   **Quick fix**: Compare your config with [Configuration](docs/configuration.md)
   and `.zvuk-grabber.example.yaml`. Token login is in
   [Usage](docs/usage.md#sign-in).\
   \
   **Pro tip**: If something breaks after an update and you haven't touched your config file in months... yeah, that's probably why.

2. **Enable Debug Logging**:\
   Set the `log_level` to `debug` in the `.zvuk-grabber.yaml` file:

   ```yaml
   log_level: "debug"
   ```

   Attach the logs when reporting issues.

3. **Check Your Token**:\
    Ensure provider tokens are valid in `.zvuk-grabber.yaml`:\
    - `zvuk_auth_token` for Zvuk URLs
    - `yandex_music_token` for Yandex Music URLs\
    There is no `auth_token` key. Refresh with:
    - `zvuk-grabber auth zvuk login`
    - `zvuk-grabber auth yandex login`

4. **Check the URL**:\
    Zvuk links with `?utm_...` (or anything after `?`) do not match. Strip the query. Yandex query strings are fine. Artist/audiobook/podcast URLs work on Zvuk only.

5. **Check Your Internet Connection**:\
    If downloads are failing, wait a moment and try again.

6. **Check Zvuk's API Status**:\
    If Zvuk's API is down, their website is the status page. I don't have a secret dashboard.

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

- Zvuk's brand and name are trademarks of their respective owners.

- Zvuk Grabber is not affiliated, sponsored, or endorsed by Zvuk.
