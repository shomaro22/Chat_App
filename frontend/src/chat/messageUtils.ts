/**
 * メッセージ一覧の整列方針（仕様）:
 * - 主キー: created_at（昇順）
 * - タイブレーク: id（文字列として昇順）
 * - WS やページングで到着順が崩れても、常にこの順へ正規化する（到着順の末尾追記だけにしない）
 */
import type { Message } from '../types/chat';

/** created_at 主、id 補助で昇順（表示用） */
export function compareMessagesAsc(a: Message, b: Message): number {
  const ta = new Date(a.created_at).getTime();
  const tb = new Date(b.created_at).getTime();
  if (ta !== tb) return ta - tb;
  return a.id.localeCompare(b.id);
}

export function sortMessagesAsc(list: Message[]): Message[] {
  return [...list].sort(compareMessagesAsc);
}

/** 1 件をマージし、常に created_at + id 昇順に整列する */
export function mergeMessageSorted(prev: Message[], next: Message): Message[] {
  const idx = prev.findIndex((m) => m.id === next.id);
  const copy = idx >= 0 ? prev.map((m, i) => (i === idx ? next : m)) : [...prev, next];
  return sortMessagesAsc(copy);
}

/**
 * 既存＋追加分を id で重複除去し、created_at + id 昇順に整列する。
 * WS やページングで順序が崩れないようにする（append だけに頼らない）。
 */
export function appendMessagesUniqueSorted(existing: Message[], added: Message[]): Message[] {
  const map = new Map<string, Message>();
  for (const m of existing) {
    map.set(m.id, m);
  }
  for (const m of added) {
    map.set(m.id, m);
  }
  return sortMessagesAsc([...map.values()]);
}

export function markMessageDeletedInList(prev: Message[], messageId: string): Message[] {
  return sortMessagesAsc(
    prev.map((m) =>
      m.id === messageId
        ? {
            ...m,
            is_deleted: true,
            content: null,
            image_url: null,
          }
        : m,
    ),
  );
}
