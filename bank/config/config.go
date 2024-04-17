package config

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

var BANKSERVERPORT int
var LeaderIPV4 string
var LeaderPort int
var IsLeader string
var ServerID int

var DB *sql.DB
var msg string
var err error
var Logger *log.Logger

func generateRandomID() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%04d", rand.Intn(10000))
}

func CreateLog(fileName, header string) *log.Logger {
	newpath := filepath.Join(".", "log")
	os.MkdirAll(newpath, os.ModePerm)
	serverLogFile, _ := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	return log.New(serverLogFile, header, log.Lmicroseconds|log.Lshortfile)
}

func ConnectWithSql() (string, error) {
	DB, err = sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/upi")
	if err != nil {
		Logger.Fatal(err)
		return "", err
	}

	err = DB.Ping()
	if err != nil {
		Logger.Fatal(err)
		return "", err
	}
	Logger.Println("Successfully connected to MySQL database")
	return "Success", nil
}

func LoadEnvData() error {

	// Load the .env file
	Logger = CreateLog("log/bank.log", "[Bank]")
	err := godotenv.Load()
	if err != nil {
		Logger.Fatalf("Error loading .env file")
		return err
	}

	// Get the environment variables
	BANKSERVERPORT, _ = strconv.Atoi(os.Getenv("BANKSERVERPORT"))
	LeaderIPV4 = os.Getenv("LEADERIPV4")
	LeaderPort, _ = strconv.Atoi(os.Getenv("LEADERPORT"))
	IsLeader = os.Getenv("ISLEADER")
	ServerID, _ = strconv.Atoi(generateRandomID())
	Logger.Printf("BANKSERVERPORT: %v", BANKSERVERPORT)
	msg, err := ConnectWithSql()
	if err != nil {
		Logger.Fatalf("Error connecting to sql: %v", err)
	}
	Logger.Printf("SQL connection status: %v", msg)
	return nil
}
