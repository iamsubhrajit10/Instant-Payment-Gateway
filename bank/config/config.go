package config

import (
	//"bank/config"

	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

var BANKSERVERPORT int
var LeaderIPV4 string
var LeaderPort int
var IsLeader string
var ServerID int

var Client *elasticsearch.Client
var DB *sql.DB
var msg string
var err error
var Logger *log.Logger

var IndexName string

type transactionLog struct {
	TransactionID string
	AccountNumber string
	Amount        int
	Type          string
}

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

func CreateElasticSearchClient() error {
	// Create a new Elasticsearch client and connect to http://
	//	Client, err = elasticsearch.NewDefaultClient()

	cfg := elasticsearch.Config{
		Addresses: []string{
			"http://localhost:9200",
		},
		Username: "elastic",
		Password: "4LhVyC8-UV+3_gC+o1PU",
	}
	Client, err = elasticsearch.NewClient(cfg)
	if err != nil {
		Logger.Fatalf("Error creating the client: %s", err)
		return err
	}
	_, err_ := Client.Ping()
	if err_ != nil {
		Logger.Fatalf("Error pinging the server: %s", err)
		return err_
	}
	Logger.Println("Successfully connected to Elasticsearch")
	return nil
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
	IndexName = os.Getenv("INDEXNAME")
	ServerID, _ = strconv.Atoi(generateRandomID())
	Logger.Printf("BANKSERVERPORT: %v", BANKSERVERPORT)
	msg, err := ConnectWithSql()
	Logger.Printf("IndexName: %v", IndexName)
	err = CreateElasticSearchClient()
	if err != nil {
		Logger.Fatalf("Error connecting to sql: %v", err)
	}
	Logger.Printf("SQL connection status: %v", msg)
	return nil
}
