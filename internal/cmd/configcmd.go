package cmd

import (
	"fmt"
	"os"

	"github.com/magifd2/stail/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a default configuration file",
	Long: `Create a default configuration file at ~/.config/stail/config.json.

The file is created with 0600 permissions (owner read/write only).
This command fails in server mode (STAIL_MODE=server).

Example:
  stail config init`,
	RunE: runConfigInit,
}

func init() {
	configCmd.AddCommand(configInitCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigInit(_ *cobra.Command, _ []string) error {
	if err := requireCLIMode(); err != nil {
		return err
	}

	// Prefer the path set by --config flag (via state.configPath),
	// falling back to the default location.
	cfgPath := state.configPath
	if cfgPath == "" {
		var err error
		cfgPath, err = config.DefaultConfigPath()
		if err != nil {
			return err
		}
	}

	// Do not overwrite an existing config.
	if _, err := os.Stat(cfgPath); err == nil {
		return fmt.Errorf("config file already exists at %s", cfgPath)
	}

	cfg := config.DefaultConfig()
	if err := config.Save(cfg, cfgPath); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Config file created: %s\n", cfgPath)
	fmt.Fprintln(os.Stdout, "Add a profile with: stail profile add <name>")
	return nil
}
