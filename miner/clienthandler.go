package main

import (
	"../shared"
	"bytes"
	"encoding/gob"
	"io"
	"net"
	"sync"
)

type ClientHandler struct {
	ListenHost string

	waitGroup *sync.WaitGroup
	miner     *Miner
}

// The miner is always listening for client connections.
func (c ClientHandler) ListenForClients() error {
	defer c.waitGroup.Done()
	addr, err := net.ResolveTCPAddr("tcp", c.ListenHost)
	if err != nil {
		lg.Printf("Error resolving TCP address: %v\n", err)
		return err
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		lg.Printf("Error listening for clients: %v\n", err)
		return err
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			lg.Printf("Error accepting: %v\n", err)
			continue
		}

		go c.ServiceClientRequest(conn)
	}
}

// Once a client connection has been accepted, the miner is always servicing requests from the client.
func (c ClientHandler) ServiceClientRequest(conn net.Conn) error {
	minerInstance := c.miner
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
				return err
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
		minerResponse := shared.RFSMinerResponse{ErrorType: shared.NO_ERROR}

		switch clientRequest.RequestType {
		case shared.CREATE_FILE:
			createFileError := (*minerInstance).CreateFileHandler(clientRequest.FileName)
			minerResponse.ErrorType = createFileError
		case shared.LIST_FILES:
			fnames := (*minerInstance).ListFilesHandler()
			minerResponse.FileNames = fnames
		case shared.TOTAL_RECS:
			numRecs, totalRecsError := (*minerInstance).TotalRecsHandler(clientRequest.FileName)
			minerResponse.NumRecords = numRecs
			minerResponse.ErrorType = totalRecsError
		case shared.READ_REC:
			readRec, readRecError := (*minerInstance).ReadRecHandler(
				clientRequest.FileName, clientRequest.RecordNum)
			minerResponse.ReadRecord = readRec
			minerResponse.ErrorType = readRecError
		case shared.APPEND_REC:
			recordNum, appendRecError :=
				(*minerInstance).AppendRecHandler(clientRequest.FileName, clientRequest.AppendRecord)
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
			return err
		}
		conn.Write(responseBuf.Bytes())
	}
}
