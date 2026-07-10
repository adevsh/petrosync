// Package ws implements the WebSocket hub for real-time GPS map updates.
package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Hub manages connected WebSocket clients for active trip map updates.
type Hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]bool
}

// NewHub creates a WebSocket hub.
func NewHub() *Hub { return &Hub{clients: make(map[*websocket.Conn]bool)} }

// HandleUpgrade upgrades an HTTP connection to WebSocket and registers it.
func (h *Hub) HandleUpgrade(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("ws upgrade failed: %v", err)
		return
	}

	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()

	// Keep connection alive — read loop handles pings
	go func() {
		defer func() {
			h.mu.Lock()
			delete(h.clients, conn)
			h.mu.Unlock()
			conn.Close()
		}()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil { break }
		}
	}()
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil { return }

	h.mu.RLock()
	defer h.mu.RUnlock()
	for conn := range h.clients {
		_ = conn.WriteMessage(websocket.TextMessage, data)
	}
}
