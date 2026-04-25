import { useCallback } from "react";
import type { Dispatch, SetStateAction } from "react";
import { toErrorMessage } from "../chat/errors";
import { fetchWithAuth } from "../lib/api";
import { requestRoomsRefresh } from "../lib/roomsRefresh";

type RoomMetaState = {
  id: string;
  type: string;
  name: string | null;
  created_by: string;
  is_frozen: boolean;
} | null;

type UseRoomSettingsActionsParams = {
  roomId?: string;
  renameDraft: string;
  addMemberRef: string;
  selfUserId: string | null;
  roomType?: string;
  directPeerUserId?: string;
  peerBlocked: boolean;
  blockBusy: boolean;
  setRoomActionBusy: (value: boolean) => void;
  setRoomModalError: (value: string | null) => void;
  setRoomMeta: Dispatch<SetStateAction<RoomMetaState>>;
  setAddMemberRef: (value: string) => void;
  setShowRoomModal: (value: boolean) => void;
  setBlockBusy: (value: boolean) => void;
  setPeerBlocked: Dispatch<SetStateAction<boolean>>;
  loadRoomDetail: () => Promise<void>;
  loadMessagesAndReadState: () => Promise<void>;
  onExitRoom: () => void;
};

export function useRoomSettingsActions({
  roomId,
  renameDraft,
  addMemberRef,
  selfUserId,
  roomType,
  directPeerUserId,
  peerBlocked,
  blockBusy,
  setRoomActionBusy,
  setRoomModalError,
  setRoomMeta,
  setAddMemberRef,
  setShowRoomModal,
  setBlockBusy,
  setPeerBlocked,
  loadRoomDetail,
  loadMessagesAndReadState,
  onExitRoom,
}: UseRoomSettingsActionsParams) {
  const patchRoomName = useCallback(async () => {
    if (!roomId || !renameDraft.trim()) return;
    setRoomActionBusy(true);
    setRoomModalError(null);
    try {
      const res = await fetchWithAuth(`/rooms/${roomId}`, {
        method: "PATCH",
        body: JSON.stringify({ name: renameDraft.trim() }),
      });
      if (res.status === 401) return;
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setRoomModalError(
          toErrorMessage(
            res.status,
            (data as { error?: string }).error ?? "更新に失敗しました。",
          ),
        );
        return;
      }
      setRoomMeta((m) => (m ? { ...m, name: renameDraft.trim() } : m));
      requestRoomsRefresh();
    } finally {
      setRoomActionBusy(false);
    }
  }, [roomId, renameDraft, setRoomActionBusy, setRoomModalError, setRoomMeta]);

  const addMember = useCallback(async () => {
    if (!roomId || !addMemberRef.trim()) return;
    setRoomActionBusy(true);
    setRoomModalError(null);
    try {
      const res = await fetchWithAuth(`/rooms/${roomId}/members`, {
        method: "POST",
        body: JSON.stringify({ member_id: addMemberRef.trim() }),
      });
      if (res.status === 401) return;
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setRoomModalError(
          toErrorMessage(
            res.status,
            (data as { error?: string }).error ?? "追加に失敗しました。",
          ),
        );
        return;
      }
      setAddMemberRef("");
      await loadRoomDetail();
      requestRoomsRefresh();
    } finally {
      setRoomActionBusy(false);
    }
  }, [
    roomId,
    addMemberRef,
    setRoomActionBusy,
    setRoomModalError,
    setAddMemberRef,
    loadRoomDetail,
  ]);

  const removeMember = useCallback(
    async (targetUserId: string) => {
      if (!roomId || !targetUserId) return;
      setRoomActionBusy(true);
      setRoomModalError(null);
      try {
        const res = await fetchWithAuth(`/rooms/${roomId}/members/${targetUserId}`, {
          method: "DELETE",
        });
        if (res.status === 401) return;
        if (!res.ok) {
          const data = await res.json().catch(() => ({}));
          setRoomModalError(
            toErrorMessage(
              res.status,
              (data as { error?: string }).error ?? "操作に失敗しました。",
            ),
          );
          return;
        }
        if (selfUserId === targetUserId) {
          setShowRoomModal(false);
          requestRoomsRefresh();
          onExitRoom();
          return;
        }
        await loadRoomDetail();
        requestRoomsRefresh();
      } finally {
        setRoomActionBusy(false);
      }
    },
    [
      roomId,
      selfUserId,
      setRoomActionBusy,
      setRoomModalError,
      setShowRoomModal,
      onExitRoom,
      loadRoomDetail,
    ],
  );

  const closeRoom = useCallback(async () => {
    if (!roomId) return;
    setRoomActionBusy(true);
    setRoomModalError(null);
    try {
      const res = await fetchWithAuth(`/rooms/${roomId}`, { method: "DELETE" });
      if (res.status === 401) return;
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setRoomModalError(
          toErrorMessage(
            res.status,
            (data as { error?: string }).error ?? "クローズに失敗しました。",
          ),
        );
        return;
      }
      setShowRoomModal(false);
      requestRoomsRefresh();
      onExitRoom();
    } finally {
      setRoomActionBusy(false);
    }
  }, [roomId, setRoomActionBusy, setRoomModalError, setShowRoomModal, onExitRoom]);

  const toggleDirectPeerBlock = useCallback(async () => {
    if (roomType !== "direct" || !directPeerUserId || blockBusy) return;
    setBlockBusy(true);
    setRoomModalError(null);
    try {
      const res = await fetchWithAuth(`/users/${directPeerUserId}/block`, {
        method: peerBlocked ? "DELETE" : "POST",
      });
      if (res.status === 401) return;
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setRoomModalError(
          toErrorMessage(
            res.status,
            (data as { error?: string }).error ?? "ブロック操作に失敗しました。",
          ),
        );
        return;
      }
      setPeerBlocked((prev) => !prev);
      await loadMessagesAndReadState();
      await loadRoomDetail();
      requestRoomsRefresh();
    } finally {
      setBlockBusy(false);
    }
  }, [
    roomType,
    directPeerUserId,
    blockBusy,
    peerBlocked,
    setBlockBusy,
    setRoomModalError,
    setPeerBlocked,
    loadMessagesAndReadState,
    loadRoomDetail,
  ]);

  return {
    patchRoomName,
    addMember,
    removeMember,
    closeRoom,
    toggleDirectPeerBlock,
  };
}
