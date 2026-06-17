package ws

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		// Allow same-origin or localhost origins for development (vite uses 5173, 4173 etc)
		// r.Host is the target host header
		if origin == "http://"+r.Host || origin == "https://"+r.Host {
			return true
		}
		// Also allow localhost/127.0.0.1 development origins specifically
		if origin == "http://localhost:5173" || origin == "http://127.0.0.1:5173" ||
			origin == "http://localhost:4173" || origin == "http://127.0.0.1:4173" ||
			origin == "http://localhost:8080" || origin == "http://127.0.0.1:8080" ||
			origin == "http://localhost:3080" || origin == "http://127.0.0.1:3080" {
			return true
		}
		return false
	},
}

type Message struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type Hub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[*websocket.Conn]struct{})}
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	h.mu.Lock()
	h.clients[conn] = struct{}{}
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients, conn)
		h.mu.Unlock()
		conn.Close()
	}()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

func (h *Hub) Broadcast(msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			conn.Close()
			delete(h.clients, conn)
		}
	}
}
