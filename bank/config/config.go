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
var MySQLIPV41 string
var MySQLIPV42 string
var MySQLIPV43 string
var ServerID int
var LEADERLISTENIPV4 string
var ElasticSearchIPV4 string
var DBPORT string
var Client *elasticsearch.Client
var DB *sql.DB
var DB2 *sql.DB
var DB3 *sql.DB
var msg string
var err error
var err1 error
var err2 error
var err3 error

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

	log.Print("Connecting to MySQL database...")
	DB, err1 = sql.Open("mysql", fmt.Sprintf("root:root@tcp(%s:%s)/upi", MySQLIPV41, DBPORT))
	DB2, err2 = sql.Open("mysql", fmt.Sprintf("root:root@tcp(%s:%s)/upi", MySQLIPV42, DBPORT))
	DB3, err3 = sql.Open("mysql", fmt.Sprintf("root:root@tcp(%s:%s)/upi", MySQLIPV43, DBPORT))
	if err1 != nil || err2 != nil || err3 != nil {
		Logger.Fatal(err)
		return "", err
	}

	log.Print("After connecting to MySQL database...")

	err1, err2, err3 = DB.Ping(), DB2.Ping(), DB3.Ping()
	if err1 != nil || err2 != nil || err3 != nil {
		log.Fatal(err1)
		log.Fatal(err2)
		log.Fatal(err3)
		return "", err
	}
	log.Println("Successfully connected to MySQL database")
	return "Success", nil
}

func CreateElasticSearchClient() error {
	// Create a new Elasticsearch client and connect to http://
	//	Client, err = elasticsearch.NewDefaultClient()
	cfg := elasticsearch.Config{
		Addresses: []string{
			// "http://10.240.1.252:9200",
			fmt.Sprintf("http://%s:9200", ElasticSearchIPV4),
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
	MySQLIPV41 = os.Getenv("MYSQLIPV41")
	MySQLIPV42 = os.Getenv("MYSQLIPV42")
	MySQLIPV43 = os.Getenv("MYSQLIPV43")

	IndexName = os.Getenv("INDEXNAME")
	LEADERLISTENIPV4 = os.Getenv("LEADERLISTENIPV4")
	ElasticSearchIPV4 = os.Getenv("ELASTICSEARCHIPV4")
	DBPORT = os.Getenv("DBPORT")

	ServerID, _ = strconv.Atoi(generateRandomID())
	Logger.Printf("BANKSERVERPORT: %v", BANKSERVERPORT)

	log.Printf("BANKSERVERPORT: %v", BANKSERVERPORT)
	time.Sleep(10 * time.Second)
	msg, err = ConnectWithSql()
	// for i := 0; i < 3; i++ {

	// 	if err == nil {
	// 		break
	// 	}
	// 	time.Sleep(5 * time.Second)
	// }

	if err != nil {
		Logger.Fatalf("Error connecting to sql: %v", err)
		return err
	}

	//msg, _ := ConnectWithSql()
	Logger.Printf("IndexName: %v", IndexName)
	err = CreateElasticSearchClient()
	if err != nil {
		Logger.Fatalf("Error connecting to sql: %v", err)
	}
	Logger.Printf("SQL connection status: %v", msg)
	return nil
}
