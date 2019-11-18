package main

import (
	"log"
	"os"

	sy "github.com/frncscsrcc/syngonizer"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatal("Missing config file in parameters")
	}
	configFile := args[0]

	config, err := sy.LoadConfig(configFile)
	if err != nil {
		log.Fatal(err)
	}

	sy.Watch(config)

}
