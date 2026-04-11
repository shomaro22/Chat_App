// Package controller は HTTP / WebSocket のハンドラを定義する。
//
// 認証: ルーターで JWT ミドルウェア（Authorization: Bearer またはクエリ access_token）が先に走り、
// 保護されたハンドラは controller.CurrentUserID などでコンテキスト上のユーザーを参照する。
// 公開ルートは POST /auth/login と POST /auth/register のみ。
//
// WebSocket は同じ JWT 前提で接続し、サーバーから message_created / message_deleted / read_updated /
// presence_updated などをブロードキャストする（契約は backend/docs/ws_events.md を参照）。
package controller
