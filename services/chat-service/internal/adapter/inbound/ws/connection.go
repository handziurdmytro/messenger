package ws

import (
	"context"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
)

const (
	sendQueueSize = 64
	writeTimeout  = 10 * time.Second
	pingInterval  = 30 * time.Second
)

type Connection struct {
	userID chat.UserID
	conn   *websocket.Conn
	send   chan []byte
	done   chan struct{}
	mu     sync.Mutex
	closed bool
}

func newConnection(userID chat.UserID, conn *websocket.Conn) *Connection {
	return &Connection{
		userID: userID,
		conn:   conn,
		send:   make(chan []byte, sendQueueSize),
		done:   make(chan struct{}),
	}
}

func (c *Connection) trySend(data []byte) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return false
	}

	select {
	case c.send <- data:
		return true
	default:
		return false
	}
}

func (c *Connection) writeLoop(ctx context.Context) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	defer c.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			return
		case data := <-c.send:
			writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := c.conn.Write(writeCtx, websocket.MessageText, data)
			cancel()
			if err != nil {
				return
			}
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := c.conn.Ping(pingCtx)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

func (c *Connection) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}

	c.closed = true
	close(c.done)
	c.mu.Unlock()

	_ = c.conn.CloseNow()
}
