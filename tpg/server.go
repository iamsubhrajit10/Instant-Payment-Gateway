package main

import (
	"tpg/internals/router"
)

func main() {
	e := router.SetupRouter()
	// Start server
	e.Logger.Fatal(e.Start(":8080"))
}
