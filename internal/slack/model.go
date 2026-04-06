// Package slack provides a Slack API client for reading messages and channels.
package slack

import (
	"encoding/json"
	"strconv"
	"time"
)

// Channel represents a Slack channel.
type Channel struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	IsPrivate  bool   `json:"is_private"`
	IsMember   bool   `json:"is_member"`
	IsArchived bool   `json:"is_archived"`
}

// User represents a Slack user.
type User struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	RealName    string `json:"real_name"`
	IsBot       bool   `json:"is_bot"`
}

// DisplayNameOrName returns the best available display name.
func (u *User) DisplayNameOrName() string {
	if u.DisplayName != "" {
		return u.DisplayName
	}
	return u.Name
}

// PostType describes the origin of a message.
type PostType string

const (
	PostTypeUser PostType = "user"
	PostTypeBot  PostType = "bot"
)

// File represents a file attached to a Slack message.
type File struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	MimeType           string `json:"mimetype"`
	URLPrivateDownload string `json:"-"` // not included in export JSON; time-limited URL
}

// Attachment represents a legacy rich attachment on a Slack message.
type Attachment struct {
	Fallback  string            `json:"fallback,omitempty"`
	Color     string            `json:"color,omitempty"`
	Pretext   string            `json:"pretext,omitempty"`
	Title     string            `json:"title,omitempty"`
	TitleLink string            `json:"title_link,omitempty"`
	Text      string            `json:"text,omitempty"`
	Fields    []AttachmentField `json:"fields,omitempty"`
	Footer    string            `json:"footer,omitempty"`
	ImageURL  string            `json:"image_url,omitempty"`
}

// AttachmentField is a key-value pair inside a legacy attachment.
type AttachmentField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// Message is an enriched Slack message ready for output.
type Message struct {
	UserID              string          `json:"user_id"`
	UserName            string          `json:"user_name"`
	PostType            PostType        `json:"post_type"`
	Timestamp           string          `json:"timestamp"`      // RFC3339
	TimestampUnix       string          `json:"timestamp_unix"` // raw Slack ts
	Text                string          `json:"text"`
	Files               []File          `json:"files"`
	Attachments         []Attachment    `json:"attachments,omitempty"`
	Blocks              json.RawMessage `json:"blocks,omitempty"`
	ThreadTimestampUnix string          `json:"thread_timestamp_unix,omitempty"`
	IsReply             bool            `json:"is_reply"`

	// Internal fields for channel resolution (not in export JSON).
	ChannelID   string `json:"-"`
	ChannelName string `json:"-"`
}

// HistoryOptions controls a conversations.history or conversations.replies request.
type HistoryOptions struct {
	Limit  int
	Oldest string // Unix timestamp string (e.g. "1234567890.123456")
	Latest string // Unix timestamp string
	Cursor string
}

// ParseTimestamp converts a Slack ts string to time.Time.
func ParseTimestamp(ts string) time.Time {
	if ts == "" {
		return time.Time{}
	}
	f, err := strconv.ParseFloat(ts, 64)
	if err != nil {
		return time.Time{}
	}
	sec := int64(f)
	nsec := int64((f - float64(sec)) * 1e9)
	return time.Unix(sec, nsec)
}

// RawMessage is the JSON shape returned by conversations.history and conversations.replies.
type RawMessage struct {
	Type        string          `json:"type"`
	SubType     string          `json:"subtype"`
	User        string          `json:"user"`
	BotID       string          `json:"bot_id"`
	Username    string          `json:"username"` // bot display name
	Text        string          `json:"text"`
	Ts          string          `json:"ts"`
	ThreadTs    string          `json:"thread_ts"`
	Files       []RawFile       `json:"files"`
	Attachments []RawAttachment `json:"attachments"`
	Blocks      json.RawMessage `json:"blocks"`
}

// RawFile is a file attachment as returned by the Slack API.
type RawFile struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	MimeType           string `json:"mimetype"`
	URLPrivateDownload string `json:"url_private_download"`
}

// RawAttachment is a legacy rich attachment as returned by the Slack API.
type RawAttachment struct {
	Fallback  string               `json:"fallback"`
	Color     string               `json:"color"`
	Pretext   string               `json:"pretext"`
	Title     string               `json:"title"`
	TitleLink string               `json:"title_link"`
	Text      string               `json:"text"`
	Fields    []RawAttachmentField `json:"fields"`
	Footer    string               `json:"footer"`
	ImageURL  string               `json:"image_url"`
}

// RawAttachmentField is a key-value pair inside a legacy Slack attachment.
type RawAttachmentField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}
