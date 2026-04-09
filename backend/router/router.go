package router

import (
	"net/http"

	"chat-app/controller"

	"github.com/labstack/echo/v4"
)

// New takes room, message, WebSocket, and message-read controllers and wires them into an Echo instance.
func New(room *controller.RoomController, messages *controller.MessageController, ws *controller.WSController, reads *controller.MessageReadController) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	e.GET("/rooms", room.ListJoinedRooms)
	e.POST("/rooms/direct", room.CreateDirectRoom)
	e.POST("/rooms/groups", room.CreateGroupRoom)
	e.POST("/rooms/:id/messages", messages.CreateMessage)
	e.GET("/rooms/:id/messages", messages.ListMessages)
	e.POST("/rooms/:id/read", reads.MarkAsRead)
	e.DELETE("/messages/:id", messages.DeleteMessage)
	e.GET("/ws", ws.Handle)
	return e
}
