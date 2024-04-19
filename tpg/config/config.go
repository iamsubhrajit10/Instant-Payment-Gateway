package config

import (
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

var DebitBankServerIPV4 string
var DebitBankServerPort string
var CreditBankServerIPV4 string
var CreditBankServerPort string
var DebitRetries int
var Logger *log.Logger

func CreateLog(fileName, header string) *log.Logger {
	newpath := filepath.Join(".", "log")
	os.MkdirAll(newpath, os.ModePerm)
	serverLogFile, _ := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	return log.New(serverLogFile, header, log.Lmicroseconds|log.Lshortfile)
}

var ResolverServerPort string
var ResolverServerIPV4 string

func LoadEnvData() error {
	// Load the .env file
	Logger = CreateLog("log/tpg.log", "[TPG]")
	err := godotenv.Load()
	if err != nil {
		Logger.Fatalf("Error loading .env file")
		return err
	}

	// Get the environment variables
	ResolverServerPort = os.Getenv("RESOLVER_SERVER_PORT")
	ResolverServerIPV4 = os.Getenv("RESOLVER_SERVER_IPV4")
	DebitBankServerIPV4 = os.Getenv("DEBITBANKSERVERIPV4")
	DebitBankServerPort = os.Getenv("DEBITPORT")
	CreditBankServerIPV4 = os.Getenv("CREDITBANKSERVERIPV4")
	CreditBankServerPort = os.Getenv("CREDITPORT")
	DebitRetries, _ = strconv.Atoi(os.Getenv("DEBET_RETRIES"))

	return nil
}
