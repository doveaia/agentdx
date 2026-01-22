package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// SSEClient represents a connected SSE client.
type SSEClient struct {
	ID       string
	Messages chan []byte
	Done     chan struct{}
}

// SSEHub manages SSE client connections.
type SSEHub struct {
	clients    map[string]*SSEClient
	register   chan *SSEClient
	unregister chan *SSEClient
	broadcast  chan sseMessage
	mu         sync.RWMutex
}

type sseMessage struct {
	Event string
	Data  interface{}
}

// NewSSEHub creates a new SSE hub.
func NewSSEHub() *SSEHub {
	return &SSEHub{
		clients:    make(map[string]*SSEClient),
		register:   make(chan *SSEClient),
		unregister: make(chan *SSEClient),
		broadcast:  make(chan sseMessage, 256),
	}
}

// Run starts the SSE hub event loop.
func (h *SSEHub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// Close all clients
			h.mu.Lock()
			for _, client := range h.clients {
				close(client.Done)
			}
			h.clients = make(map[string]*SSEClient)
			h.mu.Unlock()
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.Done)
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			data, err := json.Marshal(msg.Data)
			if err != nil {
				continue
			}
			formatted := formatSSE(msg.Event, data)

			h.mu.RLock()
			for _, client := range h.clients {
				select {
				case client.Messages <- formatted:
				default:
					// Client message buffer full, skip
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to all connected clients.
func (h *SSEHub) Broadcast(event string, data interface{}) {
	select {
	case h.broadcast <- sseMessage{Event: event, Data: data}:
	default:
		// Broadcast buffer full, drop message
	}
}

// formatSSE formats a message for SSE.
func formatSSE(event string, data []byte) []byte {
	return []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", event, string(data)))
}

// handleSSEStatus handles the SSE status endpoint.
func (s *Server) handleSSEStatus(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create client
	client := &SSEClient{
		ID:       r.RemoteAddr,
		Messages: make(chan []byte, 256),
		Done:     make(chan struct{}),
	}

	// Register client
	s.sseHub.register <- client

	// Ensure cleanup
	defer func() {
		s.sseHub.unregister <- client
	}()

	// Get flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Send initial status
	ctx := r.Context()
	status := s.getStatus(ctx)
	data, _ := json.Marshal(status)
	fmt.Fprintf(w, "event: status\ndata: %s\n\n", string(data))
	flusher.Flush()

	// Listen for messages
	for {
		select {
		case <-r.Context().Done():
			return
		case <-client.Done:
			return
		case msg := <-client.Messages:
			w.Write(msg)
			flusher.Flush()
		}
	}
}
