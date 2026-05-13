// Package sse implements a minimal in-memory broadcaster for Server-Sent
// Events. The hub fans one Broadcast call out to every connected admin
// client without blocking the publisher: slow clients have their messages
// dropped rather than backing up the broadcast goroutine.
package sse

import (
	"encoding/json"
	"fmt"
	"sync"
)

// Hub broadcasts pre-formatted SSE frames to many subscribers.
type Hub struct {
	mu      sync.RWMutex
	clients map[chan []byte]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: map[chan []byte]struct{}{}}
}

// Subscribe registers a new client and returns its message channel.
// The channel is buffered (16 frames); when full, frames are dropped
// for that one slow client only.
func (h *Hub) Subscribe() chan []byte {
	ch := make(chan []byte, 16)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// Unsubscribe removes the client and closes its channel.
func (h *Hub) Unsubscribe(ch chan []byte) {
	h.mu.Lock()
	if _, ok := h.clients[ch]; ok {
		delete(h.clients, ch)
		close(ch)
	}
	h.mu.Unlock()
}

// Broadcast marshals data to JSON, wraps it in an SSE event frame, and
// queues it for every connected client. Marshal errors are logged via
// the returned error so callers can decide what to do (most ignore it).
func (h *Hub) Broadcast(event string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	frame := []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", event, payload))
	h.mu.RLock()
	for ch := range h.clients {
		select {
		case ch <- frame:
		default:
			// drop frame for this slow client
		}
	}
	h.mu.RUnlock()
	return nil
}

// Count returns the current number of connected clients (useful for /stats).
func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
