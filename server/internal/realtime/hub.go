package realtime

import (
	"sync"

	"github.com/gofiber/websocket/v2"
)

// Conn wraps a single WebSocket connection with a write lock (fasthttp Conn is not concurrency-safe).
type Conn struct {
	ws *websocket.Conn
	mu sync.Mutex
}

// Hub manages all online connections keyed by userID (i.e. UserName), supporting multi-device login per user.
type Hub struct {
	mu    sync.RWMutex
	conns map[string][]*Conn
}

// NewHub creates a Hub instance.
func NewHub() *Hub {
	return &Hub{conns: make(map[string][]*Conn)}
}

// NewConn creates a Conn instance (ws field is unexported, so this constructor must be used).
func NewConn(ws *websocket.Conn) *Conn {
	return &Conn{ws: ws}
}

// Register adds a connection to the userID's connection list.
func (h *Hub) Register(userID string, c *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conns[userID] = append(h.conns[userID], c)
}

// Unregister removes a connection from the userID's connection list (matched by pointer).
func (h *Hub) Unregister(userID string, c *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	conns := h.conns[userID]
	for i, conn := range conns {
		if conn == c {
			h.conns[userID] = append(conns[:i], conns[i+1:]...)
			break
		}
	}
	if len(h.conns[userID]) == 0 {
		delete(h.conns, userID)
	}
}

// SendToUser pushes a message to all online connections of a given user.
// Uses RLock to read the connection list, then locks each connection individually to send;
// write failures (disconnect / slow consumer) trigger Unregister.
func (h *Hub) SendToUser(userID string, msg []byte) {
	h.mu.RLock()
	conns := make([]*Conn, len(h.conns[userID]))
	copy(conns, h.conns[userID])
	h.mu.RUnlock()

	for _, c := range conns {
		c.mu.Lock()
		err := c.ws.WriteMessage(websocket.TextMessage, msg)
		c.mu.Unlock()
		if err != nil {
			h.Unregister(userID, c)
		}
	}
}

// SendToUsers broadcasts msg to multiple users' online connections.
// Empty strings are skipped and duplicates are collapsed. Used for broadcast
// events such as task_updated. Internally delegates to SendToUser so the
// same failure cleanup applies.
func (h *Hub) SendToUsers(userIDs []string, msg []byte) {
	seen := make(map[string]bool, len(userIDs))
	for _, uid := range userIDs {
		if uid == "" || seen[uid] {
			continue
		}
		seen[uid] = true
		h.SendToUser(uid, msg)
	}
}
