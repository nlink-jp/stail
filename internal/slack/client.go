package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	apiBaseURL  = "https://slack.com/api"
	httpTimeout = 30 * time.Second
)

// Client defines the interface for Slack REST API read operations.
type Client interface {
	ListChannels(ctx context.Context) ([]Channel, error)
	ResolveChannelID(ctx context.Context, nameOrID string) (string, error)
	FetchHistory(ctx context.Context, channelID string, opts HistoryOptions) ([]RawMessage, error)
	GetUser(ctx context.Context, userID string) (*User, error)
}

// HTTPClient is the production Slack REST API client.
type HTTPClient struct {
	token      string
	httpClient *http.Client
	baseURL    string
}

// NewHTTPClient creates an HTTPClient with the given bot token.
func NewHTTPClient(token string) *HTTPClient {
	return &HTTPClient{
		token:   token,
		baseURL: apiBaseURL,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
	}
}

// WithTransport replaces the HTTP transport (used in tests).
func (c *HTTPClient) WithTransport(rt http.RoundTripper) *HTTPClient {
	c.httpClient = &http.Client{
		Transport: rt,
		Timeout:   httpTimeout,
	}
	return c
}

// WithBaseURL overrides the API base URL. Primarily used in tests to point
// the client at a local httptest server.
func (c *HTTPClient) WithBaseURL(u string) *HTTPClient {
	c.baseURL = u
	return c
}

// slackAPIResponse is the common response envelope from the Slack Web API.
type slackAPIResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func (c *HTTPClient) get(ctx context.Context, method string, params url.Values) ([]byte, error) {
	reqURL := fmt.Sprintf("%s/%s?%s", c.baseURL, method, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get %s: %w", method, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	return body, nil
}

// ListChannels returns all channels the bot token can access.
func (c *HTTPClient) ListChannels(ctx context.Context) ([]Channel, error) {
	var all []Channel
	cursor := ""
	for {
		params := url.Values{
			"limit":            {"200"},
			"exclude_archived": {"true"},
			"types":            {"public_channel,private_channel"},
		}
		if cursor != "" {
			params.Set("cursor", cursor)
		}

		body, err := c.get(ctx, "conversations.list", params)
		if err != nil {
			return nil, err
		}

		var resp struct {
			slackAPIResponse
			Channels         []Channel `json:"channels"`
			ResponseMetadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parse channels: %w", err)
		}
		if !resp.OK {
			return nil, fmt.Errorf("conversations.list: %s", resp.Error)
		}

		all = append(all, resp.Channels...)
		cursor = resp.ResponseMetadata.NextCursor
		if cursor == "" {
			break
		}
	}
	return all, nil
}

// ResolveChannelID converts a channel name ("general", "#general") or ID to a channel ID.
// If the input looks like an ID (starts with "C"), it is returned as-is.
func (c *HTTPClient) ResolveChannelID(ctx context.Context, nameOrID string) (string, error) {
	name := strings.TrimPrefix(nameOrID, "#")
	// Channel IDs start with C and are longer than 5 chars.
	if len(name) > 5 && strings.HasPrefix(name, "C") {
		return name, nil
	}
	channels, err := c.ListChannels(ctx)
	if err != nil {
		return "", err
	}
	for _, ch := range channels {
		if ch.Name == name || ch.ID == name {
			return ch.ID, nil
		}
	}
	return "", fmt.Errorf("channel %q not found", nameOrID)
}

// FetchHistory retrieves messages from a channel. Results are in reverse
// chronological order (newest first), matching the Slack API behaviour.
func (c *HTTPClient) FetchHistory(ctx context.Context, channelID string, opts HistoryOptions) ([]RawMessage, error) {
	params := url.Values{"channel": {channelID}}
	if opts.Limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", opts.Limit))
	}
	if opts.Oldest != "" {
		params.Set("oldest", opts.Oldest)
	}
	if opts.Latest != "" {
		params.Set("latest", opts.Latest)
	}
	if opts.Cursor != "" {
		params.Set("cursor", opts.Cursor)
	}

	body, err := c.get(ctx, "conversations.history", params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		slackAPIResponse
		Messages []RawMessage `json:"messages"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse history: %w", err)
	}
	if !resp.OK {
		return nil, fmt.Errorf("conversations.history: %s", resp.Error)
	}
	return resp.Messages, nil
}

// GetUser returns information about a Slack user.
func (c *HTTPClient) GetUser(ctx context.Context, userID string) (*User, error) {
	params := url.Values{"user": {userID}}
	body, err := c.get(ctx, "users.info", params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		slackAPIResponse
		User struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			IsBot   bool   `json:"is_bot"`
			Profile struct {
				DisplayName string `json:"display_name"`
				RealName    string `json:"real_name"`
			} `json:"profile"`
		} `json:"user"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse user: %w", err)
	}
	if !resp.OK {
		return nil, fmt.Errorf("users.info: %s", resp.Error)
	}
	return &User{
		ID:          resp.User.ID,
		Name:        resp.User.Name,
		IsBot:       resp.User.IsBot,
		DisplayName: resp.User.Profile.DisplayName,
		RealName:    resp.User.Profile.RealName,
	}, nil
}

// DownloadFile fetches a Slack private file URL and writes the content to w.
// The bot token is used for authentication.
func (c *HTTPClient) DownloadFile(ctx context.Context, fileURL string, w io.Writer) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download file: HTTP %d", resp.StatusCode)
	}
	_, err = io.Copy(w, resp.Body)
	return err
}

// UserCache caches resolved user information to avoid redundant API calls.
type UserCache struct {
	client Client
	mu     sync.Mutex
	cache  map[string]*User
}

// NewUserCache creates a UserCache backed by the given client.
func NewUserCache(client Client) *UserCache {
	return &UserCache{client: client, cache: make(map[string]*User)}
}

// Get returns cached user info, fetching from the API on first access.
// On API error it falls back to a placeholder using the raw user ID.
func (uc *UserCache) Get(ctx context.Context, userID string) *User {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	if u, ok := uc.cache[userID]; ok {
		return u
	}
	u, err := uc.client.GetUser(ctx, userID)
	if err != nil {
		u = &User{ID: userID, Name: userID}
	}
	uc.cache[userID] = u
	return u
}

// ChannelCache caches resolved channel information.
type ChannelCache struct {
	client Client
	mu     sync.Mutex
	byID   map[string]*Channel
}

// NewChannelCache creates a ChannelCache backed by the given client.
func NewChannelCache(client Client) *ChannelCache {
	return &ChannelCache{client: client, byID: make(map[string]*Channel)}
}

// GetName returns the channel name for a channel ID, fetching if needed.
func (cc *ChannelCache) GetName(ctx context.Context, channelID string) string {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if ch, ok := cc.byID[channelID]; ok {
		return ch.Name
	}
	// Populate the cache by listing all channels.
	channels, err := cc.client.ListChannels(ctx)
	if err != nil {
		return channelID
	}
	for i := range channels {
		cc.byID[channels[i].ID] = &channels[i]
	}
	if ch, ok := cc.byID[channelID]; ok {
		return ch.Name
	}
	return channelID
}

// EnrichMessage resolves the UserID and ChannelID of a RawMessage into
// a fully enriched Message using the provided caches.
func EnrichMessage(ctx context.Context, raw RawMessage, channelID, channelName string, users *UserCache) Message {
	pt := PostTypeUser
	userName := ""

	if raw.BotID != "" {
		pt = PostTypeBot
		userName = raw.Username
	} else if raw.User != "" {
		u := users.Get(ctx, raw.User)
		userName = u.DisplayNameOrName()
	}

	files := make([]File, 0, len(raw.Files))
	for _, f := range raw.Files {
		files = append(files, File{
			ID:                 f.ID,
			Name:               f.Name,
			MimeType:           f.MimeType,
			URLPrivateDownload: f.URLPrivateDownload,
		})
	}

	t := ParseTimestamp(raw.Ts)
	ts := ""
	if !t.IsZero() {
		ts = t.UTC().Format(time.RFC3339)
	}

	isReply := raw.ThreadTs != "" && raw.ThreadTs != raw.Ts

	return Message{
		UserID:              raw.User,
		UserName:            userName,
		PostType:            pt,
		Timestamp:           ts,
		TimestampUnix:       raw.Ts,
		Text:                raw.Text,
		Files:               files,
		ThreadTimestampUnix: raw.ThreadTs,
		IsReply:             isReply,
		ChannelID:           channelID,
		ChannelName:         channelName,
	}
}
