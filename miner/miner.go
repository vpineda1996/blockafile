package main

import (
	"../shared"
	"bytes"
	"encoding/gob"
	"io"
	"log"
	"net"
	"os"
	"sync"
)

// Miner type declaration
type Miner interface {
	CreateFileHandler(fname string) (errorType int)
	// ListFilesHandler() does not return any error because on the client-side, the only kind of error
	// that can be returned is a DisconnectedError, which means we would never reach the Miner in the first place.
	ListFilesHandler() (fnames []string)
	TotalRecsHandler(fname string) (numRecs uint16, errorType int)
	ReadRecHandler(fname string, recordNum uint16) (record [512]byte, errorType int)
	AppendRecHandler(fname string, record [512]byte) (recordNum uint16, errorType int)
}

var lg = log.New(os.Stdout, "miner: ", log.Ltime)
var wg = &sync.WaitGroup{}

func main() {
	// TODO. Currently all the miner does is listen for clients and respond to requests
	minerInstance = new(MinerInstance)
	wg.Add(1)
	lg.Println("Listening for clients...")
	go ListenForClients()
	wg.Wait()
}

// Concrete implementation of the miner interface
var minerInstance *MinerInstance = nil

type MinerInstance struct {
	// TODO. Fields
}

// errorType can be one of: FILE_EXISTS, BAD_FILENAME, -1
func (miner MinerInstance) CreateFileHandler(fname string) (errorType int) {
	// TODO
	lg.Println("Handling create file request")
	// return shared.FILE_EXISTS
	return -1
}

// errorType can be one of: -1
func (miner MinerInstance) ListFilesHandler() (fnames []string) {
	// TODO
	lg.Println("Handling list files request")
	fnames = []string{"sample_file1", "sample_file2", "sample_file3"}
	return
}

// errorType can be one of: FILE_DOES_NOT_EXIST, -1
func (miner MinerInstance) TotalRecsHandler(fname string) (numRecs uint16, errorType int) {
	// TODO
	lg.Println("Handling total records request")
	return 10, -1
}

// errorType can be one of: FILE_DOES_NOT_EXIST, RECORD_DOES_NOT_EXIST, -1
func (miner MinerInstance) ReadRecHandler(fname string, recordNum uint16) (record [512]byte, errorType int) {
	// TODO
	lg.Println("Handling read record request")
	var read_result [512]byte
	copy(read_result[:], "Some nice record stuff")
	return read_result, -1
}

// errorType can be one of: FILE_DOES_NOT_EXIST, MAX_LEN_REACHED, -1
func (miner MinerInstance) AppendRecHandler(fname string, record [512]byte) (recordNum uint16, errorType int) {
	// TODO
	lg.Println("Handling append record request")
	// return 0, shared.MAX_LEN_REACHED
	return 0, -1
}

// The miner is always listening for client connections.
func ListenForClients() {
	defer wg.Done()
	// TODO. This address should be a setting, not hardcoded.
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:9090")
	if err != nil {
		lg.Printf("Error resolving TCP address: %v\n", err)
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		lg.Printf("Error listening for clients: %v\n", err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			lg.Printf("Error accepting: %v\n", err)
		}

		go ServiceClientRequest(conn)
	}
}

// Once a client connection has been accepted, the miner is always servicing requests from the client.
func ServiceClientRequest(conn net.Conn) {
	for {
		// Make a buffer to hold incoming data.
		requestBuf := make([]byte, 1024)

		// Read the incoming connection into the buffer.
		readLen, err := conn.Read(requestBuf)
		if err != nil {
			if err == io.EOF {
				// Client connection has been closed.
				lg.Println("Closing client connection")
				conn.Close()
				return
			} else {
				// Some other sort of error. Continue trying to read.
				lg.Println(err)
				continue
			}
		}

		// Decode the client request.
		clientRequest := shared.RFSClientRequest{}
		var reader = bytes.NewReader(requestBuf[:readLen])
		dec := gob.NewDecoder(reader)
		err = dec.Decode(&clientRequest)
		if err != nil {
			lg.Println(err)
			continue
		}

		// Direct the request to the proper handler and create response
		var responseBuf bytes.Buffer
		enc := gob.NewEncoder(&responseBuf)
		minerResponse := shared.RFSMinerResponse{ErrorType: -1}

		switch clientRequest.RequestType {
		case shared.CREATE_FILE:
			createFileError := minerInstance.CreateFileHandler(clientRequest.FileName)
			minerResponse.ErrorType = createFileError
		case shared.LIST_FILES:
			fnames := minerInstance.ListFilesHandler()
			minerResponse.FileNames = fnames
		case shared.TOTAL_RECS:
			numRecs, totalRecsError := minerInstance.TotalRecsHandler(clientRequest.FileName)
			minerResponse.NumRecords = numRecs
			minerResponse.ErrorType = totalRecsError
		case shared.READ_REC:
			readRec, readRecError := minerInstance.ReadRecHandler(
				clientRequest.FileName, clientRequest.RecordNum)
			minerResponse.ReadRecord = readRec
			minerResponse.ErrorType = readRecError
		case shared.APPEND_REC:
			// TODO
			recordNum, appendRecError :=
				minerInstance.AppendRecHandler(clientRequest.FileName, clientRequest.AppendRecord)
			minerResponse.RecordNum = recordNum
			minerResponse.ErrorType = appendRecError
		default:
			// Invalid request type, ignore it
			continue
		}

		// Send a response back to the client.
		err = enc.Encode(minerResponse)
		if err != nil {
			lg.Println(err)
			return
		}
		conn.Write(responseBuf.Bytes())
	}
}
