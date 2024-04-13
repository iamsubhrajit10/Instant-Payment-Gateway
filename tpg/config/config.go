package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

var DebitBankServerIPV4 string
var DebitBankServerPort string
var CreditBankServerIPV4 string
var CreditBankServerPort string
var DebitRetries int

func LoadEnvData() error {
	// Load the .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
		return err
	}

	// Get the environment variables
	DebitBankServerIPV4 = os.Getenv("DEBITBANKSERVERIPV4")
	DebitBankServerPort = os.Getenv("DEBITPORT")
	CreditBankServerIPV4 = os.Getenv("CREDITBANKSERVERIPV4")
	CreditBankServerPort = os.Getenv("CREDITPORT")
	DebitRetries, _ = strconv.Atoi(os.Getenv("DEBET_RETRIES"))
	return nil
}
