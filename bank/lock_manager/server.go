package lock_manager

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type Request struct {
	RequestType string   `json:"requestType"`
	Accounts    []string `json:"accounts"`
}

func StartServer() {
	e := echo.New()
	e.POST("/get-lock", func(c echo.Context) error {
		var req Request
		if err := c.Bind(req); err != nil {
			return err
		}
		if req.RequestType == "request" {
			accounts := GetLocksOnAvailableAccounts(req.Accounts)
			return c.JSON(http.StatusOK, accounts)
		} else if req.RequestType == "release" {
			ReleaseLocksOnAccounts(req.Accounts)
			return c.JSON(http.StatusOK, "Locks released")
		} else {
			return c.JSON(http.StatusBadRequest, "Invalid request type")
		}
	})
	e.Logger.Fatal(e.Start(":1323"))
}
