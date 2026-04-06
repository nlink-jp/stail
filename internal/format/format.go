// Package format provides output formatters for Slack messages.
// Text format is designed for human reading; JSON format mirrors scat's export log schema.
package format

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/nlink-jp/stail/internal/slack"
)

// Format is the output format selector.
type Format string

const (
	// FormatText outputs human-readable text lines.
	FormatText Format = "text"
	// FormatJSON outputs JSONL (one JSON object per line) for streaming,
	// or a JSON array for batch export.
	FormatJSON Format = "json"
)

// ParseFormat parses a format string into a Format value.
func ParseFormat(s string) (Format, error) {
	switch s {
	case "text", "":
		return FormatText, nil
	case "json":
		return FormatJSON, nil
	default:
		return "", fmt.Errorf("unknown format %q: must be %q or %q", s, FormatText, FormatJSON)
	}
}

// WriteMessage writes a single message to w in the given format.
// For streaming (tail -f) use this per message.
func WriteMessage(w io.Writer, msg slack.Message, fmt Format) error {
	switch fmt {
	case FormatJSON:
		return writeJSONLine(w, msg)
	default:
		return writeTextLine(w, msg)
	}
}

// exportedMessage matches scat's ExportedMessage JSON schema.
type exportedMessage struct {
	UserID              string               `json:"user_id"`
	UserName            string               `json:"user_name,omitempty"`
	PostType            slack.PostType        `json:"post_type,omitempty"`
	Timestamp           string               `json:"timestamp"`
	TimestampUnix       string               `json:"timestamp_unix"`
	Text                string               `json:"text"`
	Files               []exportedFile        `json:"files"`
	Attachments         []exportedAttachment  `json:"attachments,omitempty"`
	Blocks              json.RawMessage       `json:"blocks,omitempty"`
	ThreadTimestampUnix string               `json:"thread_timestamp_unix,omitempty"`
	IsReply             bool                 `json:"is_reply"`
}

// exportedFile matches scat's ExportedFile JSON schema.
type exportedFile struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	MimeType string `json:"mimetype"`
}

// exportedAttachment represents a legacy rich attachment in the export schema.
type exportedAttachment struct {
	Fallback  string                    `json:"fallback,omitempty"`
	Color     string                    `json:"color,omitempty"`
	Pretext   string                    `json:"pretext,omitempty"`
	Title     string                    `json:"title,omitempty"`
	TitleLink string                    `json:"title_link,omitempty"`
	Text      string                    `json:"text,omitempty"`
	Fields    []exportedAttachmentField `json:"fields,omitempty"`
	Footer    string                    `json:"footer,omitempty"`
	ImageURL  string                    `json:"image_url,omitempty"`
}

// exportedAttachmentField is a key-value pair inside a legacy attachment.
type exportedAttachmentField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// ExportedLog matches scat's ExportedLog JSON schema.
type ExportedLog struct {
	ExportTimestamp string            `json:"export_timestamp"`
	ChannelName     string            `json:"channel_name"`
	Messages        []exportedMessage `json:"messages"`
}

// NewExportedLog creates an ExportedLog from enriched messages.
// channelName should include the "#" prefix.
func NewExportedLog(channelName string, messages []slack.Message) ExportedLog {
	exported := make([]exportedMessage, 0, len(messages))
	for _, m := range messages {
		exported = append(exported, messageToExported(m))
	}
	return ExportedLog{
		ExportTimestamp: time.Now().UTC().Format(time.RFC3339),
		ChannelName:     channelName,
		Messages:        exported,
	}
}

// WriteExportedLog serialises an ExportedLog as pretty-printed JSON to w.
func WriteExportedLog(w io.Writer, log ExportedLog) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(log)
}

// WriteExportStream writes a scat-compatible JSON document from pages of messages
// without accumulating all messages into a single slice.
// pages must be in newest-first order (as returned by the Slack API);
// output is written in chronological (oldest-first) order.
func WriteExportStream(w io.Writer, channelName string, pages [][]slack.Message) error {
	ts := time.Now().UTC().Format(time.RFC3339)
	tsJSON, _ := json.Marshal(ts)
	chJSON, _ := json.Marshal(channelName)

	if _, err := fmt.Fprintf(w, "{\n  \"export_timestamp\": %s,\n  \"channel_name\": %s,\n  \"messages\": [", tsJSON, chJSON); err != nil {
		return err
	}

	first := true
	// Iterate pages in reverse (oldest page first), messages within each page in reverse.
	for i := len(pages) - 1; i >= 0; i-- {
		page := pages[i]
		for j := len(page) - 1; j >= 0; j-- {
			if !first {
				if _, err := fmt.Fprint(w, ","); err != nil {
					return err
				}
			}
			em := messageToExported(page[j])
			b, err := json.Marshal(em)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintf(w, "\n    %s", b); err != nil {
				return err
			}
			first = false
		}
	}

	_, err := fmt.Fprintln(w, "\n  ]\n}")
	return err
}

func messageToExported(m slack.Message) exportedMessage {
	files := make([]exportedFile, 0, len(m.Files))
	for _, f := range m.Files {
		files = append(files, exportedFile{ID: f.ID, Name: f.Name, MimeType: f.MimeType})
	}
	return exportedMessage{
		UserID:              m.UserID,
		UserName:            m.UserName,
		PostType:            m.PostType,
		Timestamp:           m.Timestamp,
		TimestampUnix:       m.TimestampUnix,
		Text:                m.Text,
		Files:               files,
		Attachments:         toExportedAttachments(m.Attachments),
		Blocks:              m.Blocks,
		ThreadTimestampUnix: m.ThreadTimestampUnix,
		IsReply:             m.IsReply,
	}
}

// writeJSONLine writes a single message as a compact JSON line (JSONL).
func writeJSONLine(w io.Writer, msg slack.Message) error {
	em := messageToExported(msg)
	b, err := json.Marshal(em)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", b)
	return err
}

// writeTextLine writes a human-readable line for a message.
// Format: <timestamp>  #<channel>  @<user>  <text>
func writeTextLine(w io.Writer, msg slack.Message) error {
	ts := formatDisplayTime(msg.Timestamp)

	channel := msg.ChannelName
	if channel == "" {
		channel = msg.ChannelID
	}
	if channel != "" {
		channel = "#" + channel
	}

	user := msg.UserName
	if user == "" {
		user = msg.UserID
	}
	if user != "" {
		user = "@" + user
	}

	text := msg.Text
	if len(msg.Attachments) > 0 {
		for _, a := range msg.Attachments {
			label := a.Fallback
			if label == "" {
				label = a.Title
			}
			if label == "" {
				label = a.Text
			}
			if label != "" {
				info := "[添付: " + label + "]"
				if text == "" {
					text = info
				} else {
					text = text + " " + info
				}
			}
		}
	}
	if len(msg.Files) > 0 {
		names := make([]string, len(msg.Files))
		for i, f := range msg.Files {
			names[i] = f.Name
		}
		fileInfo := "[添付: " + strings.Join(names, ", ") + "]"
		if text == "" {
			text = fileInfo
		} else {
			text = text + " " + fileInfo
		}
	}

	_, err := fmt.Fprintf(w, "%-19s  %-20s  %-20s  %s\n", ts, channel, user, text)
	return err
}

// formatDisplayTime converts an RFC3339 timestamp to local display format.
func formatDisplayTime(rfc3339 string) string {
	if rfc3339 == "" {
		return "                   "
	}
	t, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		return rfc3339
	}
	return t.Local().Format("2006-01-02 15:04:05")
}

func toExportedFiles(files []slack.File) []exportedFile {
	ef := make([]exportedFile, 0, len(files))
	for _, f := range files {
		ef = append(ef, exportedFile{ID: f.ID, Name: f.Name, MimeType: f.MimeType})
	}
	return ef
}

func toExportedAttachments(attachments []slack.Attachment) []exportedAttachment {
	if len(attachments) == 0 {
		return nil
	}
	ea := make([]exportedAttachment, 0, len(attachments))
	for _, a := range attachments {
		fields := make([]exportedAttachmentField, 0, len(a.Fields))
		for _, f := range a.Fields {
			fields = append(fields, exportedAttachmentField{Title: f.Title, Value: f.Value, Short: f.Short})
		}
		ea = append(ea, exportedAttachment{
			Fallback: a.Fallback, Color: a.Color, Pretext: a.Pretext,
			Title: a.Title, TitleLink: a.TitleLink, Text: a.Text,
			Fields: fields, Footer: a.Footer, ImageURL: a.ImageURL,
		})
	}
	return ea
}
