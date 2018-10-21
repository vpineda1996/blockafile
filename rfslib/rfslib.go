/*

This package specifies the application's interface to the distributed
records system (RFS) to be used in project 1 of UBC CS 416 2018W1.

You are not allowed to change this API, but you do have to implement
it.

*/

package rfslib

import (
	"../shared"
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"os"
)

// A Record is the unit of file access (reading/appending) in RFS.
type Record [512]byte

////////////////////////////////////////////////////////////////////////////////////////////
// <ERROR DEFINITIONS>

// These type definitions allow the application to explicitly check
// for the kind of error that occurred. Each API call below lists the
// errors that it is allowed to raise.
//
// Also see:
// https://blog.golang.org/error-handling-and-go
// https://blog.golang.org/errors-are-values

// Contains minerAddr
type DisconnectedError string

func (e DisconnectedError) Error() string {
	return fmt.Sprintf("RFS: Disconnected from the miner [%s]", string(e))
}

// Contains filename. The *only* constraint on filenames in RFS is
// that must be at most 64 bytes long.
type BadFilenameError string

func (e BadFilenameError) Error() string {
	return fmt.Sprintf("RFS: Filename [%s] has the wrong length", string(e))
}

// Contains filename
type FileDoesNotExistError string

func (e FileDoesNotExistError) Error() string {
	return fmt.Sprintf("RFS: Cannot open file [%s] in D mode as it does not exist locally", string(e))
}

// Contains filename
type FileExistsError string

func (e FileExistsError) Error() string {
	return fmt.Sprintf("RFS: Cannot create file with filename [%s] as it already exists", string(e))
}

// Contains filename
type FileMaxLenReachedError string

func (e FileMaxLenReachedError) Error() string {
	return fmt.Sprintf("RFS: File [%s] has reached its maximum length", string(e))
}

// </ERROR DEFINITIONS>
////////////////////////////////////////////////////////////////////////////////////////////

// Represents a connection to the RFS system.
type RFS interface {
	// Creates a new empty RFS file with name fname.
	//
	// Can return the following errors:
	// - DisconnectedError
	// - FileExistsError
	// - BadFilenameError
	CreateFile(fname string) (err error)

	// Returns a slice of strings containing filenames of all the
	// existing files in RFS.
	//
	// Can return the following errors:
	// - DisconnectedError
	ListFiles() (fnames []string, err error)

	// Returns the total number of records in a file with filename
	// fname.
	//
	// Can return the following errors:
	// - DisconnectedError
	// - FileDoesNotExistError
	TotalRecs(fname string) (numRecs uint16, err error)

	// Reads a record from file fname at position recordNum into
	// memory pointed to by record. Returns a non-nil error if the
	// read was unsuccessful.
	//
	// Can return the following errors:
	// - DisconnectedError
	// - FileDoesNotExistError
	// - RecordDoesNotExistError (indicates record at this position has not been appended yet)
	ReadRec(fname string, recordNum uint16, record *Record) (err error)

	// Appends a new record to a file with name fname with the
	// contents pointed to by record. Returns the position of the
	// record that was just appended as recordNum. Returns a non-nil
	// error if the operation was unsuccessful.
	//
	// Can return the following errors:
	// - DisconnectedError
	// - FileDoesNotExistError
	// - FileMaxLenReachedError
	AppendRec(fname string, record *Record) (recordNum uint16, err error)
}

// Logger
var lg = log.New(os.Stdout, "rfslib: ", log.Ltime)

// The constructor for a new RFS object instance. Takes the miner's
// IP:port address string as parameter, and the localAddr which is the
// local IP:port to use to establish the connection to the miner.
//
// The returned rfs instance is singleton: an application is expected
// to interact with just one rfs at a time.
//
// This call should only succeed if the connection to the miner
// succeeds. This call can return the following errors:
// - Networking errors related to localAddr or minerAddr
func Initialize(localAddr string, minerAddr string) (rfs RFS, err error) {
	// Check if rfs has already been initialized
	if rfsInstance != nil {
		lg.Println("RFS has already been initialized")
		return *rfsInstance, nil
	}

	// Resolve TCP addresses
	laddr, err := net.ResolveTCPAddr("tcp", localAddr)
	if err != nil {
		return nil, err
	}

	maddr, err := net.ResolveTCPAddr("tcp", minerAddr)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTCP("tcp", laddr, maddr)
	if err != nil {
		return nil, err
	}

	rfsInstance = new(RFSInstance)
	rfsInstance.tcpConn = conn
	rfsInstance.minerAddr = minerAddr
	return *rfsInstance, nil
}

// For testing purposes
func TearDown() (err error) {
	err = rfsInstance.tcpConn.Close()
	rfsInstance = nil
	return
}

// Concrete implementation of RFS interface
var rfsInstance *RFSInstance = nil

type RFSInstance struct {
	tcpConn   *net.TCPConn
	minerAddr string
}

////////////////////////////////////////////////////////////////////////////////////////////
// RFS API Implementation

func (rfs RFSInstance) CreateFile(fname string) (err error) {
	// Encode and send the client request
	clientRequest := shared.RFSClientRequest{RequestType: shared.CREATE_FILE, FileName: fname}
	err = rfs.sendClientRequest(clientRequest)
	if err != nil {
		return err
	}

	// Wait for response from miner
	minerResponse, err := rfs.getMinerResponse()
	if err != nil {
		return err
	}

	// Generate the proper error to return to the client
	responseErr := rfs.generateResponseError(clientRequest, minerResponse)

	lg.Printf("Miner responded to create file request: %v\n", minerResponse)
	return responseErr
}

func (rfs RFSInstance) ListFiles() (fnames []string, err error) {
	// Encode and send the client request
	clientRequest := shared.RFSClientRequest{RequestType: shared.LIST_FILES}
	err = rfs.sendClientRequest(clientRequest)
	if err != nil {
		return nil, err
	}

	// Wait for response from miner
	minerResponse, err := rfs.getMinerResponse()
	if err != nil {
		return nil, err
	}

	// Generate the proper error to return to the client
	responseErr := rfs.generateResponseError(clientRequest, minerResponse)

	lg.Printf("Miner responded to list files request: %v\n", minerResponse)
	return minerResponse.FileNames, responseErr
}

func (rfs RFSInstance) TotalRecs(fname string) (numRecs uint16, err error) {
	// Encode and send the client request
	clientRequest := shared.RFSClientRequest{RequestType: shared.TOTAL_RECS, FileName: fname}
	err = rfs.sendClientRequest(clientRequest)
	if err != nil {
		return 0, err
	}

	// Wait for response from miner
	minerResponse, err := rfs.getMinerResponse()
	if err != nil {
		return 0, err
	}

	// Generate the proper error to return to the client
	responseErr := rfs.generateResponseError(clientRequest, minerResponse)

	lg.Printf("Miner responded to total recs request: %v\n", minerResponse)
	return minerResponse.NumRecords, responseErr
}

func (rfs RFSInstance) ReadRec(fname string, recordNum uint16, record *Record) (err error) {
	// Encode and send the client request
	clientRequest := shared.RFSClientRequest{RequestType: shared.READ_REC, FileName: fname, RecordNum: recordNum}
	err = rfs.sendClientRequest(clientRequest)
	if err != nil {
		return err
	}

	// Wait for response from miner
	minerResponse, err := rfs.getMinerResponse()
	if err != nil {
		return err
	}

	// Generate the proper error to return to the client
	responseErr := rfs.generateResponseError(clientRequest, minerResponse)

	// Copy the returned bytes into record
	copy(record[:], minerResponse.ReadRecord[:])

	lg.Printf("Miner responded to read rec request: %v\n", minerResponse)
	return responseErr
}

func (rfs RFSInstance) AppendRec(fname string, record *Record) (recordNum uint16, err error) {
	// Encode and send the client request
	clientRequest := shared.RFSClientRequest{RequestType: shared.APPEND_REC, FileName: fname, AppendRecord: *record}
	err = rfs.sendClientRequest(clientRequest)
	if err != nil {
		return 0, err
	}

	// Wait for response from miner
	minerResponse, err := rfs.getMinerResponse()
	if err != nil {
		return 0, err
	}

	// Generate the proper error to return to the client
	responseErr := rfs.generateResponseError(clientRequest, minerResponse)

	lg.Printf("Miner responded to append record request: %v\n", minerResponse)
	return minerResponse.RecordNum, responseErr
}

////////////////////////////////////////////////////////////////////////////////////////////
// RFSInstance helper functions

func (rfs RFSInstance) sendClientRequest(clientRequest shared.RFSClientRequest) error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(clientRequest)
	if err != nil {
		// This may be a little harsh, but we should never hit encoding errors
		panic(err)
	}

	// Send to miner
	lg.Println("Sending client request to miner")
	_, err = rfs.tcpConn.Write(buf.Bytes())
	if err != nil {
		// TODO. Should we be returning DisconnectedError here?
		lg.Println(err)
		return DisconnectedError(rfs.minerAddr)
	}

	return nil
}

func (rfs RFSInstance) getMinerResponse() (shared.RFSMinerResponse, error) {
	minerResponse := shared.RFSMinerResponse{}

	// Make a buffer to hold incoming response
	responseBuf := make([]byte, 1024)

	// Read the incoming connection into the buffer
	readLen, err := rfs.tcpConn.Read(responseBuf)
	if err != nil {
		// TODO. Should we be returning DisconnectedError here?
		lg.Println(err)
		return minerResponse, DisconnectedError(rfs.minerAddr)
	}

	// Decode the miner response
	var reader = bytes.NewReader(responseBuf[:readLen])
	dec := gob.NewDecoder(reader)
	err = dec.Decode(&minerResponse)
	if err != nil {
		// Again, we should never hit decoding errors
		panic(err)
	}

	return minerResponse, nil
}

func (rfs RFSInstance) generateResponseError(
	clientRequest shared.RFSClientRequest,
	minerResponse shared.RFSMinerResponse) (err error) {
	err = nil
	if minerResponse.ErrorType != shared.NO_ERROR {
		switch minerResponse.ErrorType {
		case shared.BAD_FILENAME:
			err = BadFilenameError(clientRequest.FileName)
		case shared.FILE_DOES_NOT_EXIST:
			err = FileDoesNotExistError(clientRequest.FileName)
		case shared.FILE_EXISTS:
			err = FileExistsError(clientRequest.FileName)
		case shared.MAX_LEN_REACHED:
			err = FileMaxLenReachedError(clientRequest.FileName)
		}
	}
	return
}
