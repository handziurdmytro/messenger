package chat

import "time"

type MessageType string

const (
	MessageTypeText   MessageType = "text"
	MessageTypeSystem MessageType = "system"
)

type Message struct {
	ID        string
	RoomID    RoomID
	FromUser  string
	Type      MessageType
	Body      string
	CreatedAt time.Time
}
