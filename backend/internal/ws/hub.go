// Package ws 实现 WebSocket 连接管理与家庭房间广播。
// 架构：Client（连接） → Hub（中心管理器，按家庭分房间） → 广播事件。
// Phase 1 单实例内存广播；多实例部署时通过 Redis Pub/Sub 扩展（见架构文档）。
package ws

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// 消息类型
const (
	MsgTypeRecordCreated = "record.created"
	MsgTypeRecordUpdated = "record.updated"
	MsgTypeRecordDeleted = "record.deleted"
	MsgTypeMemberJoined  = "member.joined"
	MsgTypePing          = "ping"
	MsgTypePong          = "pong"
)

// Message WebSocket 消息协议（与 API 接口文档 6.2 节一致）
type Message struct {
	Type      string `json:"type"`
	FamilyID  int64  `json:"family_id"`
	SenderID  int64  `json:"sender_id"`
	Timestamp int64  `json:"timestamp"`
	Data      any    `json:"data,omitempty"`
}

// Client 单个 WebSocket 连接
type Client struct {
	Hub       *Hub
	Conn      *websocket.Conn
	Send      chan []byte
	UserID    int64
	FamilyID  int64
	CloseOnce sync.Once
}

// Hub 连接中心管理器
type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]struct{}     // 全部连接
	familyRoom map[int64]map[*Client]struct{} // familyID -> 连接集合
	register   chan *Client
	unregister chan *Client
	broadcast  chan *Message
}

// NewHub 创建 Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]struct{}),
		familyRoom: make(map[int64]map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Message, 256),
	}
}

// Run 启动 Hub 主循环（goroutine）
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = struct{}{}
			if h.familyRoom[c.FamilyID] == nil {
				h.familyRoom[c.FamilyID] = make(map[*Client]struct{})
			}
			h.familyRoom[c.FamilyID][c] = struct{}{}
			h.mu.Unlock()
			log.Printf("[ws] 用户%d加入家庭%d房间，当前在线=%d", c.UserID, c.FamilyID, h.familyOnlineCount(c.FamilyID))

		case c := <-h.unregister:
			h.removeClient(c)

		case msg := <-h.broadcast:
			h.broadcastToFamily(msg)
		}
	}
}

// BroadcastToFamily 向指定家庭房间广播消息（异步）
func (h *Hub) BroadcastToFamily(msg *Message) {
	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().Unix()
	}
	h.broadcast <- msg
}

func (h *Hub) removeClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	c.CloseOnce.Do(func() { close(c.Send) })
	delete(h.clients, c)
	if room, ok := h.familyRoom[c.FamilyID]; ok {
		delete(room, c)
		if len(room) == 0 {
			delete(h.familyRoom, c.FamilyID)
		}
	}
}

func (h *Hub) broadcastToFamily(msg *Message) {
	h.mu.RLock()
	room := h.familyRoom[msg.FamilyID]
	targets := make([]*Client, 0, len(room))
	for c := range room {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[ws] 消息序列化失败: %v", err)
		return
	}

	sent := 0
	for _, c := range targets {
		// 不回发给消息发送者本人（其本地已有最新数据）
		if c.UserID == msg.SenderID {
			continue
		}
		select {
		case c.Send <- data:
			sent++
		default: // 发送缓冲满，判定为慢连接，踢掉
			h.removeClient(c)
		}
	}
	if sent > 0 {
		log.Printf("[ws] 家庭%d 广播 %s -> %d 个在线成员", msg.FamilyID, msg.Type, sent)
	}
}

// OnlineMembers 查询家庭在线人数
func (h *Hub) OnlineMembers(familyID int64) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.familyOnlineCount(familyID)
}

func (h *Hub) familyOnlineCount(familyID int64) int {
	return len(h.familyRoom[familyID])
}
