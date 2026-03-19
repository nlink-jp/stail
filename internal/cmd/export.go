package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/magifd2/stail/internal/format"
	"github.com/magifd2/stail/internal/slack"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export channel message history",
	Long: `Export the full message history of a Slack channel.

Output is a JSON document matching scat's export log schema:
  { "export_timestamp": "...", "channel_name": "#general", "messages": [...] }

Examples:
  stail export -c "#general"
  stail export -c "#general" --output export.json
  stail export -c "#general" --start 2024-01-01T00:00:00Z --end 2024-02-01T00:00:00Z`,
	RunE: runExport,
}

var (
	exportChannel string
	exportOutput  string
	exportStart   string
	exportEnd     string
	exportSaveDir string
)

func init() {
	exportCmd.Flags().StringVarP(&exportChannel, "channel", "c", "", "channel name or ID (required)")
	exportCmd.Flags().StringVar(&exportOutput, "output", "-", "output file path (- for stdout)")
	exportCmd.Flags().StringVar(&exportStart, "start", "", "start time in RFC3339 format")
	exportCmd.Flags().StringVar(&exportEnd, "end", "", "end time in RFC3339 format")
	exportCmd.Flags().StringVar(&exportSaveDir, "save-dir", "", "directory to save attached files (created if absent)")
	_ = exportCmd.MarkFlagRequired("channel")
	rootCmd.AddCommand(exportCmd)
}

func runExport(_ *cobra.Command, _ []string) error {
	prof := state.profile
	if prof.Token == "" {
		return fmt.Errorf("no token configured — run 'stail config init' and 'stail profile add'")
	}

	channel := exportChannel
	if channel == "" {
		return fmt.Errorf("channel is required: use -c")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if exportSaveDir != "" {
		if err := os.MkdirAll(exportSaveDir, 0o755); err != nil {
			return fmt.Errorf("create save-dir: %w", err)
		}
	}

	client := slack.NewHTTPClient(prof.Token)
	users := slack.NewUserCache(client)
	channels := slack.NewChannelCache(client)

	channelID, err := client.ResolveChannelID(ctx, channel)
	if err != nil {
		return fmt.Errorf("resolve channel: %w", err)
	}
	channelName := channels.GetName(ctx, channelID)

	opts := slack.HistoryOptions{Limit: 200}
	if exportStart != "" {
		t, err := time.Parse(time.RFC3339, exportStart)
		if err != nil {
			return fmt.Errorf("parse --start: %w", err)
		}
		opts.Oldest = fmt.Sprintf("%d.000000", t.Unix())
	}
	if exportEnd != "" {
		t, err := time.Parse(time.RFC3339, exportEnd)
		if err != nil {
			return fmt.Errorf("parse --end: %w", err)
		}
		opts.Latest = fmt.Sprintf("%d.000000", t.Unix())
	}

	// Fetch all messages page by page. Each page is kept separate so that
	// WriteExportStream can iterate in reverse without a full-slice reverse copy.
	var pages [][]slack.Message
	for {
		rawMsgs, err := client.FetchHistory(ctx, channelID, opts)
		if err != nil {
			return fmt.Errorf("fetch history: %w", err)
		}
		page := make([]slack.Message, 0, len(rawMsgs))
		for _, raw := range rawMsgs {
			page = append(page, slack.EnrichMessage(ctx, raw, channelID, channelName, users))
		}
		if len(page) > 0 {
			pages = append(pages, page)
		}
		if len(rawMsgs) < opts.Limit {
			break // no more pages
		}
		if len(rawMsgs) > 0 {
			opts.Latest = rawMsgs[len(rawMsgs)-1].Ts
		}
		if opts.Latest == "" {
			break
		}
	}

	if exportSaveDir != "" {
		// Iterate chronologically for consistent save order.
		for i := len(pages) - 1; i >= 0; i-- {
			for j := len(pages[i]) - 1; j >= 0; j-- {
				saveMessageFiles(ctx, client, pages[i][j], exportSaveDir)
			}
		}
	}

	// Determine output destination.
	out := os.Stdout
	if exportOutput != "-" && exportOutput != "" {
		f, err := os.Create(exportOutput)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer f.Close()
		out = f
	}

	return format.WriteExportStream(out, "#"+channelName, pages)
}
