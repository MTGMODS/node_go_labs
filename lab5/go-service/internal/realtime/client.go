package realtime

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait           = 10 * time.Second
	pongWait            = 60 * time.Second
	pingPeriod          = 30 * time.Second
	maxMessageSize      = 16 * 1024
	clientQueueCapacity = 32
)

type Client struct {
	id              string
	connectedAt     string
	hub             *Hub
	conn            *websocket.Conn
	send            chan []byte
	droppedMessages int
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		messageType, payload, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		if messageType != websocket.TextMessage {
			c.sendError("invalid_message", "Binary messages are not supported")
			continue
		}
		var message struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal(payload, &message); err != nil {
			c.sendError("invalid_json", "Message must be valid JSON")
			continue
		}
		message.Text = strings.TrimSpace(message.Text)
		if message.Type != "message" || message.Text == "" || len(message.Text) > 4096 {
			c.sendError("invalid_message", "Expected { type: message, text: non-empty string }")
			continue
		}
		c.hub.broadcastExcept(map[string]any{
			"type": "message", "clientId": c.id, "text": message.Text,
			"sentAt": time.Now().UTC().Format(time.RFC3339Nano),
		}, c)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) sendError(code, message string) {
	payload, _ := json.Marshal(map[string]string{"type": "error", "error": code, "message": message})
	select {
	case c.send <- payload:
	default:
	}
}
