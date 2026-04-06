package format_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/nlink-jp/stail/internal/format"
	"github.com/nlink-jp/stail/internal/slack"
)

func makeMsg(userID, userName, text, ts, channelID, channelName string) slack.Message {
	return slack.Message{
		UserID:        userID,
		UserName:      userName,
		PostType:      slack.PostTypeUser,
		Timestamp:     ts,
		TimestampUnix: "1700000000.000000",
		Text:          text,
		Files:         []slack.File{},
		ChannelID:     channelID,
		ChannelName:   channelName,
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input   string
		want    format.Format
		wantErr bool
	}{
		{"text", format.FormatText, false},
		{"", format.FormatText, false},
		{"json", format.FormatJSON, false},
		{"xml", "", true},
		{"JSON", "", true},
	}
	for _, tc := range tests {
		got, err := format.ParseFormat(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseFormat(%q): expected error, got nil", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseFormat(%q): unexpected error: %v", tc.input, err)
		}
		if got != tc.want {
			t.Errorf("ParseFormat(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestWriteMessage_Text(t *testing.T) {
	var buf bytes.Buffer
	msg := makeMsg("U001", "alice", "Hello!", "2024-01-15T10:23:45Z", "C001", "general")

	if err := format.WriteMessage(&buf, msg, format.FormatText); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}

	line := buf.String()
	if !strings.Contains(line, "#general") {
		t.Errorf("text line missing channel: %q", line)
	}
	if !strings.Contains(line, "@alice") {
		t.Errorf("text line missing user: %q", line)
	}
	if !strings.Contains(line, "Hello!") {
		t.Errorf("text line missing text: %q", line)
	}
}

func TestWriteMessage_Text_FallsBackToIDs(t *testing.T) {
	var buf bytes.Buffer
	msg := makeMsg("U001", "", "hi", "2024-01-15T10:23:45Z", "C001", "")

	if err := format.WriteMessage(&buf, msg, format.FormatText); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}

	line := buf.String()
	if !strings.Contains(line, "#C001") {
		t.Errorf("expected fallback to channel ID, got: %q", line)
	}
	if !strings.Contains(line, "@U001") {
		t.Errorf("expected fallback to user ID, got: %q", line)
	}
}

func TestWriteMessage_JSON(t *testing.T) {
	var buf bytes.Buffer
	msg := makeMsg("U001", "alice", "Hello JSON!", "2024-01-15T10:23:45Z", "C001", "general")
	msg.IsReply = false

	if err := format.WriteMessage(&buf, msg, format.FormatJSON); err != nil {
		t.Fatalf("WriteMessage JSON: %v", err)
	}

	line := strings.TrimSpace(buf.String())
	var got map[string]interface{}
	if err := json.Unmarshal([]byte(line), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %q", err, line)
	}

	checks := map[string]string{
		"user_id":   "U001",
		"user_name": "alice",
		"text":      "Hello JSON!",
		"post_type": "user",
	}
	for key, want := range checks {
		if got[key] != want {
			t.Errorf("JSON[%q] = %v, want %q", key, got[key], want)
		}
	}
}

func TestWriteMessage_JSON_Fields(t *testing.T) {
	var buf bytes.Buffer
	msg := slack.Message{
		UserID:              "U001",
		UserName:            "bob",
		PostType:            slack.PostTypeUser,
		Timestamp:           "2024-01-15T10:23:45Z",
		TimestampUnix:       "1705316625.123456",
		Text:                "reply here",
		Files:               []slack.File{},
		ThreadTimestampUnix: "1705316600.000000",
		IsReply:             true,
		ChannelID:           "C001",
		ChannelName:         "general",
	}

	if err := format.WriteMessage(&buf, msg, format.FormatJSON); err != nil {
		t.Fatalf("WriteMessage JSON: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if got["timestamp_unix"] != "1705316625.123456" {
		t.Errorf("timestamp_unix = %v", got["timestamp_unix"])
	}
	if got["is_reply"] != true {
		t.Errorf("is_reply = %v, want true", got["is_reply"])
	}
	if got["thread_timestamp_unix"] != "1705316600.000000" {
		t.Errorf("thread_timestamp_unix = %v", got["thread_timestamp_unix"])
	}
}

func TestNewExportedLog(t *testing.T) {
	msgs := []slack.Message{
		makeMsg("U001", "alice", "first", "2024-01-15T10:00:00Z", "C001", "general"),
		makeMsg("U002", "bob", "second", "2024-01-15T10:01:00Z", "C001", "general"),
	}

	log := format.NewExportedLog("#general", msgs)

	if log.ChannelName != "#general" {
		t.Errorf("ChannelName = %q, want #general", log.ChannelName)
	}
	if len(log.Messages) != 2 {
		t.Errorf("Messages len = %d, want 2", len(log.Messages))
	}
	if log.ExportTimestamp == "" {
		t.Error("ExportTimestamp should not be empty")
	}
	if log.Messages[0].UserID != "U001" {
		t.Errorf("Messages[0].UserID = %q, want U001", log.Messages[0].UserID)
	}
}

func TestWriteMessage_JSON_Attachments(t *testing.T) {
	var buf bytes.Buffer
	msg := makeMsg("U001", "alice", "", "2024-01-15T10:23:45Z", "C001", "general")
	msg.Attachments = []slack.Attachment{
		{
			Fallback: "Alert: server down",
			Color:    "#ff0000",
			Title:    "Alert",
			Text:     "Production server is not responding",
			Fields: []slack.AttachmentField{
				{Title: "Severity", Value: "Critical", Short: true},
			},
			Footer: "Monitoring",
		},
	}

	if err := format.WriteMessage(&buf, msg, format.FormatJSON); err != nil {
		t.Fatalf("WriteMessage JSON: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	attachments, ok := got["attachments"].([]interface{})
	if !ok || len(attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %v", got["attachments"])
	}
	att := attachments[0].(map[string]interface{})
	if att["fallback"] != "Alert: server down" {
		t.Errorf("fallback = %v", att["fallback"])
	}
	if att["color"] != "#ff0000" {
		t.Errorf("color = %v", att["color"])
	}
	if att["title"] != "Alert" {
		t.Errorf("title = %v", att["title"])
	}
}

func TestWriteMessage_JSON_Blocks(t *testing.T) {
	var buf bytes.Buffer
	msg := makeMsg("U001", "alice", "Hello world", "2024-01-15T10:23:45Z", "C001", "general")
	msg.Blocks = json.RawMessage(`[{"type":"section","text":{"type":"mrkdwn","text":"Hello *world*"}}]`)

	if err := format.WriteMessage(&buf, msg, format.FormatJSON); err != nil {
		t.Fatalf("WriteMessage JSON: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	blocks, ok := got["blocks"].([]interface{})
	if !ok || len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %v", got["blocks"])
	}
	block := blocks[0].(map[string]interface{})
	if block["type"] != "section" {
		t.Errorf("block type = %v", block["type"])
	}
}

func TestWriteMessage_Text_AttachmentFallback(t *testing.T) {
	var buf bytes.Buffer
	msg := makeMsg("U001", "alice", "", "2024-01-15T10:23:45Z", "C001", "general")
	msg.Attachments = []slack.Attachment{
		{Fallback: "Server is down"},
	}

	if err := format.WriteMessage(&buf, msg, format.FormatText); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}

	line := buf.String()
	if !strings.Contains(line, "[添付: Server is down]") {
		t.Errorf("text line should contain attachment fallback: %q", line)
	}
}

func TestWriteExportedLog(t *testing.T) {
	msgs := []slack.Message{
		makeMsg("U001", "alice", "hello export", "2024-01-15T10:00:00Z", "C001", "general"),
	}
	log := format.NewExportedLog("#general", msgs)

	var buf bytes.Buffer
	if err := format.WriteExportedLog(&buf, log); err != nil {
		t.Fatalf("WriteExportedLog: %v", err)
	}

	var got format.ExportedLog
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if got.ChannelName != "#general" {
		t.Errorf("ChannelName = %q, want #general", got.ChannelName)
	}
	if len(got.Messages) != 1 {
		t.Errorf("Messages len = %d, want 1", len(got.Messages))
	}
}
