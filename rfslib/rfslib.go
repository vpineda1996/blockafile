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

// Contains recordNum that does not exist
type RecordDoesNotExistError uint16

func (e RecordDoesNotExistError) Error() string {
	return fmt.Sprintf("RFS: Record with recordNum [%d] does not exist", e)
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
	return *rfsInstance, nil
}

// Concrete implementation of RFS interface
var rfsInstance *RFSInstance = nil

type RFSInstance struct {
	// TODO: Fields
	tcpConn *net.TCPConn
}

func (rfs RFSInstance) CreateFile(fname string) (err error) {
	// TODO: Implement CreateFile
	err = nil
	return
}

func (rfs RFSInstance) ListFiles() (fnames []string, err error) {
	// Encode the client request
	clientRequest := shared.RFSClientRequest{RequestType: shared.LIST_FILES}
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err = enc.Encode(clientRequest)
	if err != nil {
		lg.Println(err)
		return
	}

	// Send to miner
	lg.Println("Sending list files request to miner")
	_, err = rfs.tcpConn.Write(buf.Bytes())
	if err != nil {
		// TODO. Should be returning DisconnectedError here?
		lg.Println(err)
		return
	}

	// Make a buffer to hold incoming response
	responseBuf := make([]byte, 1024)

	// Read the incoming connection into the buffer
	readLen, err := rfs.tcpConn.Read(responseBuf)
	if err != nil {
		lg.Println(err)
		return
	}

	// Decode the miner response
	minerResponse := shared.RFSMinerResponse{}
	var reader = bytes.NewReader(responseBuf[:readLen])
	dec := gob.NewDecoder(reader)
	err = dec.Decode(&minerResponse)
	if err != nil {
		lg.Println(err)
		return
	}

	lg.Printf("miner responded to list files request: %v\n", minerResponse)
	return minerResponse.FileNames, minerResponse.Err
}

func (rfs RFSInstance) TotalRecs(fname string) (numRecs uint16, err error) {
	// TODO: Implement TotalRecs
	numRecs = 0
	err = nil
	return
}

func (rfs RFSInstance) ReadRec(fname string, recordNum uint16, record *Record) (err error) {
	// TODO: Implement ReadRec
	err = nil
	return
}

func (rfs RFSInstance) AppendRec(fname string, record *Record) (recordNum uint16, err error) {
	// TODO: Implement AppendRec
	recordNum = 0
	err = nil
	return
}
