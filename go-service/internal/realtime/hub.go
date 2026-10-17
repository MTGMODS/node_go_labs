package realtime

import (
	"encoding/json"
	"sync"
)

type broadcastMessage struct {
	payload []byte
	exclude *Client
}

type Hub struct {
	mu                 sync.RWMutex
	clients            map[*Client]bool
	broadcast          chan broadcastMessage
	register           chan *Client
	unregister         chan *Client
	maxDroppedMessages int
}

func NewHub(maxDroppedMessages int) *Hub {
	return &Hub{
		clients:            make(map[*Client]bool),
		broadcast:          make(chan broadcastMessage, 256),
		register:           make(chan *Client),
		unregister:         make(chan *Client),
		maxDroppedMessages: maxDroppedMessages,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.remove(client)
		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				if client == message.exclude {
					continue
				}
				select {
				case client.send <- message.payload:
				default:
					client.droppedMessages++
					if client.droppedMessages >= h.maxDroppedMessages {
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *Hub) remove(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
	}
}

func (h *Hub) BroadcastJSON(value any) {
	payload, err := json.Marshal(value)
	if err == nil {
		h.broadcast <- broadcastMessage{payload: payload}
	}
}

func (h *Hub) broadcastExcept(value any, exclude *Client) {
	payload, err := json.Marshal(value)
	if err == nil {
		h.broadcast <- broadcastMessage{payload: payload, exclude: exclude}
	}
}

type ClientInfo struct {
	ID              string `json:"id"`
	ConnectedAt     string `json:"connectedAt"`
	State           string `json:"state"`
	DroppedMessages int    `json:"droppedMessages"`
}

type Stats struct {
	ActiveClients    int          `json:"activeClients"`
	QueueCapacity    int          `json:"queueCapacity"`
	SlowClientPolicy string       `json:"slowClientPolicy"`
	Clients          []ClientInfo `json:"clients"`
}

func (h *Hub) Stats() Stats {
	h.mu.RLock()
	defer h.mu.RUnlock()
	clients := make([]ClientInfo, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, ClientInfo{
			ID: client.id, ConnectedAt: client.connectedAt,
			State: "OPEN", DroppedMessages: client.droppedMessages,
		})
	}
	return Stats{
		ActiveClients: len(clients), QueueCapacity: clientQueueCapacity,
		SlowClientPolicy: "drop messages, then disconnect slow client", Clients: clients,
	}
}
