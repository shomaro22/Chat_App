package wshub

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type PresenceChange struct {
	Changed  bool
	UserID   uuid.UUID
	IsOnline bool
}

type ConnectionManager struct {
	mu            sync.RWMutex
	clients       map[uuid.UUID]*wsClient
	userConnCount map[uuid.UUID]int
	roomSubs      map[uuid.UUID]map[uuid.UUID]struct{}
}

type wsClient struct {
	connID  uuid.UUID
	userID  uuid.UUID
	conn    *websocket.Conn
	writeMu sync.Mutex
	rooms   map[uuid.UUID]struct{}
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		clients:       make(map[uuid.UUID]*wsClient),
		userConnCount: make(map[uuid.UUID]int),
		roomSubs:      make(map[uuid.UUID]map[uuid.UUID]struct{}),
	}
}

func (m *ConnectionManager) Register(userID uuid.UUID, conn *websocket.Conn) (connID uuid.UUID, change PresenceChange) {
	id := uuid.New()
	c := &wsClient{
		connID: id,
		userID: userID,
		conn:   conn,
		rooms:  make(map[uuid.UUID]struct{}),
	}
	m.mu.Lock()
	m.clients[id] = c
	before := m.userConnCount[userID]
	m.userConnCount[userID] = before + 1
	m.mu.Unlock()

	if before == 0 {
		change = PresenceChange{Changed: true, UserID: userID, IsOnline: true}
	}
	return id, change
}

// IsUserOnline は当該ユーザーに少なくとも1件の接続があるかを返す。
func (m *ConnectionManager) IsUserOnline(userID uuid.UUID) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.userConnCount[userID] > 0
}

// removeConnLocked は mu がロック済みであること。登録を即時に外し、購読も解く（管理者切断と通常切断の共通処理）。
func (m *ConnectionManager) removeConnLocked(connID uuid.UUID) (change PresenceChange, wsConn *websocket.Conn) {
	c, ok := m.clients[connID]
	if !ok {
		return PresenceChange{}, nil
	}
	wsConn = c.conn
	for roomID := range c.rooms {
		if subs, ok := m.roomSubs[roomID]; ok {
			delete(subs, connID)
			if len(subs) == 0 {
				delete(m.roomSubs, roomID)
			}
		}
	}
	delete(m.clients, connID)

	before := m.userConnCount[c.userID]
	if before <= 1 {
		delete(m.userConnCount, c.userID)
	} else {
		m.userConnCount[c.userID] = before - 1
	}
	if before == 1 {
		return PresenceChange{Changed: true, UserID: c.userID, IsOnline: false}, wsConn
	}
	return PresenceChange{}, wsConn
}

func (m *ConnectionManager) RemoveConn(connID uuid.UUID) (change PresenceChange) {
	m.mu.Lock()
	defer m.mu.Unlock()
	change, _ = m.removeConnLocked(connID)
	return change
}

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

func (m *ConnectionManager) BroadcastToUsersInRooms(roomIDs []uuid.UUID, payload []byte) {
	if len(roomIDs) == 0 || len(payload) == 0 {
		return
	}
	m.mu.RLock()
	seen := make(map[uuid.UUID]struct{})
	var targets []*wsClient
	for _, rid := range roomIDs {
		subs := m.roomSubs[rid]
		for cid := range subs {
			if _, ok := seen[cid]; ok {
				continue
			}
			seen[cid] = struct{}{}
			if c, ok := m.clients[cid]; ok {
				targets = append(targets, c)
			}
		}
	}
	m.mu.RUnlock()

	for _, c := range targets {
		c.writeMu.Lock()
		c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		_ = c.conn.WriteMessage(websocket.TextMessage, payload)
		c.writeMu.Unlock()
	}
}

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

// CloseUserConnections は指定ユーザーの接続をハブから即時に外し（購読も解く）、その後ソケットを閉じる。
// 戻り値は「最後の接続が落ちた」等、RemoveConn と同様のプレゼンス境界（管理者切断後の presence_updated 用）。
func (m *ConnectionManager) CloseUserConnections(userID uuid.UUID) (change PresenceChange) {
	m.mu.Lock()
	var ids []uuid.UUID
	for id, c := range m.clients {
		if c.userID == userID {
			ids = append(ids, id)
		}
	}
	var toClose []*websocket.Conn
	var last PresenceChange
	for _, id := range ids {
		ch, conn := m.removeConnLocked(id)
		if conn != nil {
			toClose = append(toClose, conn)
		}
		if ch.Changed {
			last = ch
		}
	}
	m.mu.Unlock()
	for _, conn := range toClose {
		if conn != nil {
			_ = conn.Close()
		}
	}
	return last
}

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
