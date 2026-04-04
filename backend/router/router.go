package router

import (
	"net/http"

	"chat-app/controller"

	"github.com/labstack/echo/v4"
)

// New builds Echo with health check and room routes. Pass a non-nil RoomController.
func New(room *controller.RoomController) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	e.GET("/rooms", room.ListJoinedRooms)
	e.POST("/rooms/direct", room.CreateDirectRoom)
	e.POST("/rooms/groups", room.CreateGroupRoom)
	return e
}
