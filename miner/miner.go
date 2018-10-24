package main

import (
	"../fdlib"
	. "../shared"
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"
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

type MinerConfiguration struct {
	MinedCoinsPerOpBlock int
	MinedCoinsPerNoOpBlock int
	NumCoinsPerFileCreate int
	GenOpBlockTimeout int
	GenesisBlockHash string
	PowPerOpBlock int
	PowPerNoOpBlock int
	ConfirmsPerFileCreate int
	ConfirmsPerFileAppend int
	MinerID string
	PeerMinersAddrs []string
	IncomingMinersAddr string
	OutgoingMinersIP string
	IncomingClientsAddr string
}

var lg = log.New(os.Stdout, "miner: ", log.Ltime)

func main() {
	argsWithoutProg := os.Args[1:]
	if len(argsWithoutProg) != 1 {
		lg.Println("usage: go run miner.go [settings]")
		os.Exit(1)
	}

	// TODO ksenia. Currently all the miner does is listen for clients and respond to requests
	conf, err := ParseConfig(argsWithoutProg[0])
	if err != nil {
		lg.Println(err)
		os.Exit(1)
	}
	var minerInstance Miner = MinerInstance{minerConf: conf}

	// Initialize failure detector and start responding
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	epochNonce := r1.Uint64()
	fd, _, err := fdlib.InitializeFDLib(uint64(epochNonce), 5)
	if err != nil {
		lg.Println(err)
		os.Exit(1)
	}
	fd.StartResponding(conf.IncomingClientsAddr)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	lg.Println("Listening for clients...")

	ci := ClientHandler{
		ListenHost: conf.IncomingClientsAddr,
		miner:      &minerInstance,
		waitGroup:  wg,
	}
	go ci.ListenForClients()
	wg.Wait()
}

type MinerInstance struct {
	minerConf MinerConfiguration
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

/////////// Helpers ///////////////////////////////////////////////////////

func ParseConfig(fileName string) (MinerConfiguration, error){
	var m MinerConfiguration

	file, err := os.Open(fileName)
	if err != nil {
		return MinerConfiguration{}, err
	}

	b, err := ioutil.ReadAll(file)
	if err != nil {
		return MinerConfiguration{}, err
	}

	err = json.Unmarshal(b, &m)
	if err != nil {
		return MinerConfiguration{}, err
	}
	return m, nil
}
