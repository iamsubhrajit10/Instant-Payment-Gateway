package scheduler

import (
	"encoding/csv"
	"os"
	"strconv"
	"tpg/config"
	ph "tpg/internals/paymenthandler"
)

func reverseTransaction(transaction []string) {
	// Open the file.
	if transaction[3] == "debit" {
		amount, err := strconv.Atoi(transaction[4])
		if err != nil {
			config.Logger.Fatal(err)
		}
		data := ph.RequestDataBank{TransactionID: transaction[0], AccountNumber: transaction[5], Amount: amount, Type: "reverse"}
		msg, err := ph.ReverseDebit(transaction[1], data)

		if msg != "Transaction Reversed" || err != nil {

			file, err := os.OpenFile("Failed_Transaction.csv", os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				config.Logger.Fatal(err)
			}
			defer file.Close()

			// Write the transaction to the file.
			writer := csv.NewWriter(file)
			defer writer.Flush()
			if err := writer.Write(transaction); err != nil {
				config.Logger.Fatal(err)
			}
		}
	}
}

func Reverse() interface{} {

	file, err := os.Open("Failed_Transaction.csv")
	if err != nil {
		config.Logger.Fatal(err)
	}
	defer file.Close()
	// Create a CSV reader.
	reader := csv.NewReader(file)
	// Read all data from the CSV file.
	data, err := reader.ReadAll()
	if err != nil {
		config.Logger.Fatal(err)
	}
	if err := os.Truncate("Failed_Transaction.csv", 0); err != nil {
		config.Logger.Printf("Failed to truncate: %v", err)
	}
	// Print the data.
	for _, row := range data {
		config.Logger.Printf("Transaction ID: %v, Account Number: %v, Amount: %v, Type: %v\n", row[0], row[5], row[2], row[3])
		// Reverse the transaction.
		reverseTransaction(row)
	}

	// Clear the file.

	//defer file.Close()

	// Get the current time
	//currentTime := time.Now()

	// Format the time as a string
	//	timeString := currentTime.Format("2006-01-02 15:04:05")

	// Write the formatted time to the file
	// if _, err := file.WriteString(timeString + "\n"); err != nil {
	// 	fmt.Println("Error writing to file:", err)
	// 	return nil
	// }

	// fmt.Println("Current time appended to file successfully.")
	return nil
}
