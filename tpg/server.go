package main

import (
	"fmt"
	"log"
	"time"

	"tpg/internals/router"
	"tpg/scheduler"

	// "tpg/scheduler"

	"github.com/go-co-op/gocron/v2"
)

func main() {
	e := router.SetupRouter()

	s, err := gocron.NewScheduler()
	if err != nil {
		log.Fatal(err)
	}

	j, err := s.NewJob(
		gocron.DurationJob(
			10*time.Second,
		),
		gocron.NewTask(
			scheduler.Reverse,
		),
	)

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(j.ID())
	s.Start()
	e.Logger.Fatal(e.Start(":8080"))
}

// package main

// import (
// 	"fmt"
// 	"log"
// 	"time"
// 	"tpg/scheduler"

// 	"github.com/go-co-op/gocron/v2"
// )

// func main() {
// 	s, err := gocron.NewScheduler()
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	j, err := s.NewJob(
// 		gocron.DurationJob(
// 			10*time.Second,
// 		),
// 		gocron.NewTask(
// 			scheduler.Reverse,
// 		),
// 	)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	fmt.Println(j.ID())
// 	s.Start()
// 	err = s.Shutdown()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// }
