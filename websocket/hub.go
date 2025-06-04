package websocket

import (
	"campus-canvas-chat/database"
	"campus-canvas-chat/models"
	"campus-canvas-chat/redis"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域
	},
}

// Client WebSocket客户端
type Client struct {
	Conn   *websocket.Conn
	UserID int64
	RoomID *int64 // 可选的房间ID，用于群聊
	Send   chan []byte
}

// Hub WebSocket连接管理器
type Hub struct {
	Clients    map[*Client]bool
	Broadcast  chan []byte
	Register   chan *Client
	Unregister chan *Client
	Rooms      map[int64]map[*Client]bool // roomID -> clients
	Users      map[int64]*Client          // userID -> client (用于私聊)
	Mutex      sync.RWMutex
}

// Message WebSocket消息结构
type WSMessage struct {
	Type      string      `json:"type"` // message, join, leave, error
	RoomID    int64       `json:"room_id"`
	UserID    int64       `json:"user_id"`
	Username  string      `json:"username"`
	Content   string      `json:"content"`
	Timestamp int64       `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

// NewHub 创建新的Hub
func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Rooms:      make(map[int64]map[*Client]bool),
		Users:      make(map[int64]*Client),
	}
}

// Run 运行Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.registerClient(client)

		case client := <-h.Unregister:
			h.unregisterClient(client)

		case message := <-h.Broadcast:
			h.broadcastMessage(message)
		}
	}
}

// registerClient 注册客户端
func (h *Hub) registerClient(client *Client) {
	h.Mutex.Lock()
	defer h.Mutex.Unlock()

	h.Clients[client] = true

	// 注册用户连接（用于私聊）
	h.Users[client.UserID] = client

	// 如果指定了房间ID，则添加到房间（用于群聊）
	if client.RoomID != nil {
		if h.Rooms[*client.RoomID] == nil {
			h.Rooms[*client.RoomID] = make(map[*Client]bool)
		}
		h.Rooms[*client.RoomID][client] = true
		redis.AddUserToRoom(*client.RoomID, client.UserID)
		log.Printf("用户 %d 加入房间 %d", client.UserID, *client.RoomID)
	} else {
		log.Printf("用户 %d 建立WebSocket连接", client.UserID)
	}

	// 设置用户在线状态
	redis.SetUserOnline(client.UserID)
}

// unregisterClient 注销客户端
func (h *Hub) unregisterClient(client *Client) {
	h.Mutex.Lock()
	defer h.Mutex.Unlock()

	if _, ok := h.Clients[client]; ok {
		delete(h.Clients, client)
		close(client.Send)

		// 从用户映射中移除
		delete(h.Users, client.UserID)

		// 如果在房间中，从房间中移除
		if client.RoomID != nil {
			if clients, exists := h.Rooms[*client.RoomID]; exists {
				delete(clients, client)
				if len(clients) == 0 {
					delete(h.Rooms, *client.RoomID)
				}
			}
			redis.RemoveUserFromRoom(*client.RoomID, client.UserID)
			log.Printf("用户 %d 离开房间 %d", client.UserID, *client.RoomID)
		} else {
			log.Printf("用户 %d 断开WebSocket连接", client.UserID)
		}

		// 设置用户离线状态
		redis.SetUserOffline(client.UserID)
	}
}

// SendPrivateMessage 发送私聊消息给指定用户
func (h *Hub) SendPrivateMessage(userID int64, message []byte) {
	h.Mutex.RLock()
	defer h.Mutex.RUnlock()

	if client, exists := h.Users[userID]; exists {
		select {
		case client.Send <- message:
		default:
			close(client.Send)
			delete(h.Clients, client)
			delete(h.Users, userID)
		}
	}
}

// broadcastMessage 广播消息
func (h *Hub) broadcastMessage(message []byte) {
	h.Mutex.RLock()
	defer h.Mutex.RUnlock()

	// 解析消息获取房间ID
	var wsMsg WSMessage
	if err := json.Unmarshal(message, &wsMsg); err != nil {
		log.Printf("解析消息失败: %v", err)
		return
	}

	// 向指定房间的所有客户端发送消息
	if room, exists := h.Rooms[wsMsg.RoomID]; exists {
		for client := range room {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(h.Clients, client)
				delete(room, client)
			}
		}
	}
}

// BroadcastToRoom 向指定房间广播消息
func (h *Hub) BroadcastToRoom(roomID int64, message []byte) {
	h.Mutex.RLock()
	defer h.Mutex.RUnlock()

	if room, exists := h.Rooms[roomID]; exists {
		for client := range room {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(h.Clients, client)
				delete(room, client)
			}
		}
	}
}

// SendToUser 向指定用户发送消息
func (h *Hub) SendToUser(userID int64, message []byte) {
	h.Mutex.RLock()
	defer h.Mutex.RUnlock()

	if client, exists := h.Users[userID]; exists {
		select {
		case client.Send <- message:
		default:
			close(client.Send)
			delete(h.Clients, client)
			delete(h.Users, userID)
			if client.RoomID != nil {
				if room, exists := h.Rooms[*client.RoomID]; exists {
					delete(room, client)
				}
			}
		}
	}
}

// HandleWebSocket 处理WebSocket连接
func (h *Hub) HandleWebSocket(c *gin.Context) {
	// 获取用户ID（必需）
	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少用户ID"})
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	// 验证用户是否存在
	db := database.GetDB()
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 获取房间ID（可选，用于群聊）
	roomIDStr := c.Query("room_id")
	var roomID *int64
	if roomIDStr != "" {
		parsedRoomID, err := strconv.ParseInt(roomIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的房间ID"})
			return
		}

		// 验证房间是否存在
		var chatRoom models.ChatRoom
		if err := db.First(&chatRoom, parsedRoomID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "聊天室不存在"})
			return
		}

		// 验证用户是否是房间成员
		var member models.ChatRoomMember
		if err := db.Where("chat_room_id = ? AND user_id = ?", parsedRoomID, userID).First(&member).Error; err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "您不是该聊天室的成员"})
			return
		}

		roomID = &parsedRoomID
	}

	// 升级HTTP连接为WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket升级失败: %v", err)
		return
	}

	// 创建客户端
	client := &Client{
		Conn:   conn,
		UserID: userID,
		RoomID: roomID,
		Send:   make(chan []byte, 256),
	}

	// 注册客户端
	h.Register <- client

	// 启动读写协程
	go client.writePump()
	go client.readPump(h)
}

// readPump 读取消息
func (c *Client) readPump(hub *Hub) {
	defer func() {
		hub.Unregister <- c
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket错误: %v", err)
			}
			break
		}

		// 处理接收到的消息
		var wsMsg WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			log.Printf("解析消息失败: %v", err)
			continue
		}

		// 设置发送者信息
		wsMsg.UserID = c.UserID
		if c.RoomID != nil {
			wsMsg.RoomID = *c.RoomID
		}

		// 重新序列化消息
		messageData, err := json.Marshal(wsMsg)
		if err != nil {
			log.Printf("序列化消息失败: %v", err)
			continue
		}

		// 广播消息
		hub.Broadcast <- messageData
	}
}

// writePump 发送消息
func (c *Client) writePump() {
	defer c.Conn.Close()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("发送消息失败: %v", err)
				return
			}
		}
	}
}
