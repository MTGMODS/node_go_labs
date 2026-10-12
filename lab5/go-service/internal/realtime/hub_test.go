package realtime

import (
	"testing"
	"time"
)

func TestHubDropsMessagesAndRemovesSlowClient(t *testing.T) {
	hub := NewHub(2)
	go hub.Run()
	client := &Client{
		id:   "slow",
		send: make(chan []byte, 1),
	}
	client.send <- []byte("already buffered")
	hub.register <- client
	deadline := time.Now().Add(time.Second)
	for hub.Stats().ActiveClients != 1 {
		if time.Now().After(deadline) {
			t.Fatal("client was not registered")
		}
		time.Sleep(time.Millisecond)
	}

	for range 2 {
		hub.BroadcastJSON(map[string]string{"type": "progress"})
	}

	deadline = time.Now().Add(time.Second)
	for hub.Stats().ActiveClients != 0 {
		if time.Now().After(deadline) {
			t.Fatal("slow client should be removed after exceeding the drop limit")
		}
		time.Sleep(time.Millisecond)
	}
}
