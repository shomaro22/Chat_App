package controller

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSessionReadyPayloadJSON(t *testing.T) {
	uid := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	b, err := json.Marshal(wsSessionReadyOutbound{
		Type:          WSEventSessionReady,
		SchemaVersion: WSEventSchemaVersion,
		UserID:        uid,
	})
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(b, &out))
	require.Equal(t, WSEventSessionReady, out["type"])
	require.EqualValues(t, WSEventSchemaVersion, out["schema_version"])
	require.Equal(t, uid.String(), out["user_id"])
}

func TestWSErrorPayloadWithRoomIDJSON(t *testing.T) {
	rid := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	rs := rid.String()
	b, err := json.Marshal(wsErrorOutbound{
		Type:          WSEventError,
		SchemaVersion: WSEventSchemaVersion,
		Code:          WSCodeRoomFrozen,
		Message:       "room is frozen",
		RoomID:        &rs,
	})
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(b, &out))
	require.Equal(t, "error", out["type"])
	require.Equal(t, "ROOM_FROZEN", out["code"])
	require.Equal(t, rs, out["room_id"])
}

func TestPresenceUpdatedPayloadJSON(t *testing.T) {
	uid := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	b, err := json.Marshal(presenceUpdatedPayload{
		Type:          WSEventPresenceUpdated,
		SchemaVersion: WSEventSchemaVersion,
		UserID:        uid,
		IsOnline:      true,
	})
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(b, &out))
	require.Equal(t, WSEventPresenceUpdated, out["type"])
	require.EqualValues(t, WSEventSchemaVersion, out["schema_version"])
	require.Equal(t, uid.String(), out["user_id"])
	require.Equal(t, true, out["is_online"])
}

func TestPongOutboundJSON(t *testing.T) {
	b, err := json.Marshal(wsPongOutbound{Type: WSEventPong, SchemaVersion: WSEventSchemaVersion})
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(b, &out))
	require.Equal(t, WSEventPong, out["type"])
	require.EqualValues(t, WSEventSchemaVersion, out["schema_version"])
}
