package chat

import "time"

type MessageType string

const (
	MessageTypeText    MessageType = "text"
	MessageTypeSystem  MessageType = "system"
	MessageTypeConfirm MessageType = "confirm"
)

type Message struct {
	ID        string      `json:"id"`
	RoomID    RoomID      `json:"room_id"`
	FromUser  string      `json:"from_user"`
	Type      MessageType `json:"type"`
	Body      string      `json:"body"`
	CreatedAt time.Time   `json:"created_at"`
}
