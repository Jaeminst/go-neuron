package main

import (
	"log"
	"time"

	"github.com/Jaeminst/go-neuron"
	"github.com/Jaeminst/go-neuron/examples/common"
)

func main() {
	var config common.AppConfig

	live, err := neuron.NewSync(&config)
	if err != nil {
		log.Fatal(err)
	}

	live.OnChange(func(cfg common.AppConfig) {
		log.Printf("[OnChange] Count=%d, Message=%s\n", cfg.Count, cfg.Message)
	})

	for {
		time.Sleep(10 * time.Second)
		log.Printf("[Loop] Count=%d, Message=%s\n", config.Count, config.Message)
	}
}
