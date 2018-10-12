package main

import (
	"../rfslib"
	"log"
	"os"
)

const (
	SAMPLE_FNAME = "sample_file1" //TODO
)

var lg = log.New(os.Stdout, "client_basic: ", log.Ltime)

// usage: go run client_basic.go [localip:localport] [minerip:minerport]
// e.g. : go run client_basic.go 127.0.0.1:8080		 127.0.0.1:9090
func main() {
	argsWithoutProg := os.Args[1:]
	if len(argsWithoutProg) != 2 {
		lg.Println("usage: go run client_basic.go [localip:localport] [minerip:minerport]")
		os.Exit(1)
	}

	// Initialize rfslib instance for this client
	localAddr := argsWithoutProg[0]
	minerAddr := argsWithoutProg[1]
	rfs, err := rfslib.Initialize(localAddr, minerAddr)
	if err != nil {
		lg.Println(err)
		os.Exit(1)
	}

	// Make some requests to the miner
	// List files
	fnames, err := rfs.ListFiles()
	if err != nil {
		lg.Println(err)
		os.Exit(1)
	} else {
		for i, fname := range fnames {
			lg.Printf("File %v: %s\n", i, fname)
		}
	}

	// Total records count
	numRecs, err := rfs.TotalRecs(SAMPLE_FNAME)
	if err != nil {
		lg.Println(err)
		os.Exit(1)
	} else {
		lg.Printf("Total number of records in file %s: %v\n", SAMPLE_FNAME, numRecs)
	}

	// Read record
	record := new(rfslib.Record)
	index := uint16(0)
	err = rfs.ReadRec(SAMPLE_FNAME, index, record)
	if err != nil {
		lg.Println(err)
		os.Exit(1)
	} else {
		lg.Printf("Read file %s at index %v: %v\n", SAMPLE_FNAME, index, string(record[:]))
	}

	err = rfslib.TearDown()
	if err != nil {
		lg.Println(err)
		os.Exit(1)
	}
}
