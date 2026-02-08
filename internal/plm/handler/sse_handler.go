package handler

import (
	"fmt"
	"time"

	"github.com/bitfantasy/nimo/internal/plm/sse"
	"github.com/gin-gonic/gin"
)

// SSEHandler handles SSE connections
type SSEHandler struct {
	hub *sse.Hub
}

// NewSSEHandler creates a new SSE handler
func NewSSEHandler() *SSEHandler {
	return &SSEHandler{hub: sse.GlobalHub}
}

// Stream handles the SSE endpoint
// GET /api/v1/sse/events?token=xxx
func (h *SSEHandler) Stream(c *gin.Context) {
	userID := GetUserID(c)
	clientID := fmt.Sprintf("%s_%d", userID, time.Now().UnixNano())

	client := &sse.Client{
		ID:     clientID,
		UserID: userID,
		Events: make(chan sse.Event, 64),
	}

	h.hub.Register(client)

	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	// Send initial connection event
	c.Writer.WriteString("event: connected\ndata: {\"client_id\":\"" + clientID + "\"}\n\n")
	c.Writer.Flush()

	// Heartbeat ticker
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	// Client disconnect detection
	clientGone := c.Request.Context().Done()

	for {
		select {
		case <-clientGone:
			h.hub.Unregister(clientID)
			return
		case event, ok := <-client.Events:
			if !ok {
				return
			}
			c.Writer.WriteString(fmt.Sprintf("event: %s\ndata: %s\n\n", event.EventType, event.Data))
			c.Writer.Flush()
		case <-heartbeat.C:
			c.Writer.WriteString(": keepalive\n\n")
			c.Writer.Flush()
		}
	}
}
