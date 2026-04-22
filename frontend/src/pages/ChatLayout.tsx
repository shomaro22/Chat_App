import { useCallback, useEffect, useState } from 'react';
import { Route, Routes, useLocation, useNavigate } from 'react-router-dom';
import { fetchWithAuth } from '../lib/api';
import { ROOMS_REFRESH_EVENT } from '../lib/roomsRefresh';
import { toErrorMessage } from '../chat/errors';
import { RoomCreateModal } from '../components/RoomCreateModal';
import { RoomList } from '../components/RoomList';
import type { MeUser, RoomSummary } from '../types/chat';
import { RoomThread } from './RoomThread';

export function ChatLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const [rooms, setRooms] = useState<RoomSummary[]>([]);
  const [me, setMe] = useState<MeUser | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [createKind, setCreateKind] = useState<'direct' | 'group'>('direct');
  const [directPeerRef, setDirectPeerRef] = useState('');
  const [groupDisplayName, setGroupDisplayName] = useState('');
  const [groupMemberRefsText, setGroupMemberRefsText] = useState('');
  const [roomError, setRoomError] = useState<string | null>(null);
  const [creatingRoom, setCreatingRoom] = useState(false);

  useEffect(() => {
    if (!showCreateModal) return;
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setShowCreateModal(false);
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [showCreateModal]);

  const loadRooms = useCallback(async () => {
    const res = await fetchWithAuth('/rooms');
    if (res.status === 401) {
      return;
    }
    if (!res.ok) {
      setRoomError(toErrorMessage(res.status, 'ルーム一覧の取得に失敗しました。'));
      return;
    }
    const data = await res.json();
    setRooms(data.rooms ?? data.Rooms ?? []);
  }, []);

  const loadMe = useCallback(async () => {
    const res = await fetchWithAuth('/me');
    if (res.status === 401) {
      return;
    }
    if (!res.ok) return;
    const data = await res.json().catch(() => ({}));
    const u = data.user as MeUser | undefined;
    if (u?.id && u.email) {
      setMe(u);
    }
  }, []);

  useEffect(() => {
    void loadRooms();
    void loadMe();
  }, [loadRooms, loadMe]);

  useEffect(() => {
    if (me?.role === 'admin') {
      navigate('/admin', { replace: true });
    }
  }, [me, navigate]);

  useEffect(() => {
    const p = location.pathname.replace(/\/$/, '');
    if (p === '/rooms') {
      void loadRooms();
    }
  }, [location.pathname, loadRooms]);

  useEffect(() => {
    const onRefresh = () => void loadRooms();
    window.addEventListener(ROOMS_REFRESH_EVENT, onRefresh);
    return () => window.removeEventListener(ROOMS_REFRESH_EVENT, onRefresh);
  }, [loadRooms]);

  const handleCreateRoom = async (): Promise<string | null> => {
    setCreatingRoom(true);
    setRoomError(null);
    try {
      if (createKind === 'direct') {
        if (!directPeerRef.trim()) {
          setRoomError('相手のメールアドレスまたはユーザーIDを入力してください。');
          return null;
        }
        const res = await fetchWithAuth('/rooms/direct', {
          method: 'POST',
          body: JSON.stringify({ target_user_id: directPeerRef.trim() }),
        });
        const data = await res.json().catch(() => ({}));
        if (!res.ok) {
          if (res.status === 401) return null;
          setRoomError(toErrorMessage(res.status, data.error ?? 'ダイレクトルーム作成に失敗しました。'));
          return null;
        }
        if (!data.id) {
          setRoomError('作成レスポンスに room id がありません。');
          return null;
        }
        const createdRoomId = String(data.id);
        setDirectPeerRef('');
        setGroupDisplayName('');
        setGroupMemberRefsText('');
        await loadRooms();
        return createdRoomId;
      }
      const memberIds = groupMemberRefsText
        .split(/[,\n]/)
        .map((v) => v.trim())
        .filter(Boolean);
      if (!groupDisplayName.trim()) {
        setRoomError('グループ名を入力してください。');
        return null;
      }
      if (memberIds.length === 0) {
        setRoomError('参加メンバー（メールまたはユーザーID）を1件以上入力してください。');
        return null;
      }
      const res = await fetchWithAuth('/rooms/groups', {
        method: 'POST',
        body: JSON.stringify({ name: groupDisplayName.trim(), member_ids: memberIds }),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        if (res.status === 401) return null;
        setRoomError(toErrorMessage(res.status, data.error ?? 'グループルーム作成に失敗しました。'));
        return null;
      }
      if (!data.id) {
        setRoomError('作成レスポンスに room id がありません。');
        return null;
      }
      const createdRoomId = String(data.id);
      setDirectPeerRef('');
      setGroupDisplayName('');
      setGroupMemberRefsText('');
      await loadRooms();
      return createdRoomId;
    } finally {
      setCreatingRoom(false);
    }
  };

  return (
    <div className="chat-layout">
      <div className="chat-shell">
        <RoomList rooms={rooms} me={me} onOpenCreateModal={() => setShowCreateModal(true)} />
        <main className="chat-main">
          <Routes>
            <Route path=":id" element={<RoomThread />} />
            <Route path="*" element={<div className="chat-empty">ルームを選択してください</div>} />
          </Routes>
        </main>
      </div>
      <RoomCreateModal
        open={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        createKind={createKind}
        onCreateKindChange={setCreateKind}
        directPeerRef={directPeerRef}
        onDirectPeerRefChange={(v) => {
          setDirectPeerRef(v);
          setRoomError(null);
        }}
        groupDisplayName={groupDisplayName}
        onGroupDisplayNameChange={setGroupDisplayName}
        groupMemberRefsText={groupMemberRefsText}
        onGroupMemberRefsTextChange={(v) => {
          setGroupMemberRefsText(v);
          setRoomError(null);
        }}
        roomError={roomError}
        creatingRoom={creatingRoom}
        onSubmit={() => void handleCreateRoom().then((rid) => {
          if (rid) {
            setShowCreateModal(false);
            navigate(`/rooms/${rid}`);
          }
        })}
        submitDisabled={
          creatingRoom ||
          (createKind === 'direct' && !directPeerRef.trim()) ||
          (createKind === 'group' && (!groupDisplayName.trim() || !groupMemberRefsText.trim()))
        }
      />
    </div>
  );
}
