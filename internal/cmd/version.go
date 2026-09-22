package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// versionCmd prints the same line as --version. Neither reads a config file:
// cobra answers --version before any pre-run hook, and this command replaces
// the root's config-loading hook with one that does nothing.
var versionCmd = &cobra.Command{
	Use:               "version",
	Short:             "Print the version",
	Args:              cobra.NoArgs,
	PersistentPreRunE: func(*cobra.Command, []string) error { return nil },
	RunE: func(cmd *cobra.Command, _ []string) error {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s version %s\n", rootCmd.Name(), rootCmd.Version)
		return err
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
