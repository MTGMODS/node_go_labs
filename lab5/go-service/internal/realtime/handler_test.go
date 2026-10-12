package realtime_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"node-go-labs/lab5/go-service/internal/realtime"
)

func TestWebSocketLifecycleAndBroadcast(t *testing.T) {
	hub := realtime.NewHub(5)
	go hub.Run()
	handler := realtime.NewHandler(hub)
	server := httptest.NewServer(http.HandlerFunc(handler.ServeWS))
	defer server.Close()
	url := "ws" + strings.TrimPrefix(server.URL, "http")

	first, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()

	var connected map[string]any
	if err := first.ReadJSON(&connected); err != nil || connected["type"] != "connected" {
		t.Fatalf("first connected event=%v err=%v", connected, err)
	}
	if err := second.ReadJSON(&connected); err != nil || connected["type"] != "connected" {
		t.Fatalf("second connected event=%v err=%v", connected, err)
	}

	if err := first.WriteJSON(map[string]string{"type": "message", "text": "Hello"}); err != nil {
		t.Fatal(err)
	}
	_ = second.SetReadDeadline(time.Now().Add(2 * time.Second))
	var broadcast map[string]any
	if err := second.ReadJSON(&broadcast); err != nil {
		t.Fatal(err)
	}
	if broadcast["type"] != "message" || broadcast["text"] != "Hello" {
		t.Fatalf("unexpected broadcast: %v", broadcast)
	}

	if err := first.WriteMessage(websocket.TextMessage, []byte("not-json")); err != nil {
		t.Fatal(err)
	}
	_ = first.SetReadDeadline(time.Now().Add(2 * time.Second))
	var response map[string]any
	if err := first.ReadJSON(&response); err != nil {
		t.Fatal(err)
	}
	if response["error"] != "invalid_json" {
		t.Fatalf("unexpected error response: %v", response)
	}
	if hub.Stats().ActiveClients != 2 {
		t.Fatalf("active clients=%d", hub.Stats().ActiveClients)
	}
}
