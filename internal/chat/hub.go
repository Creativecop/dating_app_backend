package chat

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const (
	clientSendBuffer  = 32
	typingThrottleTTL = 2 * time.Second
	writeTimeout      = 10 * time.Second
)

type Hub struct {
	mu         sync.Mutex
	clients    map[uint64]map[*Client]struct{}
	typingSeen map[string]time.Time
}

type Client struct {
	UserID uint64
	conn   *websocket.Conn
	send   chan Event
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uint64]map[*Client]struct{}),
		typingSeen: make(map[string]time.Time),
	}
}

func NewClient(userID uint64, conn *websocket.Conn) *Client {
	return &Client{
		UserID: userID,
		conn:   conn,
		send:   make(chan Event, clientSendBuffer),
	}
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[client.UserID] == nil {
		h.clients[client.UserID] = make(map[*Client]struct{})
	}
	h.clients[client.UserID][client] = struct{}{}
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients := h.clients[client.UserID]; clients != nil {
		if _, ok := clients[client]; ok {
			delete(clients, client)
			close(client.send)
		}
		if len(clients) == 0 {
			delete(h.clients, client.UserID)
		}
	}
}

func (h *Hub) SendToUser(userID uint64, event Event) int {
	h.mu.Lock()
	defer h.mu.Unlock()

	delivered := 0
	for client := range h.clients[userID] {
		select {
		case client.send <- event:
			delivered++
		default:
		}
	}
	return delivered
}

func (h *Hub) DisconnectUser(userID uint64) int {
	h.mu.Lock()
	clients := h.clients[userID]
	if len(clients) == 0 {
		h.mu.Unlock()
		return 0
	}
	delete(h.clients, userID)
	copied := make([]*Client, 0, len(clients))
	for client := range clients {
		copied = append(copied, client)
		close(client.send)
	}
	h.mu.Unlock()

	for _, client := range copied {
		_ = client.conn.Close(websocket.StatusPolicyViolation, "user restricted")
	}
	return len(copied)
}

func (h *Hub) AllowTypingEvent(userID uint64, conversationID uint64, event string) bool {
	key := strconv.FormatUint(userID, 10) + ":" + strconv.FormatUint(conversationID, 10) + ":" + event
	now := time.Now().UTC()

	h.mu.Lock()
	defer h.mu.Unlock()

	if last, ok := h.typingSeen[key]; ok && now.Sub(last) < typingThrottleTTL {
		return false
	}
	h.typingSeen[key] = now
	return true
}

func (c *Client) WritePump(ctx context.Context) {
	for event := range c.send {
		writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
		err := wsjson.Write(writeCtx, c.conn, event)
		cancel()
		if err != nil {
			return
		}
	}
}
