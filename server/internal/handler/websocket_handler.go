package handler

import (
	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/realtime"

	"github.com/gofiber/websocket/v2"
)

type WebSocketHandler struct {
	hub *realtime.Hub
}

func NewWebSocketHandler(hub *realtime.Hub) *WebSocketHandler {
	return &WebSocketHandler{hub: hub}
}

func (h *WebSocketHandler) HandleWS(c *websocket.Conn) {
	val := c.Locals(contextutil.FiberUserKey)
	user, ok := val.(*model.Account)
	if !ok || user == nil {
		_ = c.Close()
		return
	}
	conn := realtime.NewConn(c)
	h.hub.Register(user.UserName, conn)
	defer h.hub.Unregister(user.UserName, conn)

	for {
		if _, _, err := c.ReadMessage(); err != nil {
			break
		}
	}
}
