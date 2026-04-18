import { useCallback, useState } from 'react';
import { fetchWithAuth } from '../lib/api';
import type { RoomMemberRow } from '../types/chat';

export function useRoomDetail(roomId: string | undefined) {
  const [roomMembers, setRoomMembers] = useState<RoomMemberRow[]>([]);
  const [unreadCount, setUnreadCount] = useState<number | null>(null);
  const [myLastReadMessageId, setMyLastReadMessageId] = useState<string | null>(null);
  const [roomMeta, setRoomMeta] = useState<{
    id: string;
    type: string;
    name: string | null;
    created_by: string;
    is_frozen: boolean;
  } | null>(null);
  const [directPeer, setDirectPeer] = useState<{ user_id: string; name: string } | null>(null);

  const loadRoomDetail = useCallback(async () => {
    if (!roomId) return;
    const res = await fetchWithAuth(`/rooms/${roomId}`);
    if (res.status === 401) {
      return;
    }
    if (!res.ok) {
      setRoomMembers([]);
      setRoomMeta(null);
      setDirectPeer(null);
      setUnreadCount(null);
      setMyLastReadMessageId(null);
      return;
    }
    const data = await res.json().catch(() => ({}));
    const rawUnread = data.unread_count ?? data.UnreadCount;
    if (rawUnread === undefined || rawUnread === null || rawUnread === '') {
      setUnreadCount(null);
    } else {
      const n = typeof rawUnread === 'number' ? rawUnread : Number(rawUnread);
      setUnreadCount(Number.isFinite(n) ? n : null);
    }
    const rawLastRead = data.my_last_read_message_id ?? data.MyLastReadMessageID;
    setMyLastReadMessageId(rawLastRead != null && String(rawLastRead).trim() ? String(rawLastRead) : null);
    const roomRaw = data.room ?? data.Room;
    if (roomRaw && typeof roomRaw === 'object') {
      const rid = roomRaw.id ?? roomRaw.ID;
      const rtype = String(roomRaw.type ?? roomRaw.Type ?? '');
      const rname = roomRaw.name ?? roomRaw.Name ?? null;
      const created = String(roomRaw.created_by ?? roomRaw.CreatedBy ?? '');
      const frozen = Boolean(roomRaw.is_frozen ?? roomRaw.IsFrozen ?? false);
      if (rid) {
        setRoomMeta({
          id: String(rid),
          type: rtype,
          name: rname != null ? String(rname) : null,
          created_by: created,
          is_frozen: frozen,
        });
      } else {
        setRoomMeta(null);
      }
    } else {
      setRoomMeta(null);
    }
    const dp = data.direct_peer ?? data.DirectPeer;
    if (dp && (dp.user_id ?? dp.UserID)) {
      setDirectPeer({
        user_id: String(dp.user_id ?? dp.UserID),
        name: String(dp.name ?? dp.Name ?? ''),
      });
    } else {
      setDirectPeer(null);
    }
    const raw = data.members ?? data.Members ?? [];
    const rows: RoomMemberRow[] = raw.map(
      (m: { user_id?: string; UserID?: string; name?: string; Name?: string; online?: boolean; Online?: boolean }) => ({
        user_id: String(m.user_id ?? m.UserID ?? ''),
        name: String(m.name ?? m.Name ?? ''),
        online: Boolean(m.online ?? m.Online ?? false),
      }),
    );
    setRoomMembers(rows.filter((r) => r.user_id));
  }, [roomId]);

  return {
    roomMembers,
    setRoomMembers,
    roomMeta,
    setRoomMeta,
    directPeer,
    unreadCount,
    myLastReadMessageId,
    loadRoomDetail,
  };
}
