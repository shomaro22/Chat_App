// Package controller defines HTTP and WebSocket handlers. JWT is not wired yet; room, message,
// read-position, and WS endpoints accept user IDs via query parameters or JSON bodies for development (replace with auth later).
// WebSocket events include message_created, read_updated, and presence_updated (user online/offline per joined rooms).
package controller
