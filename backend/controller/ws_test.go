package controller

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPresenceUpdatedPayloadJSON(t *testing.T) {
	uid := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	b, err := json.Marshal(presenceUpdatedPayload{
		Type:          WSEventPresenceChanged,
		SchemaVersion: WSEventSchemaVersion,
		UserID:        uid,
		IsOnline:      true,
	})
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(b, &out))
	require.Equal(t, WSEventPresenceChanged, out["type"])
	require.EqualValues(t, WSEventSchemaVersion, out["schema_version"])
	require.Equal(t, uid.String(), out["user_id"])
	require.Equal(t, true, out["is_online"])
}
