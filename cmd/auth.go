package cmd

import (
	"github.com/spf13/cobra"

	"github.com/oshokin/zvuk-grabber/internal/app"
)

// loginCommandUse is the subcommand name shared by provider login commands.
const loginCommandUse = "login"

var (
	// authCmd is the root command for authentication management.
	authCmd = &cobra.Command{
		Use:   "auth",
		Short: "Authentication management commands",
		Long: `Manage provider authentication.

Use explicit provider commands:
  zvuk-grabber auth zvuk login
  zvuk-grabber auth yandex login`,
	}

	// authZvukCmd is the parent command for Zvuk authentication subcommands.
	authZvukCmd = &cobra.Command{
		Use:   "zvuk",
		Short: "Zvuk authentication commands",
	}

	// authZvukLoginCmd opens a browser to log in to Zvuk and save the auth token.
	authZvukLoginCmd = &cobra.Command{
		Use:   loginCommandUse,
		Short: "Login to Zvuk and extract authentication token",
		Long: `Opens a browser window for Zvuk login.

After successful login, the auth cookie is extracted and saved to zvuk_auth_token.`,
		PersistentPreRunE: initConfig,
		Run: func(cmd *cobra.Command, args []string) {
			app.ExecuteZvukAuthLoginCommand(cmd.Context(), appConfig)
		},
	}

	// authYandexCmd is the parent command for Yandex Music authentication subcommands.
	authYandexCmd = &cobra.Command{
		Use:   "yandex",
		Short: "Yandex Music authentication commands",
	}

	// authYandexLoginCmd opens a browser to log in to Yandex Music and save the OAuth token.
	authYandexLoginCmd = &cobra.Command{
		Use:   loginCommandUse,
		Short: "Login to Yandex Music and extract OAuth token",
		Long: `Opens a visible browser window at https://music.yandex.ru/.

Log in to Yandex Music in that browser. The command watches the browser session
with go-rod and saves the detected Yandex Music OAuth token to yandex_music_token.
The token value is never printed to logs.`,
		PersistentPreRunE: initConfig,
		Run: func(cmd *cobra.Command, args []string) {
			app.ExecuteYandexAuthLoginCommand(cmd.Context(), appConfig)
		},
	}
)

// init registers auth commands with the root CLI command.
//
//nolint:gochecknoinits // Cobra requires the init function to set up commands.
func init() {
	authZvukCmd.AddCommand(authZvukLoginCmd)
	authYandexCmd.AddCommand(authYandexLoginCmd)
	authCmd.AddCommand(authZvukCmd, authYandexCmd)
	rootCmd.AddCommand(authCmd)
}
