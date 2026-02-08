package sse

import (
	"fmt"
	"log"
	"sync"
)

// Event represents a Server-Sent Event
type Event struct {
	EventType string `json:"event"`
	Data      string `json:"data"`
}

// Client represents a connected SSE client
type Client struct {
	ID     string
	UserID string
	Events chan Event
}

// Hub manages all SSE client connections
type Hub struct {
	mu      sync.RWMutex
	clients map[string]*Client
}

// GlobalHub is the singleton SSE Hub instance
var GlobalHub = NewHub()

// NewHub creates a new SSE Hub
func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]*Client),
	}
}

// Register adds a new client to the hub
func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client.ID] = client
	log.Printf("[SSE] Client registered: id=%s user=%s (total: %d)", client.ID, client.UserID, len(h.clients))
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(clientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if client, ok := h.clients[clientID]; ok {
		close(client.Events)
		delete(h.clients, clientID)
		log.Printf("[SSE] Client unregistered: id=%s (total: %d)", clientID, len(h.clients))
	}
}

// Broadcast sends an event to all connected clients
func (h *Hub) Broadcast(event Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, client := range h.clients {
		select {
		case client.Events <- event:
		default:
			log.Printf("[SSE] Client %s buffer full, skipping event", client.ID)
		}
	}
}

// PublishTaskUpdate sends a task update event to all connected clients
func PublishTaskUpdate(projectID, taskID, action string) {
	data := fmt.Sprintf(`{"project_id":"%s","task_id":"%s","action":"%s"}`, projectID, taskID, action)
	GlobalHub.Broadcast(Event{
		EventType: "task_update",
		Data:      data,
	})
	log.Printf("[SSE] Published task_update: project=%s task=%s action=%s", projectID, taskID, action)
}

// PublishProjectUpdate 项目级别更新（创建、进度、状态变化）
func PublishProjectUpdate(projectID, action string) {
	data := fmt.Sprintf(`{"project_id":"%s","action":"%s"}`, projectID, action)
	GlobalHub.Broadcast(Event{
		EventType: "project_update",
		Data:      data,
	})
	log.Printf("[SSE] Published project_update: project=%s action=%s", projectID, action)
}

// SendToUser 给特定用户发送事件（而非广播）
func SendToUser(userID string, event Event) {
	GlobalHub.mu.RLock()
	defer GlobalHub.mu.RUnlock()
	for _, client := range GlobalHub.clients {
		if client.UserID == userID {
			select {
			case client.Events <- event:
			default:
				log.Printf("[SSE] Client %s buffer full, skipping user event", client.ID)
			}
		}
	}
}

// PublishUserTaskUpdate 给特定用户发送任务更新（用于我的任务列表刷新）
func PublishUserTaskUpdate(userID, projectID, taskID, action string) {
	data := fmt.Sprintf(`{"project_id":"%s","task_id":"%s","action":"%s"}`, projectID, taskID, action)
	SendToUser(userID, Event{
		EventType: "my_task_update",
		Data:      data,
	})
	log.Printf("[SSE] Published my_task_update to user=%s: project=%s task=%s action=%s", userID, projectID, taskID, action)
}
