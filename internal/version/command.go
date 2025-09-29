package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

// AttachCobraVersionCommand attaches a `version` subcommand to the provided root command.
// It prints detailed build info.
func AttachCobraVersionCommand(root *cobra.Command) {
	// Subcommand: `version`.
	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version information.",
		Long: "Print detailed version information including build metadata, commit hash, and build timestamp. " +
			"This information is automatically injected during the build process from Git tags and repository state.",
		Run: func(cmd *cobra.Command, _ []string) {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), Full())
		},
	})
}
