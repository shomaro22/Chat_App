package wshub

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestPresence_NoDuplicateSend(t *testing.T) {
	m := NewConnectionManager()
	u := uuid.New()
	r1, r2 := uuid.New(), uuid.New()
	upgrader := websocket.Upgrader{}
	ready := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		id, _ := m.Register(u, c)
		m.Subscribe(id, r1)
		m.Subscribe(id, r2)
		close(ready)
		for {
			_, _, err := c.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	client, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer client.Close()

	<-ready

	payload := []byte(`{"type":"presence_updated","test":true}`)
	m.BroadcastToUsersInRooms([]uuid.UUID{r1, r2}, payload)

	_, msg, err := client.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, payload, msg)

	require.NoError(t, client.SetReadDeadline(time.Now().Add(80*time.Millisecond)))
	_, _, err = client.ReadMessage()
	require.Error(t, err)
}
