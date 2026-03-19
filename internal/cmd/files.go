package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/magifd2/stail/internal/slack"
)

// saveMessageFiles downloads all files attached to msg into dir.
// Files with no download URL are silently skipped.
// Download errors are printed to stderr but do not abort the process.
func saveMessageFiles(ctx context.Context, client *slack.HTTPClient, msg slack.Message, dir string) {
	for _, f := range msg.Files {
		if f.URLPrivateDownload == "" {
			continue
		}
		filename := filepath.Join(dir, f.ID+"_"+sanitizeFilename(f.Name))
		if err := downloadToFile(ctx, client, f.URLPrivateDownload, filename); err != nil {
			fmt.Fprintf(os.Stderr, "warn: download %s: %v\n", f.Name, err)
			continue
		}
		fmt.Fprintf(os.Stderr, "saved: %s\n", filename)
	}
}

func downloadToFile(ctx context.Context, client *slack.HTTPClient, url, dest string) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	return client.DownloadFile(ctx, url, f)
}

// sanitizeFilename replaces characters that are invalid in file names.
func sanitizeFilename(name string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		}
		return r
	}, name)
}
