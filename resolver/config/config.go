package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

var RESOLVER_SERVER_PORT int

func LoadEnvData() error {
	// Load the .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
		return err
	}
	// Get the environment variables
	RESOLVER_SERVER_PORT, _ = strconv.Atoi(os.Getenv("RESOLVER_SERVER_PORT"))
	log.Printf("RESOLVER_SERVER_PORT: %v", RESOLVER_SERVER_PORT)
	return nil
}
