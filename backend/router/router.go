// Package router registers HTTP routes on Echo.
package router

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// New returns an Echo instance with global routes (e.g. health).
func New() *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	return e
}
