package router

import (
	"net/http"

	"chat-app/controller"

	"github.com/labstack/echo/v4"
)

// New takes room and message controllers and wires them into an Echo instance (health, room, and message routes).
func New(room *controller.RoomController, messages *controller.MessageController) *echo.Echo {
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
	e.DELETE("/messages/:id", messages.DeleteMessage)
	return e
}
