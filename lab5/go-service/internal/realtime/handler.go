package realtime

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

type Handler struct {
	hub      *Hub
	upgrader websocket.Upgrader
}

func NewHandler(hub *Hub) *Handler {
	return &Handler{
		hub: hub,
		upgrader: websocket.Upgrader{
			ReadBufferSize: 1024, WriteBufferSize: 1024,
			CheckOrigin: func(*http.Request) bool { return true },
		},
	}
}

func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	connection, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	id := strconv.FormatInt(time.Now().UnixNano(), 36)
	client := &Client{
		id: id, connectedAt: time.Now().UTC().Format(time.RFC3339Nano),
		hub: h.hub, conn: connection, send: make(chan []byte, clientQueueCapacity),
	}
	h.hub.register <- client
	connected, _ := json.Marshal(map[string]string{
		"type": "connected", "clientId": id, "connectedAt": client.connectedAt,
	})
	client.send <- connected
	go client.writePump()
	go client.readPump()
}

func (h *Handler) Stats(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(h.hub.Stats())
}
