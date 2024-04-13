package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

var BANKSERVERPORT int

func LoadEnvData() error {
	// Load the .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
		return err
	}
	// Get the environment variables
	BANKSERVERPORT, _ = strconv.Atoi(os.Getenv("BANKSERVERPORT"))
	log.Printf("BANKSERVERPORT: %v", BANKSERVERPORT)
	return nil
}
