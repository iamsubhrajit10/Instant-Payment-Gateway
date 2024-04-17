package config

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
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

func generateRandomID() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%04d", rand.Intn(10000))
}

func connectWithSql() (*sql.DB, string, error) {
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/upi")
	if err != nil {
		log.Fatal(err)
		return nil, "", err
	}
	//defer DB.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
		return nil, "", err
	}

	log.Println("Successfully connected to MySQL database")
	return db, "Success", nil
}

func LoadEnvData() error {
	// Load the .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
		return err
	}

	// Get the environment variables
	BANKSERVERPORT, _ = strconv.Atoi(os.Getenv("BANKSERVERPORT"))
	LeaderIPV4 = os.Getenv("LEADERIPV4")
	LeaderPort, _ = strconv.Atoi(os.Getenv("LEADERPORT"))
	IsLeader = os.Getenv("ISLEADER")
	ServerID, _ = strconv.Atoi(generateRandomID())
	log.Printf("BANKSERVERPORT: %v", BANKSERVERPORT)

	DB, msg, err = connectWithSql()
	if err != nil {
		log.Fatalf("Error connecting to sql: %v", err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatal(err)
		//return nil, "", err
	}
	// results, err := DB.Query("SELECT Amount FROM bank_details WHERE Account_number = ?", "1234")
	// if err != nil {
	// 	//log.Fatal(err)
	// 	//return "", err
	// }

	// for results.Next() {
	// 	var amount int
	// 	// for each row, scan the result into our tag composite object
	// 	err = results.Scan(&amount)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 		// proper error handling instead of panic in your app
	// 		//	return "", err
	// 	}
	// 	log.Printf("Processing debit request3: %v", amount)

	// }

	log.Printf("SQL connection status: %v", msg)
	return nil
}
