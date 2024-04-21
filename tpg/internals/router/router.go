package router

import (
	"net/http"
	paymenthandler "tpg/internals/paymenthandler"

	"github.com/labstack/echo/v4"
)

// NewRouter creates a new instance of Echo router
func SetupRouter() *echo.Echo {
	e := echo.New()
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})
	e.POST("/transfer", paymenthandler.TransferHandler)
	return e
}
