package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/magifd2/stail/internal/slack"
)

// maxFilenameBytes is the maximum byte length for a saved filename (common FS limit is 255).
const maxFilenameBytes = 200

// saveMessageFiles downloads all files attached to msg into dir.
// Files with no download URL are silently skipped.
// Download errors are printed to stderr but do not abort the process.
func saveMessageFiles(ctx context.Context, client *slack.HTTPClient, msg slack.Message, dir string) {
	for _, f := range msg.Files {
		if f.URLPrivateDownload == "" {
			continue
		}
		filename := filepath.Join(dir, buildFilename(f.ID, f.Name))
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

// buildFilename constructs a safe filename from a Slack file ID and name.
// Both components are sanitized and the combined result is truncated to
// avoid hitting filesystem limits (typically 255 bytes on Linux/macOS).
func buildFilename(id, name string) string {
	cleanID := sanitizeFilename(id)
	cleanName := sanitizeFilename(name)
	combined := cleanID + "_" + cleanName
	return truncateFilename(combined)
}

// truncateFilename ensures the filename does not exceed maxFilenameBytes bytes.
// Truncation is performed on rune boundaries to avoid splitting multi-byte characters.
func truncateFilename(name string) string {
	if len(name) <= maxFilenameBytes {
		return name
	}
	b := []byte(name)[:maxFilenameBytes]
	// Walk back to a valid rune boundary.
	for !utf8.Valid(b) {
		b = b[:len(b)-1]
	}
	return string(b)
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
