package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"

	"github.com/yantx/baby-care-workbench/backend/internal/config"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
	sendBufferSize = 64
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	// 小程序请求不带 Origin，直接放行（生产环境需校验）
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WSClaims WebSocket 连接鉴权（复用 JWT）
type WSClaims struct {
	UserID   int64 `json:"user_id"`
	FamilyID int64 `json:"family_id"`
	jwt.RegisteredClaims
}

// ParseToken 解析并校验 JWT
func ParseToken(tokenStr string) (*WSClaims, error) {
	claims := &WSClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		return []byte(config.Get().JWT.Secret), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	return claims, nil
}

// HandleWS /ws?token=xxx 连接入口
func HandleWS(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.Query("token")
		claims, err := ParseToken(tokenStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 40101, "message": "无效的 token"})
			return
		}
		if claims.FamilyID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "尚未加入家庭，无法建立实时连接"})
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("[ws] 升级连接失败: %v", err)
			return
		}

		client := &Client{
			Hub:      hub,
			Conn:     conn,
			Send:     make(chan []byte, sendBufferSize),
			UserID:   claims.UserID,
			FamilyID: claims.FamilyID,
		}
		hub.register <- client

		go client.writePump()
		go client.readPump()
	}
}

// readPump 读循环：处理客户端消息 + 心跳
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		return c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, data, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		switch msg.Type {
		case MsgTypePing:
			pong, _ := json.Marshal(Message{Type: MsgTypePong, Timestamp: time.Now().Unix()})
			select {
			case c.Send <- pong:
			default:
			}
		default:
			// Phase 1 客户端只收不发业务消息，其他类型忽略
		}
	}
}

// writePump 写循环：发送消息 + 心跳
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case data, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
