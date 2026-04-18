/** サイドバー用ルーム一覧を親（ChatLayout）が再取得するためのブラウザイベント名 */
export const ROOMS_REFRESH_EVENT = 'chat:rooms-refresh';

export function requestRoomsRefresh(): void {
  window.dispatchEvent(new Event(ROOMS_REFRESH_EVENT));
}
