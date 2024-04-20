package main

import (
	"fmt"
	"log"
	"time"

	"tpg/internals/router"
	"tpg/scheduler"
	"tpg/config"
	"github.com/go-co-op/gocron/v2"
)

func main() {
	s, err := gocron.NewScheduler()
	if err != nil {
		log.Fatal("error: unable to create new scheduler")
	}

	port := "80"

	j, err := s.NewJob(
		gocron.DurationJob(
			10*time.Second,
		),
		gocron.NewTask(
			scheduler.Reverse,
		),
	)

	if err != nil {
		log.Fatal("error: unable to populate scheduler with reverse job")
	}
	fmt.Println(j.ID())
	s.Start()

	//Start the echo server
	e := router.SetupRouter()
	err_env := config.LoadEnvData()
	if err_env != nil {
		log.Fatalf("Error loading .env file")
	}
	e.Logger.Fatal(e.Start(":" + port))
}
