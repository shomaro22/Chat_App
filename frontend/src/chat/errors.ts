export function toErrorMessage(status: number, fallback: string): string {
  if (status === 401) return '認証が切れました。再ログインしてください。';
  if (status === 403) return 'この操作は許可されていません。';
  if (status === 404) return '対象が見つかりません。';
  if (status === 429) return 'リクエストが多すぎます。少し待って再試行してください。';
  return fallback;
}
