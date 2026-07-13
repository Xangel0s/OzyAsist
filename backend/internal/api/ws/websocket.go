package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Client struct {
	Conn *websocket.Conn
	Send chan []byte
}

var (
	clients   = make(map[*Client]bool)
	register  = make(chan *Client)
	unregister = make(chan *Client)
	mu        sync.RWMutex
)

func HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WS upgrade error: %v", err)
		return
	}

	client := &Client{Conn: conn, Send: make(chan []byte, 256)}

	mu.Lock()
	clients[client] = true
	mu.Unlock()

	go client.writePump()
	client.readPump()

	mu.Lock()
	delete(clients, client)
	mu.Unlock()
	conn.Close()
}

func (c *Client) readPump() {
	for {
		_, raw, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var msg struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "message":
			var chatMsg clientMessage
			if err := json.Unmarshal(raw, &chatMsg); err == nil && chatMsg.ChatID != "" {
				handleChatMessage(c.Conn, chatMsg)
			}
		case "cancel":
			var cancelMsg struct {
				Type   string `json:"type"`
				ChatID string `json:"chat_id"`
			}
			if err := json.Unmarshal(raw, &cancelMsg); err == nil {
				CancelStream(cancelMsg.ChatID)
			}
		case "consent_response":
			var consentMsg clientMessage
			if err := json.Unmarshal(raw, &consentMsg); err == nil {
				handleConsentResponse(c.Conn, consentMsg)
			}
		}
	}
}

func (c *Client) writePump() {
	for msg := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			break
		}
	}
}

func Broadcast(msg []byte) {
	mu.RLock()
	defer mu.RUnlock()
	for client := range clients {
		select {
		case client.Send <- msg:
		default:
			delete(clients, client)
			close(client.Send)
		}
	}
}
