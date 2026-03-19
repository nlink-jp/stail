package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/magifd2/stail/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage configuration profiles",
}

// ── profile list ────────────────────────────────────────────────────────────

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := requireCLIMode(); err != nil {
			return err
		}
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		for name, p := range cfg.Profiles {
			marker := "  "
			if name == cfg.CurrentProfile {
				marker = "* "
			}
			fmt.Printf("%s%s (provider: %s)\n", marker, name, p.Provider)
		}
		return nil
	},
}

// ── profile use ─────────────────────────────────────────────────────────────

var profileUseCmd = &cobra.Command{
	Use:   "use <profile-name>",
	Short: "Set the active profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		if err := requireCLIMode(); err != nil {
			return err
		}
		name := args[0]
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		if _, ok := cfg.Profiles[name]; !ok {
			return fmt.Errorf("profile %q not found", name)
		}
		cfg.CurrentProfile = name
		if err := config.Save(cfg, state.configPath); err != nil {
			return err
		}
		fmt.Printf("Active profile set to %q\n", name)
		return nil
	},
}

// ── profile add ─────────────────────────────────────────────────────────────

var (
	addProvider string
	addChannel  string
	addUsername string
)

var profileAddCmd = &cobra.Command{
	Use:   "add <profile-name>",
	Short: "Add a new profile",
	Long: `Add a new profile. You will be prompted to enter the Bot Token and,
optionally, the App-Level Token securely (input is not echoed).

Example:
  stail profile add my-workspace --provider slack --channel "#general"`,
	Args: cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		if err := requireCLIMode(); err != nil {
			return err
		}
		name := args[0]
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		if _, ok := cfg.Profiles[name]; ok {
			return fmt.Errorf("profile %q already exists", name)
		}

		token, err := readSecret("Bot Token (xoxb-...): ")
		if err != nil {
			return fmt.Errorf("read token: %w", err)
		}
		appToken, err := readSecret("App Token (xapp-..., leave empty to skip): ")
		if err != nil {
			return fmt.Errorf("read app token: %w", err)
		}

		cfg.Profiles[name] = config.Profile{
			Provider: addProvider,
			Token:    token,
			AppToken: appToken,
			Channel:  addChannel,
			Username: addUsername,
		}

		if err := config.Save(cfg, state.configPath); err != nil {
			return err
		}
		fmt.Printf("Profile %q added.\n", name)
		return nil
	},
}

// ── profile set ─────────────────────────────────────────────────────────────

var profileSetCmd = &cobra.Command{
	Use:   "set <key> [value]",
	Short: "Set a value in the current profile",
	Long: `Set a value in the currently active profile.

Settable keys:
  provider   Provider type (slack)
  token      Bot Token (prompted securely if value is omitted)
  app_token  App-Level Token (prompted securely if value is omitted)
  channel    Default channel
  username   Default display name

Example:
  stail profile set channel "#ops"
  stail profile set token`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(_ *cobra.Command, args []string) error {
		if err := requireCLIMode(); err != nil {
			return err
		}
		key := args[0]
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		name := cfg.CurrentProfile
		p, ok := cfg.Profiles[name]
		if !ok {
			return fmt.Errorf("current profile %q not found", name)
		}

		value := ""
		if len(args) == 2 {
			value = args[1]
		}

		switch key {
		case "provider":
			if value == "" {
				return fmt.Errorf("provider requires a value")
			}
			p.Provider = value
		case "token":
			if value == "" {
				value, err = readSecret("Bot Token: ")
				if err != nil {
					return err
				}
			}
			p.Token = value
		case "app_token":
			if value == "" {
				value, err = readSecret("App Token: ")
				if err != nil {
					return err
				}
			}
			p.AppToken = value
		case "channel":
			p.Channel = value
		case "username":
			p.Username = value
		default:
			return fmt.Errorf("unknown key %q — valid keys: provider, token, app_token, channel, username", key)
		}

		cfg.Profiles[name] = p
		if err := config.Save(cfg, state.configPath); err != nil {
			return err
		}
		fmt.Printf("Profile %q updated: %s\n", name, key)
		return nil
	},
}

// ── profile remove ───────────────────────────────────────────────────────────

var profileRemoveCmd = &cobra.Command{
	Use:     "remove <profile-name>",
	Aliases: []string{"rm"},
	Short:   "Remove a profile",
	Args:    cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		if err := requireCLIMode(); err != nil {
			return err
		}
		name := args[0]
		if name == "default" {
			return fmt.Errorf("cannot remove the default profile")
		}
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		if name == cfg.CurrentProfile {
			return fmt.Errorf("cannot remove the currently active profile %q", name)
		}
		if _, ok := cfg.Profiles[name]; !ok {
			return fmt.Errorf("profile %q not found", name)
		}
		delete(cfg.Profiles, name)
		if err := config.Save(cfg, state.configPath); err != nil {
			return err
		}
		fmt.Printf("Profile %q removed.\n", name)
		return nil
	},
}

func init() {
	profileAddCmd.Flags().StringVar(&addProvider, "provider", config.ProviderSlack, "provider type")
	profileAddCmd.Flags().StringVarP(&addChannel, "channel", "c", "", "default channel")
	profileAddCmd.Flags().StringVarP(&addUsername, "username", "u", "", "default display name")

	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileUseCmd)
	profileCmd.AddCommand(profileAddCmd)
	profileCmd.AddCommand(profileSetCmd)
	profileCmd.AddCommand(profileRemoveCmd)
	rootCmd.AddCommand(profileCmd)
}

// readSecret reads a secret from the terminal without echoing input.
func readSecret(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr) // newline after hidden input
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

