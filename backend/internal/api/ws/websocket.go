package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

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

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

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
				handleChatMessage(c, chatMsg)
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
				handleConsentResponse(c, consentMsg)
			}
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case msg, ok := <-c.Send:
			if !ok {
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func Broadcast(msg []byte) {
	mu.Lock()
	defer mu.Unlock()
	for client := range clients {
		select {
		case client.Send <- msg:
		default:
			delete(clients, client)
			close(client.Send)
		}
	}
}

func ShutdownAll() {
	// Cancel all active streams
	activeStreamsMu.Lock()
	for _, s := range activeStreams {
		s.cancel()
	}
	activeStreams = nil
	activeStreamsMu.Unlock()

	// Close all WebSocket clients
	mu.Lock()
	defer mu.Unlock()
	for client := range clients {
		close(client.Send)
		client.Conn.Close()
	}
	clients = make(map[*Client]bool)
}
