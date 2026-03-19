package slack_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/magifd2/stail/internal/slack"
)

// wsTextMessage mirrors websocket.TextMessage without importing gorilla/websocket.
const wsTextMessage = 1

// testWsConn is an in-memory WsConn for socket tests.
// ReadMessage replays pre-loaded messages; after exhausting them it blocks
// until Close is called via the done channel.
type testWsConn struct {
	messages [][]byte
	sent     [][]byte
	pos      int
	done     chan struct{}
}

func newTestWsConn(messages [][]byte) *testWsConn {
	return &testWsConn{messages: messages, done: make(chan struct{})}
}

func (c *testWsConn) ReadMessage() (int, []byte, error) {
	if c.pos < len(c.messages) {
		msg := c.messages[c.pos]
		c.pos++
		return wsTextMessage, msg, nil
	}
	// Block until done to simulate an idle connection.
	<-c.done
	return 0, nil, fmt.Errorf("connection closed")
}

func (c *testWsConn) WriteMessage(_ int, data []byte) error {
	c.sent = append(c.sent, data)
	return nil
}

func (c *testWsConn) Close() error {
	select {
	case <-c.done:
	default:
		close(c.done)
	}
	return nil
}

func (c *testWsConn) SetPingHandler(_ func(string) error) {}

// staticOpenFunc returns a fixed URL, bypassing apps.connections.open.
func staticOpenFunc(url string) slack.OpenFunc {
	return func(_ context.Context) (string, error) { return url, nil }
}

// staticDialFunc returns a fixed WsConn, bypassing real WebSocket dialing.
func staticDialFunc(conn slack.WsConn) slack.DialFunc {
	return func(_ string, _ http.Header) (slack.WsConn, error) { return conn, nil }
}

func mustJSON(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func msgEnvelope(envelopeID, channelID, userID, text, ts string) []byte {
	return mustJSON(map[string]interface{}{
		"envelope_id": envelopeID,
		"type":        "events_api",
		"payload": map[string]interface{}{
			"event": map[string]interface{}{
				"type":    "message",
				"channel": channelID,
				"user":    userID,
				"text":    text,
				"ts":      ts,
			},
		},
	})
}

func TestSocketClient_MessageRouting(t *testing.T) {
	conn := newTestWsConn([][]byte{
		msgEnvelope("env-001", "C001", "U001", "hello socket", "1700000000.000000"),
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	received := make([]slack.Message, 0, 1)
	handler := func(msg slack.Message) error {
		received = append(received, msg)
		cancel() // stop after first message
		return nil
	}

	client := slack.NewSlackSocketClient("xapp-test").
		WithOpenFunc(staticOpenFunc("wss://test")).
		WithDialFunc(staticDialFunc(conn))

	_ = client.Run(ctx, nil, handler)

	if len(received) != 1 {
		t.Fatalf("got %d messages, want 1", len(received))
	}
	if received[0].Text != "hello socket" {
		t.Errorf("Text = %q, want hello socket", received[0].Text)
	}
	if received[0].ChannelID != "C001" {
		t.Errorf("ChannelID = %q, want C001", received[0].ChannelID)
	}
}

func TestSocketClient_ChannelFilter(t *testing.T) {
	conn := newTestWsConn([][]byte{
		msgEnvelope("env-001", "C999", "U001", "filtered out", "1700000001.000000"),
		msgEnvelope("env-002", "C001", "U001", "target message", "1700000002.000000"),
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	received := make([]slack.Message, 0, 1)
	handler := func(msg slack.Message) error {
		received = append(received, msg)
		cancel()
		return nil
	}
	filter := func(channelID string) bool { return channelID == "C001" }

	client := slack.NewSlackSocketClient("xapp-test").
		WithOpenFunc(staticOpenFunc("wss://test")).
		WithDialFunc(staticDialFunc(conn))

	_ = client.Run(ctx, filter, handler)

	if len(received) != 1 {
		t.Fatalf("got %d messages, want 1 (filter should drop C999)", len(received))
	}
	if received[0].Text != "target message" {
		t.Errorf("Text = %q, want target message", received[0].Text)
	}
}

func TestSocketClient_AcknowledgesEnvelope(t *testing.T) {
	conn := newTestWsConn([][]byte{
		msgEnvelope("env-123", "C001", "U001", "ack test", "1700000000.000000"),
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler := func(msg slack.Message) error {
		cancel()
		return nil
	}

	client := slack.NewSlackSocketClient("xapp-test").
		WithOpenFunc(staticOpenFunc("wss://test")).
		WithDialFunc(staticDialFunc(conn))

	_ = client.Run(ctx, nil, handler)

	if len(conn.sent) == 0 {
		t.Fatal("expected ACK message to be sent")
	}
	var ack map[string]string
	if err := json.Unmarshal(conn.sent[0], &ack); err != nil {
		t.Fatalf("ACK is not valid JSON: %v", err)
	}
	if ack["envelope_id"] != "env-123" {
		t.Errorf("ACK envelope_id = %q, want env-123", ack["envelope_id"])
	}
}

func TestSocketClient_SkipsSubtypes(t *testing.T) {
	conn := newTestWsConn([][]byte{
		mustJSON(map[string]interface{}{
			"envelope_id": "env-001",
			"type":        "events_api",
			"payload": map[string]interface{}{
				"event": map[string]interface{}{
					"type":    "message",
					"subtype": "message_changed",
					"channel": "C001",
					"ts":      "1700000000.000000",
				},
			},
		}),
	})
	ctx, cancel := context.WithCancel(context.Background())

	received := 0
	handler := func(msg slack.Message) error {
		received++
		return nil
	}

	client := slack.NewSlackSocketClient("xapp-test").
		WithOpenFunc(staticOpenFunc("wss://test")).
		WithDialFunc(staticDialFunc(conn))

	go func() { cancel() }()
	_ = client.Run(ctx, nil, handler)

	if received != 0 {
		t.Errorf("handler called %d times, want 0 (subtype should be skipped)", received)
	}
}

func TestSocketClient_BotMessage(t *testing.T) {
	conn := newTestWsConn([][]byte{
		mustJSON(map[string]interface{}{
			"envelope_id": "env-bot",
			"type":        "events_api",
			"payload": map[string]interface{}{
				"event": map[string]interface{}{
					"type":     "message",
					"channel":  "C001",
					"bot_id":   "B001",
					"username": "DeployBot",
					"text":     "Deployed v1.0",
					"ts":       "1700000005.000000",
				},
			},
		}),
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	received := make([]slack.Message, 0, 1)
	handler := func(msg slack.Message) error {
		received = append(received, msg)
		cancel()
		return nil
	}

	client := slack.NewSlackSocketClient("xapp-test").
		WithOpenFunc(staticOpenFunc("wss://test")).
		WithDialFunc(staticDialFunc(conn))

	_ = client.Run(ctx, nil, handler)

	if len(received) != 1 {
		t.Fatalf("got %d messages, want 1", len(received))
	}
	if received[0].PostType != slack.PostTypeBot {
		t.Errorf("PostType = %q, want bot", received[0].PostType)
	}
	if received[0].UserName != "DeployBot" {
		t.Errorf("UserName = %q, want DeployBot", received[0].UserName)
	}
}
