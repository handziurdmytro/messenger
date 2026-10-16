package ws

import (
	"sync"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
)

type Registry struct {
	mu          sync.RWMutex
	connections map[chat.UserID]map[*Connection]struct{}
	closed      bool
}

func NewRegistry() *Registry {
	return &Registry{
		connections: make(map[chat.UserID]map[*Connection]struct{}),
	}
}

func (r *Registry) Add(c *Connection) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return false
	}

	if r.connections[c.userID] == nil {
		r.connections[c.userID] = make(map[*Connection]struct{})
	}
	r.connections[c.userID][c] = struct{}{}

	return true
}

func (r *Registry) Remove(c *Connection) {
	r.mu.Lock()

	conns := r.connections[c.userID]
	delete(conns, c)

	if len(conns) == 0 {
		delete(r.connections, c.userID)
	}

	r.mu.Unlock()
	c.Close()
}

func (r *Registry) SendTo(users []chat.UserID, data []byte) {
	r.mu.RLock()

	var slow []*Connection
	seen := make(map[chat.UserID]struct{}, len(users))

	for _, userID := range users {
		if _, exists := seen[userID]; exists {
			continue
		}
		seen[userID] = struct{}{}

		for conn := range r.connections[userID] {
			if !conn.trySend(data) {
				slow = append(slow, conn)
			}
		}
	}

	r.mu.RUnlock()

	for _, conn := range slow {
		r.Remove(conn)
	}
}

func (r *Registry) Close() {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}

	r.closed = true

	var all []*Connection
	for _, conns := range r.connections {
		for conn := range conns {
			all = append(all, conn)
		}
	}
	clear(r.connections)
	r.mu.Unlock()

	for _, conn := range all {
		conn.Close()
	}
}
