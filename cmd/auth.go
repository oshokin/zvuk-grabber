package cmd

import (
	"github.com/spf13/cobra"

	"github.com/oshokin/zvuk-grabber/internal/app"
)

var (
	authCmd = &cobra.Command{
		Use:   "auth",
		Short: "Authentication management commands",
		Long: `Manage authentication for Zvuk.

Use 'auth login' to log in via browser and automatically extract your authentication token.`,
	}

	authLoginCmd = &cobra.Command{
		Use:   "login",
		Short: "Login to Zvuk and extract authentication token",
		Long: `Opens a browser window for you to log in to Zvuk.

The login process:
1. Browser opens at https://zvuk.com/login
2. Accept cookies if prompted
3. Enter your phone number (e.g., +71488251742)
4. Click "Получить СМС-код" (Get SMS code)
5. Enter the 5-digit SMS code you receive
6. Wait for authentication to complete

After successful login, the authentication token will be automatically
extracted from your profile and saved to the configuration file.

You can then use the token to download music:
zvuk-grabber https://zvuk.com/album/123456`,
		PersistentPreRun: initConfig,
		Run: func(cmd *cobra.Command, args []string) {
			app.ExecuteAuthLoginCommand(cmd.Context(), appConfig)
		},
	}
)

//nolint:gochecknoinits // Cobra requires the init function to set up commands.
func init() {
	// Add login subcommand to auth command.
	authCmd.AddCommand(authLoginCmd)

	// Add auth command to root command.
	rootCmd.AddCommand(authCmd)
}
