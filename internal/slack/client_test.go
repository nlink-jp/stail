package slack_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/magifd2/stail/internal/slack"
)

// mockTransport implements http.RoundTripper without binding to a network port.
type mockTransport struct {
	handler func(path string, query map[string]string) (interface{}, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	query := make(map[string]string)
	for k, v := range req.URL.Query() {
		if len(v) > 0 {
			query[k] = v[0]
		}
	}
	body, err := m.handler(req.URL.Path, query)
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(body)
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(string(b))),
		Header:     make(http.Header),
	}, nil
}

func newMockClient(handler func(path string, query map[string]string) (interface{}, error)) *slack.HTTPClient {
	// The Slack client builds URLs as "https://slack.com/api/<method>",
	// so the HTTP path seen by RoundTrip is "/api/<method>".
	// We strip the "/api" prefix so callers can match on just "/<method>".
	wrapped := func(path string, query map[string]string) (interface{}, error) {
		return handler(strings.TrimPrefix(path, "/api"), query)
	}
	rt := &mockTransport{handler: wrapped}
	return slack.NewHTTPClient("xoxb-test").WithTransport(rt)
}

// okChannels is a helper that returns a channels.list response.
func okChannels(channels []map[string]interface{}) interface{} {
	return map[string]interface{}{
		"ok":                true,
		"channels":          channels,
		"response_metadata": map[string]string{"next_cursor": ""},
	}
}

func TestListChannels(t *testing.T) {
	client := newMockClient(func(path string, _ map[string]string) (interface{}, error) {
		if path != "/conversations.list" {
			return nil, fmt.Errorf("unexpected path %s", path)
		}
		return okChannels([]map[string]interface{}{
			{"id": "C001", "name": "general", "is_private": false},
			{"id": "C002", "name": "random", "is_private": false},
		}), nil
	})

	channels, err := client.ListChannels(context.Background())
	if err != nil {
		t.Fatalf("ListChannels: %v", err)
	}
	if len(channels) != 2 {
		t.Errorf("got %d channels, want 2", len(channels))
	}
	if channels[0].Name != "general" {
		t.Errorf("channels[0].Name = %q, want general", channels[0].Name)
	}
}

func TestListChannels_Pagination(t *testing.T) {
	callCount := 0
	client := newMockClient(func(path string, query map[string]string) (interface{}, error) {
		callCount++
		if query["cursor"] == "" {
			return map[string]interface{}{
				"ok":       true,
				"channels": []map[string]interface{}{{"id": "C001", "name": "ch1"}},
				"response_metadata": map[string]string{"next_cursor": "next-page"},
			}, nil
		}
		return okChannels([]map[string]interface{}{{"id": "C002", "name": "ch2"}}), nil
	})

	channels, err := client.ListChannels(context.Background())
	if err != nil {
		t.Fatalf("ListChannels: %v", err)
	}
	if len(channels) != 2 {
		t.Errorf("got %d channels, want 2", len(channels))
	}
	if callCount != 2 {
		t.Errorf("API called %d times, want 2", callCount)
	}
}

func TestListChannels_APIError(t *testing.T) {
	client := newMockClient(func(_ string, _ map[string]string) (interface{}, error) {
		return map[string]interface{}{"ok": false, "error": "not_authed"}, nil
	})

	_, err := client.ListChannels(context.Background())
	if err == nil {
		t.Error("expected error on API failure, got nil")
	}
}

func TestResolveChannelID_ByName(t *testing.T) {
	client := newMockClient(func(_ string, _ map[string]string) (interface{}, error) {
		return okChannels([]map[string]interface{}{{"id": "C123", "name": "general"}}), nil
	})

	id, err := client.ResolveChannelID(context.Background(), "#general")
	if err != nil {
		t.Fatalf("ResolveChannelID: %v", err)
	}
	if id != "C123" {
		t.Errorf("got %q, want C123", id)
	}
}

func TestResolveChannelID_ByID(t *testing.T) {
	// Should not make any API call when the input looks like an ID.
	apiCalled := false
	client := newMockClient(func(_ string, _ map[string]string) (interface{}, error) {
		apiCalled = true
		return nil, fmt.Errorf("unexpected API call")
	})

	id, err := client.ResolveChannelID(context.Background(), "C123456789")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "C123456789" {
		t.Errorf("got %q, want C123456789", id)
	}
	if apiCalled {
		t.Error("expected no API call for ID-shaped input")
	}
}

func TestResolveChannelID_NotFound(t *testing.T) {
	client := newMockClient(func(_ string, _ map[string]string) (interface{}, error) {
		return okChannels([]map[string]interface{}{}), nil
	})

	_, err := client.ResolveChannelID(context.Background(), "unknown")
	if err == nil {
		t.Error("expected error for unknown channel, got nil")
	}
}

func TestFetchHistory(t *testing.T) {
	client := newMockClient(func(path string, _ map[string]string) (interface{}, error) {
		if path != "/conversations.history" {
			return nil, fmt.Errorf("unexpected path %s", path)
		}
		return map[string]interface{}{
			"ok": true,
			"messages": []map[string]interface{}{
				{"type": "message", "user": "U001", "text": "hello", "ts": "1700000002.000000"},
				{"type": "message", "user": "U002", "text": "world", "ts": "1700000001.000000"},
			},
		}, nil
	})

	msgs, err := client.FetchHistory(context.Background(), "C001", slack.HistoryOptions{Limit: 10})
	if err != nil {
		t.Fatalf("FetchHistory: %v", err)
	}
	if len(msgs) != 2 {
		t.Errorf("got %d messages, want 2", len(msgs))
	}
}

func TestGetUser(t *testing.T) {
	client := newMockClient(func(path string, _ map[string]string) (interface{}, error) {
		if path != "/users.info" {
			return nil, fmt.Errorf("unexpected path %s", path)
		}
		return map[string]interface{}{
			"ok": true,
			"user": map[string]interface{}{
				"id":   "U001",
				"name": "alice",
				"profile": map[string]interface{}{
					"display_name": "Alice",
					"real_name":    "Alice Smith",
				},
			},
		}, nil
	})

	u, err := client.GetUser(context.Background(), "U001")
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if u.Name != "alice" {
		t.Errorf("Name = %q, want alice", u.Name)
	}
	if u.DisplayName != "Alice" {
		t.Errorf("DisplayName = %q, want Alice", u.DisplayName)
	}
}

func TestGetUser_APIError(t *testing.T) {
	client := newMockClient(func(_ string, _ map[string]string) (interface{}, error) {
		return map[string]interface{}{"ok": false, "error": "user_not_found"}, nil
	})

	_, err := client.GetUser(context.Background(), "U999")
	if err == nil {
		t.Error("expected error for API failure, got nil")
	}
}

func TestUserCache_CachesResult(t *testing.T) {
	callCount := 0
	client := newMockClient(func(_ string, _ map[string]string) (interface{}, error) {
		callCount++
		return map[string]interface{}{
			"ok": true,
			"user": map[string]interface{}{
				"id": "U001", "name": "alice",
				"profile": map[string]string{"display_name": "Alice", "real_name": ""},
			},
		}, nil
	})

	cache := slack.NewUserCache(client)
	ctx := context.Background()

	u1 := cache.Get(ctx, "U001")
	u2 := cache.Get(ctx, "U001") // should hit cache

	if callCount != 1 {
		t.Errorf("API called %d times, want 1 (cache on second call)", callCount)
	}
	if u1.Name != u2.Name {
		t.Error("cached result differs from original")
	}
}

func TestEnrichMessage_User(t *testing.T) {
	client := newMockClient(func(_ string, _ map[string]string) (interface{}, error) {
		return map[string]interface{}{
			"ok": true,
			"user": map[string]interface{}{
				"id": "U001", "name": "alice",
				"profile": map[string]string{"display_name": "Alice", "real_name": ""},
			},
		}, nil
	})

	users := slack.NewUserCache(client)
	raw := slack.RawMessage{
		Type: "message",
		User: "U001",
		Text: "hello",
		Ts:   "1700000000.000000",
	}
	msg := slack.EnrichMessage(context.Background(), raw, "C001", "general", users)

	if msg.PostType != slack.PostTypeUser {
		t.Errorf("PostType = %q, want user", msg.PostType)
	}
	if msg.UserName != "Alice" {
		t.Errorf("UserName = %q, want Alice", msg.UserName)
	}
	if msg.Text != "hello" {
		t.Errorf("Text = %q, want hello", msg.Text)
	}
	if msg.IsReply {
		t.Error("expected IsReply = false for non-reply message")
	}
	if msg.ChannelID != "C001" {
		t.Errorf("ChannelID = %q, want C001", msg.ChannelID)
	}
}

func TestEnrichMessage_Bot(t *testing.T) {
	// Bot messages should not trigger users.info calls.
	apiCalled := false
	client := newMockClient(func(_ string, _ map[string]string) (interface{}, error) {
		apiCalled = true
		return nil, fmt.Errorf("unexpected API call")
	})

	users := slack.NewUserCache(client)
	raw := slack.RawMessage{
		Type:     "message",
		BotID:    "B001",
		Username: "MyBot",
		Text:     "bot says hi",
		Ts:       "1700000001.000000",
	}
	msg := slack.EnrichMessage(context.Background(), raw, "C001", "general", users)

	if msg.PostType != slack.PostTypeBot {
		t.Errorf("PostType = %q, want bot", msg.PostType)
	}
	if msg.UserName != "MyBot" {
		t.Errorf("UserName = %q, want MyBot", msg.UserName)
	}
	if apiCalled {
		t.Error("expected no API call for bot message")
	}
}

func TestEnrichMessage_Reply(t *testing.T) {
	client := newMockClient(func(_ string, _ map[string]string) (interface{}, error) {
		return map[string]interface{}{
			"ok": true,
			"user": map[string]interface{}{
				"id": "U001", "name": "alice",
				"profile": map[string]string{"display_name": "", "real_name": ""},
			},
		}, nil
	})

	users := slack.NewUserCache(client)
	raw := slack.RawMessage{
		Type:     "message",
		User:     "U001",
		Text:     "reply text",
		Ts:       "1700000002.000000",
		ThreadTs: "1700000000.000000",
	}
	msg := slack.EnrichMessage(context.Background(), raw, "C001", "general", users)

	if !msg.IsReply {
		t.Error("expected IsReply = true for thread reply")
	}
	if msg.ThreadTimestampUnix != "1700000000.000000" {
		t.Errorf("ThreadTimestampUnix = %q", msg.ThreadTimestampUnix)
	}
}

func TestParseTimestamp(t *testing.T) {
	t.Run("valid ts", func(t *testing.T) {
		got := slack.ParseTimestamp("1700000000.123456")
		if got.IsZero() {
			t.Error("expected non-zero time for valid ts")
		}
		if got.Unix() != 1700000000 {
			t.Errorf("Unix = %d, want 1700000000", got.Unix())
		}
	})

	t.Run("empty ts", func(t *testing.T) {
		got := slack.ParseTimestamp("")
		if !got.IsZero() {
			t.Error("expected zero time for empty ts")
		}
	})

	t.Run("invalid ts", func(t *testing.T) {
		got := slack.ParseTimestamp("not-a-number")
		if !got.IsZero() {
			t.Error("expected zero time for invalid ts")
		}
	})
}
