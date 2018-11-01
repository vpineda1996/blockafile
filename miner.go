package main

import (
	"./miner/instance"
	"./shared"
	"log"
	"os"
	"sync"
)

func main() {
	argsWithoutProg := os.Args[1:]
	if len(argsWithoutProg) != 1 {
		log.Println("usage: go run miner.go [settings]")
		os.Exit(1)
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	instance.NewMinerInstance(argsWithoutProg[0], wg, shared.DEFAULT_SINGLE_MINER_DISCONNECTED)
	log.Println("Listening for clients...")
	wg.Wait()
}
