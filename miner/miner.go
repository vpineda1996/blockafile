package main

import (
	. "../shared"
	"log"
	"os"
	"sync"
)

// Miner type declaration
type Miner interface {
	CreateFileHandler(fname string) (errorType FailureType)
	// ListFilesHandler() does not return any error because on the client-side, the only kind of error
	// that can be returned is a DisconnectedError, which means we would never reach the Miner in the first place.
	ListFilesHandler() (fnames []string)
	TotalRecsHandler(fname string) (numRecs uint16, errorType FailureType)
	ReadRecHandler(fname string, recordNum uint16) (record [512]byte, errorType FailureType)
	AppendRecHandler(fname string, record [512]byte) (recordNum uint16, errorType FailureType)
}

var lg = log.New(os.Stdout, "miner: ", log.Ltime)

func main() {
	// TODO. Currently all the miner does is listen for clients and respond to requests
	var minerInstance Miner = MinerInstance{}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	lg.Println("Listening for clients...")

	// TODO. This address should be a setting, not hardcoded.
	ci := ClientHandler{
		ListenHost: "127.0.0.1:9090",
		miner:      &minerInstance,
		waitGroup:  wg,
	}
	go ci.ListenForClients()
	wg.Wait()
}

type MinerInstance struct {
	// TODO. Fields
}

// errorType can be one of: FILE_EXISTS, BAD_FILENAME, NO_ERROR
func (miner MinerInstance) CreateFileHandler(fname string) (errorType FailureType) {
	// TODO
	lg.Println("Handling create file request")
	// return shared.FILE_EXISTS
	return -1
}

func (miner MinerInstance) ListFilesHandler() (fnames []string) {
	// TODO
	lg.Println("Handling list files request")
	fnames = []string{"sample_file1", "sample_file2", "sample_file3"}
	return
}

// errorType can be one of: FILE_DOES_NOT_EXIST, NO_ERROR
func (miner MinerInstance) TotalRecsHandler(fname string) (numRecs uint16, errorType FailureType) {
	// TODO
	lg.Println("Handling total records request")
	return 10, -1
}

// errorType can be one of: FILE_DOES_NOT_EXIST, NO_ERROR
func (miner MinerInstance) ReadRecHandler(fname string, recordNum uint16) (record [512]byte, errorType FailureType) {
	// TODO
	lg.Println("Handling read record request")
	var read_result [512]byte
	copy(read_result[:], "Some nice record stuff")
	return read_result, -1
}

// errorType can be one of: FILE_DOES_NOT_EXIST, MAX_LEN_REACHED, NO_ERROR
func (miner MinerInstance) AppendRecHandler(fname string, record [512]byte) (recordNum uint16, errorType FailureType) {
	// TODO
	lg.Println("Handling append record request")
	// return 0, shared.MAX_LEN_REACHED
	return 0, -1
}
