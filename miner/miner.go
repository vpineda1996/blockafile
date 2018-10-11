package main

import (
	"../shared"
	"bytes"
	"encoding/gob"
	"errors"
	"log"
	"net"
	"os"
	"sync"
)

// Miner type declaration
type Miner interface {
	CreateFileHandler(fname string) (err error)
	// ListFilesHandler() does not return any error because on the client-side, the only kind of error
	// that can be returned is a DisconnectedError, which means we would never reach the Miner in the first place.
	ListFilesHandler() (fnames []string)
	TotalRecsHandler(fname string) (numRecs uint16, err error)
	ReadRecHandler(fname string, recordNum uint16, record []byte) (err error)
	AppendRecHandler(fname string, record []byte) (recordNum uint16, err error)
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

func (miner *MinerInstance) CreateFileHandler(fname string) (err error) {
	// TODO
	return errors.New("implement CreateFileHandler")
}

func (miner *MinerInstance) ListFilesHandler() (fnames []string) {
	// TODO
	lg.Println("Handling list files request")
	return make([]string, 0)
}

func (miner *MinerInstance) TotalRecsHandler(fname string) (numRecs uint16, err error) {
	// TODO
	return 0, errors.New("implement TotalRecsHandler")
}

func (miner *MinerInstance) ReadRecHandler(fname string, recordNum uint16, record []byte) (err error) {
	// TODO
	return errors.New("implement ReadRecHandler")
}

func (miner *MinerInstance) AppendRecHandler(fname string, record []byte) (recordNum uint16, err error) {
	// TODO
	return 0, errors.New("implement AppendRecHandler")
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
			//lg.Println(err) // TODO. Prints infinite amount of EOFs
			continue
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
		minerResponse := shared.RFSMinerResponse{Err: nil}

		switch clientRequest.RequestType {
		case shared.CREATE_FILE:
			createFileError := minerInstance.CreateFileHandler(clientRequest.FileName)
			minerResponse.Err = createFileError
		case shared.LIST_FILES:
			fnames := minerInstance.ListFilesHandler()
			minerResponse.FileNames = fnames
		case shared.TOTAL_RECS:
			numRecs, totalRecsError := minerInstance.TotalRecsHandler(clientRequest.FileName)
			minerResponse.NumRecords = numRecs
			minerResponse.Err = totalRecsError
		case shared.READ_REC:
			readRecError := minerInstance.ReadRecHandler(
				clientRequest.FileName, clientRequest.RecordNum, clientRequest.Record)
			minerResponse.Err = readRecError
		case shared.APPEND_REC:
			recordNum, appendRecError := minerInstance.AppendRecHandler(clientRequest.FileName, clientRequest.Record)
			minerResponse.RecordNum = recordNum
			minerResponse.Err = appendRecError
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
