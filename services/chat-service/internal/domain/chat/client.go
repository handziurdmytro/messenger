package chat

import "errors"

type ClientID string

var ErrClientNotFound = errors.New("client not found")

type Client struct {
	id     ClientID
	userID string
	roomID RoomID
	send   chan Message
}

func newClient(userID string, roomID RoomID) *Client {
	return &Client{
		id:     ClientID(userID),
		userID: userID,
		roomID: roomID,
		send:   make(chan Message, 256),
	}
}

func (c *Client) Send() <-chan Message {
	return c.send
}

func (c *Client) ID() ClientID   { return c.id }
func (c *Client) UserID() string { return c.userID }
func (c *Client) RoomID() RoomID { return c.roomID }
