import { useCallback } from "react";
import { fetchWithAuth } from "../lib/api";
import { requestRoomsRefresh } from "../lib/roomsRefresh";
import { toErrorMessage } from "../chat/errors";
import type { Message } from "../types/chat";

type APIErrorBody = {
  error?: string;
  code?: string;
};

type UploadResponse = {
  url?: string;
  filename?: string;
  content_type?: string;
};

type SendResponse = {
  message?: Message;
  error?: string;
};

type UseRoomComposerActionsParams = {
  roomId?: string;
  input: string;
  replyTo: Message | null;
  setInput: (value: string) => void;
  setReplyTo: (message: Message | null) => void;
  setError: (value: string | null) => void;
  setLoading: (value: boolean) => void;
  setUploading: (value: boolean) => void;
  upsertMessage: (message: Message) => void;
  loadRoomDetail: () => Promise<void>;
};

function formatUploadError(
  status: number,
  body: APIErrorBody,
  kind: "image" | "file",
): string {
  if (status === 413 || body.code === "UPLOAD_TOO_LARGE") {
    return kind === "image"
      ? "画像サイズが上限を超えています（最大10MB）。"
      : "ファイルサイズが上限を超えています（最大25MB）。";
  }
  if (status === 415 || body.code === "UNSUPPORTED_MEDIA") {
    return kind === "image"
      ? "画像形式が未対応です。jpg / png / gif / webp を選択してください。"
      : "ファイル形式が未対応です。pdf / txt / zip を選択してください。";
  }
  if (body.error?.trim()) return body.error;
  return toErrorMessage(
    status,
    kind === "image"
      ? "画像アップロードに失敗しました。"
      : "ファイルのアップロードに失敗しました。",
  );
}

export function useRoomComposerActions({
  roomId,
  input,
  replyTo,
  setInput,
  setReplyTo,
  setError,
  setLoading,
  setUploading,
  upsertMessage,
  loadRoomDetail,
}: UseRoomComposerActionsParams) {
  const sendMessagePayload = useCallback(
    async (body: Record<string, string>): Promise<boolean> => {
      if (!roomId) return false;
      const res = await fetchWithAuth(`/rooms/${roomId}/messages`, {
        method: "POST",
        body: JSON.stringify(body),
      });
      const data = (await res.json().catch(() => ({}))) as SendResponse;
      if (res.ok && data.message) {
        upsertMessage(data.message);
        setInput("");
        setReplyTo(null);
        void loadRoomDetail();
        requestRoomsRefresh();
        return true;
      }
      if (!res.ok) {
        if (res.status === 401) return false;
        setError(toErrorMessage(res.status, "メッセージ送信に失敗しました。"));
      }
      return false;
    },
    [roomId, upsertMessage, setInput, setReplyTo, loadRoomDetail, setError],
  );

  const handleSend = useCallback(async () => {
    if (!roomId || !input.trim()) return;
    setLoading(true);
    setError(null);
    try {
      const body: Record<string, string> = { content: input.trim() };
      if (replyTo) body.parent_message_id = replyTo.id;
      await sendMessagePayload(body);
    } finally {
      setLoading(false);
    }
  }, [roomId, input, replyTo, setLoading, setError, sendMessagePayload]);

  const handleImageFile = useCallback(
    async (file: File) => {
      if (!roomId) return;
      setUploading(true);
      setError(null);
      try {
        const form = new FormData();
        form.append("file", file);
        const uploadRes = await fetchWithAuth("/media/images", {
          method: "POST",
          body: form,
        });
        const uploadData = (await uploadRes.json().catch(() => ({}))) as UploadResponse &
          APIErrorBody;
        if (!uploadRes.ok || !uploadData.url) {
          if (uploadRes.status === 401) return;
          setError(formatUploadError(uploadRes.status, uploadData, "image"));
          return;
        }
        const sendBody: Record<string, string> = { image_url: uploadData.url };
        const caption = input.trim();
        if (caption) sendBody.content = caption;
        if (replyTo) sendBody.parent_message_id = replyTo.id;

        const res = await fetchWithAuth(`/rooms/${roomId}/messages`, {
          method: "POST",
          body: JSON.stringify(sendBody),
        });
        const data = (await res.json().catch(() => ({}))) as SendResponse;
        if (res.ok && data.message) {
          upsertMessage(data.message);
          setInput("");
          setReplyTo(null);
          void loadRoomDetail();
          requestRoomsRefresh();
        } else if (!res.ok) {
          if (res.status === 401) return;
          setError(
            data.error?.trim() ||
              toErrorMessage(res.status, "画像メッセージ送信に失敗しました。"),
          );
        }
      } catch {
        setError(
          "通信エラーのため画像を送信できませんでした。接続を確認して再試行してください。",
        );
      } finally {
        setUploading(false);
      }
    },
    [
      roomId,
      input,
      replyTo,
      setUploading,
      setError,
      upsertMessage,
      setInput,
      setReplyTo,
      loadRoomDetail,
    ],
  );

  const handleGenericFile = useCallback(
    async (file: File) => {
      if (!roomId) return;
      setUploading(true);
      setError(null);
      try {
        const form = new FormData();
        form.append("file", file);
        const uploadRes = await fetchWithAuth("/media/files", {
          method: "POST",
          body: form,
        });
        const uploadData = (await uploadRes.json().catch(() => ({}))) as UploadResponse &
          APIErrorBody;
        if (!uploadRes.ok || !uploadData.url) {
          if (uploadRes.status === 401) return;
          setError(formatUploadError(uploadRes.status, uploadData, "file"));
          return;
        }
        const caption = input.trim();
        const sendBody: Record<string, string> = {
          file_url: uploadData.url,
          file_name: uploadData.filename ?? file.name,
          file_content_type: uploadData.content_type ?? file.type,
        };
        if (caption) sendBody.content = caption;
        if (replyTo) sendBody.parent_message_id = replyTo.id;

        const res = await fetchWithAuth(`/rooms/${roomId}/messages`, {
          method: "POST",
          body: JSON.stringify(sendBody),
        });
        const data = (await res.json().catch(() => ({}))) as SendResponse;
        if (res.ok && data.message) {
          upsertMessage(data.message);
          setInput("");
          setReplyTo(null);
          void loadRoomDetail();
          requestRoomsRefresh();
        } else if (!res.ok) {
          if (res.status === 401) return;
          setError(
            data.error?.trim() ||
              toErrorMessage(res.status, "ファイル付きメッセージの送信に失敗しました。"),
          );
        }
      } catch {
        setError(
          "通信エラーのためファイルを送信できませんでした。接続を確認して再試行してください。",
        );
      } finally {
        setUploading(false);
      }
    },
    [
      roomId,
      input,
      replyTo,
      setUploading,
      setError,
      upsertMessage,
      setInput,
      setReplyTo,
      loadRoomDetail,
    ],
  );

  return { handleSend, handleImageFile, handleGenericFile };
}
