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

	if _, err := neuron.NewSync(&config); err != nil {
		log.Fatal(err)
	}

	for {
		now := time.Now().Format("15:04:05")
		config.Count++
		config.Message = now
		config.XYZ.X++
		config.XYZ.Y++
		config.XYZ.Z++
		fmt.Println("[Writer] Updated shared struct at", now, "[", config.Count, "] [", config.XYZ.X, config.XYZ.Y, config.XYZ.Z, "]")
		time.Sleep(2 * time.Second)
	}
}
