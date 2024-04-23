package main

import (
	//"fmt"

	"fmt"
	"time"

	//"os"

	"log"
	"tpg/config"
	"tpg/internals/router"
	"tpg/scheduler"

	//"tpg/scheduler"
	"github.com/go-co-op/gocron/v2"
)

func main() {
	s, err := gocron.NewScheduler()
	if err != nil {
		log.Fatalf("error: unable to create new scheduler")
	}

	//port := "8081"
	// if len(os.Args) > 1 {
	// 	port = os.Args[1]
	// }
	j, err := s.NewJob(
		gocron.DurationJob(
			10*time.Second,
		),
		gocron.NewTask(
			scheduler.Reverse,
		),
	)

	if err != nil {
		log.Fatalf("error: unable to populate scheduler with reverse job")
	}
	//log.Println(j.ID())
	fmt.Println(j.ID())

	s.Start()

	//Start the echo server
	e := router.SetupRouter()
	err_env := config.LoadEnvData()
	if err_env != nil {
		log.Fatalf("Error loading .env file")
	}
	e.Logger.Fatal(e.Start(":8081"))
}
