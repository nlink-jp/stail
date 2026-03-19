// Package cmd implements the stail CLI commands.
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/magifd2/stail/internal/config"
	"github.com/spf13/cobra"
)

// appState holds resolved runtime configuration shared across subcommands.
type appState struct {
	serverMode bool
	profile    config.Profile
	configPath string
}

var (
	state       appState
	flagConfig  string
	flagProfile string
	flagDebug   bool
)

// rootCmd is the base command.
var rootCmd = &cobra.Command{
	Use:   "stail",
	Short: "Slack tail — read and stream Slack messages",
	Long: `stail reads Slack messages from the command line.

Use 'stail tail' to stream messages in real-time (like tail -f),
or 'stail export' to download full channel history.`,
	SilenceUsage: true,
}

// Execute runs the CLI. Call this from main().
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagConfig, "config", "",
		"config file path (default: ~/.config/stail/config.json)")
	rootCmd.PersistentFlags().StringVarP(&flagProfile, "profile", "p", "",
		"profile to use (overrides current profile)")
	rootCmd.PersistentFlags().BoolVar(&flagDebug, "debug", false,
		"enable debug logging")

	rootCmd.PersistentPreRunE = persistentPreRunE
}

// persistentPreRunE loads configuration before any subcommand runs.
func persistentPreRunE(cmd *cobra.Command, _ []string) error {
	serverMode, err := config.DetectServerMode()
	if err != nil {
		return err
	}
	state.serverMode = serverMode

	if serverMode {
		// In server mode the --config and --profile flags are not allowed.
		if flagConfig != "" {
			return fmt.Errorf("--config flag cannot be used in server mode (STAIL_MODE=server)")
		}
		if flagProfile != "" {
			return fmt.Errorf("--profile flag cannot be used in server mode (STAIL_MODE=server)")
		}
		cfg, err := config.BuildConfigFromEnv()
		if err != nil {
			return err
		}
		p, err := cfg.GetProfile("")
		if err != nil {
			return err
		}
		state.profile = p
		return nil
	}

	// CLI mode: load from config file.
	cfgPath := flagConfig
	if cfgPath == "" {
		cfgPath, err = config.DefaultConfigPath()
		if err != nil {
			return fmt.Errorf("resolve config path: %w", err)
		}
	}
	state.configPath = cfgPath

	cfg, err := config.Load(cfgPath)
	if err != nil {
		// A missing config file is only an error for commands that need it.
		// Return nil here; commands will check state.profile themselves.
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	p, err := cfg.GetProfile(flagProfile)
	if err != nil {
		return err
	}
	state.profile = p
	return nil
}

// requireCLIMode returns an error when run in server mode.
// Use this in profile/config subcommands that cannot work in server mode.
func requireCLIMode() error {
	if state.serverMode {
		return fmt.Errorf("this command is not available in server mode (STAIL_MODE=server)")
	}
	return nil
}

// loadConfig loads (and returns) the current config from disk.
// Must only be called in CLI mode.
func loadConfig() (*config.Config, error) {
	if state.configPath == "" {
		var err error
		state.configPath, err = config.DefaultConfigPath()
		if err != nil {
			return nil, err
		}
	}
	return config.Load(state.configPath)
}
