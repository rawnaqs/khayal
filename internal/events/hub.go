// Package events provides a minimal in-process pub-sub hub for broadcasting
// queue job updates to connected WebSocket clients.
package events

import (
	"sync"

	"github.com/rawnaqs/khayal/internal/queue"
)

// Event is a message broadcast to all subscribers.
type Event struct {
	Event string     `json:"event"` // "job_updated"
	Job   *queue.Job `json:"job"`
}

// Hub fans job updates out to subscribers. Publish never blocks: a slow
// consumer's buffered channel is simply dropped instead of stalling the
// worker.
type Hub struct {
	mu       sync.RWMutex
	channels map[chan Event]struct{}
}

// NewHub creates an empty hub.
func NewHub() *Hub {
	return &Hub{channels: make(map[chan Event]struct{})}
}

// Subscribe registers a new subscriber channel (buffered).
func (h *Hub) Subscribe() chan Event {
	ch := make(chan Event, 16)
	h.mu.Lock()
	h.channels[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// Unsubscribe removes and closes a subscriber channel.
func (h *Hub) Unsubscribe(ch chan Event) {
	h.mu.Lock()
	if _, ok := h.channels[ch]; ok {
		delete(h.channels, ch)
		close(ch)
	}
	h.mu.Unlock()
}

// Publish broadcasts an event to every subscriber without blocking.
func (h *Hub) Publish(evt Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.channels {
		select {
		case ch <- evt:
		default: // slow consumer: drop rather than block the publisher
		}
	}
}
