package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Jaeminst/go-neuron"
	"github.com/Jaeminst/go-neuron/examples/common"
)

func main() {
	config, err := common.NewConfig[common.AppConfig]("config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	live, err := neuron.NewSync(&config)
	if err != nil {
		log.Fatal(err)
	}

	live.AutoFlush(300 * time.Millisecond)

	for {
		now := time.Now().Format("15:04:05")
		config.Count++
		config.Message = now
		fmt.Println("[Writer] Updated shared struct at", now, "[", config.Count, "]")
		time.Sleep(2 * time.Second)
	}
}
