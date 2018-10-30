package main

import (
	"log"
	"os"
	"sync"
	"./miner/instance"
)

func main() {
	argsWithoutProg := os.Args[1:]
	if len(argsWithoutProg) != 1 {
		log.Println("usage: go run miner.go [settings]")
		os.Exit(1)
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	instance.NewMinerInstance(argsWithoutProg[0], wg)
	log.Println("Listening for clients...")
	wg.Wait()
}
