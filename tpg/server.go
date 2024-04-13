package main

import (
	"log"
	"tpg/config"
	"tpg/internals/router"
)

func main() {
	e := router.SetupRouter()

	err := config.LoadEnvData()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	// Start server
	e.Logger.Fatal(e.Start(":8081"))
}
