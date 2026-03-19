// Package slack provides a Slack API client for reading messages and channels.
package slack

import (
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

// Message is an enriched Slack message ready for output.
type Message struct {
	UserID              string   `json:"user_id"`
	UserName            string   `json:"user_name"`
	PostType            PostType `json:"post_type"`
	Timestamp           string   `json:"timestamp"`      // RFC3339
	TimestampUnix       string   `json:"timestamp_unix"` // raw Slack ts
	Text                string   `json:"text"`
	Files               []File   `json:"files"`
	ThreadTimestampUnix string   `json:"thread_timestamp_unix,omitempty"`
	IsReply             bool     `json:"is_reply"`

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
	Type     string    `json:"type"`
	SubType  string    `json:"subtype"`
	User     string    `json:"user"`
	BotID    string    `json:"bot_id"`
	Username string    `json:"username"` // bot display name
	Text     string    `json:"text"`
	Ts       string    `json:"ts"`
	ThreadTs string    `json:"thread_ts"`
	Files    []RawFile `json:"files"`
}

// RawFile is a file attachment as returned by the Slack API.
type RawFile struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	MimeType           string `json:"mimetype"`
	URLPrivateDownload string `json:"url_private_download"`
}
