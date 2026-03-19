package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/magifd2/stail/internal/format"
	"github.com/magifd2/stail/internal/slack"
	"github.com/spf13/cobra"
)

var tailCmd = &cobra.Command{
	Use:   "tail",
	Short: "Stream messages from a Slack channel",
	Long: `Show the last N messages from a channel, then optionally follow new messages.

Without -f, behaves like 'tail -n': prints the last N messages and exits.
With -f, behaves like 'tail -f': prints historical messages then streams new
ones in real time via Slack Socket Mode (requires app_token in profile).

--since fetches all messages from the given timestamp onwards (Slack ts or
RFC3339). Combined with -n it returns the newest N messages since that time.

Examples:
  stail tail -c "#general"
  stail tail -c "#general" -n 50
  stail tail -c "#general" --since 2024-01-15T10:00:00Z
  stail tail -c "#general" --since 1742378100.123456
  stail tail -c "#general" --since 2024-01-15T10:00:00Z -n 5
  stail tail -c "#general" -f
  stail tail -c "#general" -f --format json`,
	RunE: runTail,
}

var (
	tailChannel string
	tailLines   int
	tailFollow  bool
	tailFormat  string
	tailSaveDir string
	tailSince   string
)

func init() {
	tailCmd.Flags().StringVarP(&tailChannel, "channel", "c", "", "channel name or ID")
	tailCmd.Flags().IntVarP(&tailLines, "lines", "n", 10, "number of historical messages to show")
	tailCmd.Flags().BoolVarP(&tailFollow, "follow", "f", false, "follow: stream new messages in real time")
	tailCmd.Flags().StringVar(&tailFormat, "format", "text", "output format: text or json")
	tailCmd.Flags().StringVar(&tailSaveDir, "save-dir", "", "directory to save attached files (created if absent)")
	tailCmd.Flags().StringVarP(&tailSince, "since", "S", "", "show messages since timestamp (Slack ts or RFC3339, e.g. 2024-01-15T10:00:00Z)")
	rootCmd.AddCommand(tailCmd)
}

func runTail(cmd *cobra.Command, _ []string) error {
	prof := state.profile
	if prof.Token == "" {
		return fmt.Errorf("no token configured — run 'stail config init' and 'stail profile add'")
	}

	outFmt, err := format.ParseFormat(tailFormat)
	if err != nil {
		return err
	}

	channel := tailChannel
	if channel == "" {
		channel = prof.Channel
	}
	if channel == "" {
		return fmt.Errorf("channel is required: use -c or set a default channel in your profile")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if tailSaveDir != "" {
		if err := os.MkdirAll(tailSaveDir, 0o755); err != nil {
			return fmt.Errorf("create save-dir: %w", err)
		}
	}

	client := slack.NewHTTPClient(prof.Token)
	users := slack.NewUserCache(client)
	channels := slack.NewChannelCache(client)

	// Resolve channel name to ID.
	channelID, err := client.ResolveChannelID(ctx, channel)
	if err != nil {
		return fmt.Errorf("resolve channel: %w", err)
	}
	channelName := channels.GetName(ctx, channelID)

	// Build history options and fetch messages.
	var rawMsgs []slack.RawMessage
	linesChanged := cmd.Flags().Changed("lines")

	if tailSince != "" {
		oldest, err := parseTimestamp(tailSince)
		if err != nil {
			return fmt.Errorf("parse --since: %w", err)
		}

		if linesChanged {
			// Limit to newest N messages since the given timestamp.
			rawMsgs, err = client.FetchHistory(ctx, channelID, slack.HistoryOptions{
				Oldest: oldest,
				Limit:  tailLines,
			})
			if err != nil {
				return fmt.Errorf("fetch history: %w", err)
			}
		} else {
			// Paginate to collect all messages since the given timestamp.
			opts := slack.HistoryOptions{Oldest: oldest, Limit: 999}
			for {
				page, err := client.FetchHistory(ctx, channelID, opts)
				if err != nil {
					return fmt.Errorf("fetch history: %w", err)
				}
				rawMsgs = append(rawMsgs, page...)
				if len(page) < opts.Limit {
					break
				}
				opts.Latest = page[len(page)-1].Ts
				if opts.Latest == "" {
					break
				}
			}
		}
	} else {
		rawMsgs, err = client.FetchHistory(ctx, channelID, slack.HistoryOptions{Limit: tailLines})
		if err != nil {
			return fmt.Errorf("fetch history: %w", err)
		}
	}

	// Reverse to chronological order.
	for i, j := 0, len(rawMsgs)-1; i < j; i, j = i+1, j-1 {
		rawMsgs[i], rawMsgs[j] = rawMsgs[j], rawMsgs[i]
	}

	for _, raw := range rawMsgs {
		msg := slack.EnrichMessage(ctx, raw, channelID, channelName, users)
		if err := format.WriteMessage(os.Stdout, msg, outFmt); err != nil {
			return err
		}
		if tailSaveDir != "" {
			saveMessageFiles(ctx, client, msg, tailSaveDir)
		}
	}

	if !tailFollow {
		return nil
	}

	// Follow mode: connect via Socket Mode.
	if prof.AppToken == "" {
		return fmt.Errorf("follow mode requires app_token in profile (Socket Mode App-Level Token)")
	}

	socketClient := slack.NewSlackSocketClient(prof.AppToken).WithDebug(flagDebug)
	filter := func(id string) bool { return id == channelID }

	handler := func(socketMsg slack.Message) error {
		// Resolve user name for real-time messages.
		if socketMsg.UserID != "" && socketMsg.UserName == "" {
			u := users.Get(ctx, socketMsg.UserID)
			socketMsg.UserName = u.DisplayNameOrName()
		}
		socketMsg.ChannelName = channelName
		if err := format.WriteMessage(os.Stdout, socketMsg, outFmt); err != nil {
			return err
		}
		if tailSaveDir != "" {
			saveMessageFiles(ctx, client, socketMsg, tailSaveDir)
		}
		return nil
	}

	if !flagQuiet {
		fmt.Fprintf(os.Stderr, "Streaming #%s (Ctrl-C to stop)...\n", channelName)
	}
	return socketClient.Run(ctx, filter, handler)
}
