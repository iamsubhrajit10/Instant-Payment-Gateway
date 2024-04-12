package scheduler

import (
	"fmt"
	"os"
	"time"
)

func Reverse() interface{} {
	file, err := os.OpenFile("output.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return nil
	}
	defer file.Close()

	// Get the current time
	currentTime := time.Now()

	// Format the time as a string
	timeString := currentTime.Format("2006-01-02 15:04:05")

	// Write the formatted time to the file
	if _, err := file.WriteString(timeString + "\n"); err != nil {
		fmt.Println("Error writing to file:", err)
		return nil
	}

	fmt.Println("Current time appended to file successfully.")
	return nil
}
