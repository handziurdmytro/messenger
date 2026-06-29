package chat

import "sync"

type Hub struct {
	mu    sync.RWMutex
	rooms map[RoomID]*Room
}

func NewHub() *Hub {
	return &Hub{
		rooms: make(map[RoomID]*Room),
	}
}

func (h *Hub) Join(userID string, roomID RoomID) (*Client, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, ok := h.rooms[roomID]
	if !ok {
		return nil, ErrRoomNotFound
	}

	client := newClient(userID, roomID)
	if err := room.join(client); err != nil {
		return nil, err
	}

	return client, nil
}

func (h *Hub) Leave(client *Client) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, ok := h.rooms[client.roomID]
	if !ok {
		return ErrRoomNotFound
	}

	if err := room.leave(client); err != nil {
		return err
	}

	if room.isEmpty() {
		delete(h.rooms, client.roomID)
	}

	close(client.send)
	return nil
}

func (h *Hub) Publish(msg Message) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	room, ok := h.rooms[msg.RoomID]
	if !ok {
		return ErrRoomNotFound
	}

	room.broadcast(msg, ClientID(msg.FromUser))
	return nil
}

func (h *Hub) CreateRoom(id RoomID) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.rooms[id]; exists {
		return nil
	}

	h.rooms[id] = newRoom(id)
	return nil
}
