package wshub

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// ConnectionManager tracks WebSocket clients by connection id, and which rooms each client is subscribed to.
// It is safe for concurrent use. Real-time delivery only; business rules live in controllers / use cases.
type ConnectionManager struct {
	mu sync.RWMutex
	// connID -> live client
	clients map[uuid.UUID]*wsClient
	// roomID -> set of connIDs subscribed to that room
	roomSubs map[uuid.UUID]map[uuid.UUID]struct{}
}

type wsClient struct {
	connID uuid.UUID
	userID uuid.UUID
	conn   *websocket.Conn
	// writeMu serializes WriteMessage / WriteControl (gorilla/websocket requirement).
	writeMu sync.Mutex
	rooms   map[uuid.UUID]struct{}
}

// NewConnectionManager creates an empty manager.
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		clients:  make(map[uuid.UUID]*wsClient),
		roomSubs: make(map[uuid.UUID]map[uuid.UUID]struct{}),
	}
}

// Register binds a WebSocket to a user and returns a new connection id. Caller must call RemoveConn when the socket closes.
func (m *ConnectionManager) Register(userID uuid.UUID, conn *websocket.Conn) uuid.UUID {
	id := uuid.New()
	c := &wsClient{
		connID: id,
		userID: userID,
		conn:   conn,
		rooms:  make(map[uuid.UUID]struct{}),
	}
	m.mu.Lock()
	m.clients[id] = c
	m.mu.Unlock()
	return id
}

// RemoveConn drops the connection and all room subscriptions. It does not close the underlying socket.
func (m *ConnectionManager) RemoveConn(connID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.clients[connID]
	if !ok {
		return
	}
	for roomID := range c.rooms {
		if subs, ok := m.roomSubs[roomID]; ok {
			delete(subs, connID)
			if len(subs) == 0 {
				delete(m.roomSubs, roomID)
			}
		}
	}
	delete(m.clients, connID)
}

// Subscribe adds connID to roomID. No authorization here — caller must verify membership first.
func (m *ConnectionManager) Subscribe(connID, roomID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.clients[connID]
	if !ok {
		return
	}
	c.rooms[roomID] = struct{}{}
	if m.roomSubs[roomID] == nil {
		m.roomSubs[roomID] = make(map[uuid.UUID]struct{})
	}
	m.roomSubs[roomID][connID] = struct{}{}
}

// Unsubscribe removes connID from roomID (idempotent).
func (m *ConnectionManager) Unsubscribe(connID, roomID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.clients[connID]
	if !ok {
		return
	}
	delete(c.rooms, roomID)
	if subs, ok := m.roomSubs[roomID]; ok {
		delete(subs, connID)
		if len(subs) == 0 {
			delete(m.roomSubs, roomID)
		}
	}
}

// BroadcastToRoom sends a text payload to every subscriber of roomID. Failed writes are ignored per client.
func (m *ConnectionManager) BroadcastToRoom(roomID uuid.UUID, payload []byte) {
	m.mu.RLock()
	subs := m.roomSubs[roomID]
	if len(subs) == 0 {
		m.mu.RUnlock()
		return
	}
	list := make([]*wsClient, 0, len(subs))
	for cid := range subs {
		if c, ok := m.clients[cid]; ok {
			list = append(list, c)
		}
	}
	m.mu.RUnlock()

	for _, c := range list {
		c.writeMu.Lock()
		c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		_ = c.conn.WriteMessage(websocket.TextMessage, payload)
		c.writeMu.Unlock()
	}
}

// SendPing writes a WebSocket ping control frame. Returns false if the connection is gone or the write fails.
func (m *ConnectionManager) SendPing(connID uuid.UUID) bool {
	m.mu.RLock()
	c, ok := m.clients[connID]
	m.mu.RUnlock()
	if !ok {
		return false
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err := c.conn.WriteMessage(websocket.PingMessage, nil)
	return err == nil
}

// WriteText sends a single text frame with the same serialization rules as BroadcastToRoom (one writer at a time).
func (m *ConnectionManager) WriteText(connID uuid.UUID, payload []byte) bool {
	m.mu.RLock()
	c, ok := m.clients[connID]
	m.mu.RUnlock()
	if !ok {
		return false
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err := c.conn.WriteMessage(websocket.TextMessage, payload)
	return err == nil
}
