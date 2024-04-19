package config

import (
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

var RESOLVER_SERVER_PORT int
var DB_PATH string
var Logger *log.Logger

func CreateLog(fileName, header string) *log.Logger {
	newpath := filepath.Join(".", "log")
	os.MkdirAll(newpath, os.ModePerm)
	serverLogFile, _ := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	return log.New(serverLogFile, header, log.Lmicroseconds|log.Lshortfile)
}

func LoadEnvData() error {
	// Load the .env file
	Logger = CreateLog("log/resolver.log", "[RESOLVER]")
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
		return err
	}
	// Get the environment variables
	RESOLVER_SERVER_PORT, _ = strconv.Atoi(os.Getenv("RESOLVER_SERVER_PORT"))
	DB_PATH = os.Getenv("DB_PATH")
	log.Printf("RESOLVER_SERVER_PORT: %v", RESOLVER_SERVER_PORT)
	return nil
}
