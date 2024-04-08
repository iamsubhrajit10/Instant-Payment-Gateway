package paymenthandler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

func TransferHandler(c echo.Context) error {
	//reply that i am responsible for transfer
	time.Sleep(1 * time.Second)
	return c.String(http.StatusOK, "I am responsible for transfer")
}
