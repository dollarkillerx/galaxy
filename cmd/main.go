package main

import (
	"github.com/dollarkillerx/galaxy/internal/config"
	"github.com/dollarkillerx/galaxy/internal/scheduler"

	"log"
)

func main() {
	log.SetFlags(log.Llongfile | log.LstdFlags)

	err := config.InitConfig()
	if err != nil {
		log.Fatalln(err)
	}

	schedule := scheduler.NewSchedule()
	if err := schedule.Run(); err != nil {
		log.Fatalln(err)
	}
}
