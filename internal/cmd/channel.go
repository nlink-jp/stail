package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"

	"github.com/magifd2/stail/internal/slack"
	"github.com/spf13/cobra"
)

var channelCmd = &cobra.Command{
	Use:   "channel",
	Short: "Manage Slack channels",
}

var channelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List channels with their IDs",
	Long: `List all Slack channels the bot has access to.

Examples:
  stail channel list
  stail channel list --json`,
	RunE: runChannelList,
}

var channelListJSON bool

func init() {
	channelListCmd.Flags().BoolVar(&channelListJSON, "json", false, "output in JSON format")
	channelCmd.AddCommand(channelListCmd)
	rootCmd.AddCommand(channelCmd)
}

func runChannelList(_ *cobra.Command, _ []string) error {
	prof := state.profile
	if prof.Token == "" {
		return fmt.Errorf("no token configured — run 'stail config init' and 'stail profile add'")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client := slack.NewHTTPClient(prof.Token)
	channels, err := client.ListChannels(ctx)
	if err != nil {
		return fmt.Errorf("list channels: %w", err)
	}

	if channelListJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(channels)
	}

	// Human-readable table.
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tPRIVATE\tMEMBER")
	for _, ch := range channels {
		private := "no"
		if ch.IsPrivate {
			private = "yes"
		}
		member := "no"
		if ch.IsMember {
			member = "yes"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", ch.ID, ch.Name, private, member)
	}
	return tw.Flush()
}
