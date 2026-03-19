package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// MessageHandler is called for each incoming message event.
type MessageHandler func(msg Message) error

// ChannelFilter decides whether events from a channel should be processed.
// A nil filter accepts all channels.
type ChannelFilter func(channelID string) bool

// SocketClient defines the interface for Socket Mode streaming.
type SocketClient interface {
	Run(ctx context.Context, filter ChannelFilter, handler MessageHandler) error
}

// WsConn abstracts a WebSocket connection for testability.
type WsConn interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	Close() error
	SetPingHandler(h func(appData string) error)
}

// DialFunc opens a WebSocket connection to the given URL.
type DialFunc func(urlStr string, header http.Header) (WsConn, error)

// OpenFunc obtains a WebSocket URL (e.g. by calling apps.connections.open).
type OpenFunc func(ctx context.Context) (string, error)

// SlackSocketClient is the production implementation of SocketClient.
type SlackSocketClient struct {
	appToken   string
	httpClient *http.Client
	dial       DialFunc
	open       OpenFunc
	baseURL    string
	debug      bool
}

// NewSlackSocketClient creates a SlackSocketClient with the given App-Level Token.
func NewSlackSocketClient(appToken string) *SlackSocketClient {
	c := &SlackSocketClient{
		appToken: appToken,
		baseURL:  apiBaseURL,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
	}
	// Default dial uses gorilla/websocket with TLS verification enabled.
	c.dial = func(urlStr string, header http.Header) (WsConn, error) {
		dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
		conn, _, err := dialer.Dial(urlStr, header)
		return conn, err
	}
	// Default open calls apps.connections.open.
	c.open = c.defaultOpen
	return c
}

// WithDialFunc replaces the WebSocket dial function (used in tests).
func (c *SlackSocketClient) WithDialFunc(fn DialFunc) *SlackSocketClient {
	c.dial = fn
	return c
}

// WithOpenFunc replaces the function that obtains a WebSocket URL (used in tests).
func (c *SlackSocketClient) WithOpenFunc(fn OpenFunc) *SlackSocketClient {
	c.open = fn
	return c
}

// WithDebug enables verbose debug logging to stderr.
func (c *SlackSocketClient) WithDebug(debug bool) *SlackSocketClient {
	c.debug = debug
	return c
}

// WithSocketBaseURL overrides the API base URL (used in tests).
func (c *SlackSocketClient) WithSocketBaseURL(u string) *SlackSocketClient {
	c.baseURL = u
	c.open = c.defaultOpen
	return c
}

// defaultOpen calls apps.connections.open and returns the WebSocket URL.
func (c *SlackSocketClient) defaultOpen(ctx context.Context) (string, error) {
	u := fmt.Sprintf("%s/apps.connections.open", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(""))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.appToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("apps.connections.open: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var result struct {
		OK    bool   `json:"ok"`
		URL   string `json:"url"`
		Error string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse connection response: %w", err)
	}
	if !result.OK {
		return "", fmt.Errorf("apps.connections.open: %s", result.Error)
	}
	return result.URL, nil
}

// socketEnvelope is the outer envelope for all Socket Mode messages.
type socketEnvelope struct {
	EnvelopeID string          `json:"envelope_id"`
	Type       string          `json:"type"`
	Payload    json.RawMessage `json:"payload"`
	Reason     string          `json:"reason"` // present on disconnect
}

// socketEventPayload is the payload for events_api envelopes.
type socketEventPayload struct {
	Event struct {
		Type        string `json:"type"`
		SubType     string `json:"subtype"`
		Channel     string `json:"channel"`
		User        string `json:"user"`
		BotID       string `json:"bot_id"`
		Username    string `json:"username"`
		Text        string `json:"text"`
		Ts          string `json:"ts"`
		ThreadTs    string `json:"thread_ts"`
		ChannelType string `json:"channel_type"`
		Files       []struct {
			ID                 string `json:"id"`
			Name               string `json:"name"`
			MimeType           string `json:"mimetype"`
			URLPrivateDownload string `json:"url_private_download"`
		} `json:"files"`
	} `json:"event"`
}

// Run connects to Slack via Socket Mode and streams messages to handler.
// It automatically reconnects when Slack sends a disconnect event.
// Blocks until ctx is cancelled or a non-recoverable error occurs.
func (c *SlackSocketClient) Run(ctx context.Context, filter ChannelFilter, handler MessageHandler) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		wsURL, err := c.open(ctx)
		if err != nil {
			return fmt.Errorf("open socket connection: %w", err)
		}
		if !strings.HasPrefix(wsURL, "wss://") {
			return fmt.Errorf("invalid websocket URL from Slack (expected wss://): %q", wsURL)
		}

		shouldReconnect, err := c.runSession(ctx, wsURL, filter, handler)
		if err != nil {
			return err
		}
		if !shouldReconnect {
			return nil
		}

		// Brief pause before reconnecting to avoid hammering the API.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

// runSession handles one WebSocket session.
// Returns (true, nil) when Slack requests a reconnect.
func (c *SlackSocketClient) runSession(ctx context.Context, wsURL string, filter ChannelFilter, handler MessageHandler) (bool, error) {
	conn, err := c.dial(wsURL, nil)
	if err != nil {
		return false, fmt.Errorf("websocket dial: %w", err)
	}
	defer conn.Close()

	conn.SetPingHandler(func(appData string) error {
		return conn.WriteMessage(websocket.PongMessage, []byte(appData))
	})

	msgCh := make(chan []byte, 8)
	errCh := make(chan error, 1)

	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				select {
				case errCh <- err:
				case <-ctx.Done():
				}
				return
			}
			select {
			case msgCh <- msg:
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case err := <-errCh:
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return true, nil
			}
			return false, fmt.Errorf("websocket read: %w", err)
		case raw := <-msgCh:
			reconnect, err := c.handleEnvelope(conn, raw, filter, handler)
			if err != nil {
				return false, err
			}
			if reconnect {
				return true, nil
			}
		}
	}
}

// handleEnvelope processes one Socket Mode envelope.
// Returns (true, nil) when reconnection is requested.
func (c *SlackSocketClient) handleEnvelope(conn WsConn, raw []byte, filter ChannelFilter, handler MessageHandler) (bool, error) {
	var env socketEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		// Malformed envelopes are ignored to keep the stream alive.
		return false, nil
	}

	if c.debug {
		fmt.Fprintf(os.Stderr, "[debug] envelope type=%q envelope_id=%q\n", env.Type, env.EnvelopeID)
	}

	switch env.Type {
	case "hello":
		// Connection established — nothing to do.

	case "disconnect":
		// Slack is asking us to reconnect.
		return true, nil

	case "events_api":
		// Acknowledge before processing to meet Slack's 3-second ACK requirement.
		if env.EnvelopeID != "" {
			ack, _ := json.Marshal(map[string]string{"envelope_id": env.EnvelopeID})
			if err := conn.WriteMessage(websocket.TextMessage, ack); err != nil {
				return false, fmt.Errorf("send ack: %w", err)
			}
		}

		var payload socketEventPayload
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			return false, nil
		}

		ev := payload.Event
		if c.debug {
			fmt.Fprintf(os.Stderr, "[debug] event type=%q subtype=%q channel=%q user=%q bot_id=%q text=%q\n",
				ev.Type, ev.SubType, ev.Channel, ev.User, ev.BotID, ev.Text)
		}

		if ev.Type != "message" {
			return false, nil
		}
		// Skip subtypes (message_deleted, message_changed, etc.).
		if ev.SubType != "" {
			if c.debug {
				fmt.Fprintf(os.Stderr, "[debug] skipping subtype=%q\n", ev.SubType)
			}
			return false, nil
		}
		if filter != nil && !filter(ev.Channel) {
			if c.debug {
				fmt.Fprintf(os.Stderr, "[debug] channel filtered out: %q\n", ev.Channel)
			}
			return false, nil
		}

		files := make([]RawFile, 0, len(ev.Files))
		for _, f := range ev.Files {
			files = append(files, RawFile{
				ID:                 f.ID,
				Name:               f.Name,
				MimeType:           f.MimeType,
				URLPrivateDownload: f.URLPrivateDownload,
			})
		}
		rawMsg := RawMessage{
			Type:     "message",
			User:     ev.User,
			BotID:    ev.BotID,
			Username: ev.Username,
			Text:     ev.Text,
			Ts:       ev.Ts,
			ThreadTs: ev.ThreadTs,
			Files:    files,
		}
		msg := Message{
			UserID:              rawMsg.User,
			UserName:            rawMsg.Username,
			PostType:            postTypeFrom(rawMsg),
			Timestamp:           formatTimestamp(rawMsg.Ts),
			TimestampUnix:       rawMsg.Ts,
			Text:                rawMsg.Text,
			Files:               toFiles(rawMsg.Files),
			ThreadTimestampUnix: rawMsg.ThreadTs,
			IsReply:             rawMsg.ThreadTs != "" && rawMsg.ThreadTs != rawMsg.Ts,
			ChannelID:           ev.Channel,
		}

		if err := handler(msg); err != nil {
			return false, err
		}
	}

	return false, nil
}

func postTypeFrom(raw RawMessage) PostType {
	if raw.BotID != "" {
		return PostTypeBot
	}
	return PostTypeUser
}

func formatTimestamp(ts string) string {
	t := ParseTimestamp(ts)
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func toFiles(rf []RawFile) []File {
	files := make([]File, 0, len(rf))
	for _, f := range rf {
		files = append(files, File{
			ID:                 f.ID,
			Name:               f.Name,
			MimeType:           f.MimeType,
			URLPrivateDownload: f.URLPrivateDownload,
		})
	}
	return files
}
