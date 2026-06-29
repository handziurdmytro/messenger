package chat

import "errors"

type RoomID string

var (
	ErrRoomNotFound  = errors.New("room not found")
	ErrRoomFull      = errors.New("room is full")
	ErrAlreadyInRoom = errors.New("client already in room")
	ErrNotInRoom     = errors.New("client not in room")
)

const maxRoomSize = 100

type Room struct {
	id      RoomID
	clients map[ClientID]*Client
}

func newRoom(id RoomID) *Room {
	return &Room{
		id:      id,
		clients: make(map[ClientID]*Client),
	}
}

func (r *Room) join(c *Client) error {
	if _, exists := r.clients[c.id]; exists {
		return ErrAlreadyInRoom
	}
	if len(r.clients) >= maxRoomSize {
		return ErrRoomFull
	}
	r.clients[c.id] = c
	return nil
}

func (r *Room) leave(c *Client) error {
	if _, exists := r.clients[c.id]; !exists {
		return ErrNotInRoom
	}
	delete(r.clients, c.id)
	return nil
}

func (r *Room) broadcast(msg Message, from ClientID) {
	for id, client := range r.clients {
		if id == from {
			continue
		}
		select {
		case client.send <- msg:
		default:
		}
	}
}

func (r *Room) isEmpty() bool {
	return len(r.clients) == 0
}
