package instance

import (
	. "../../shared"
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"math/rand"
	"net"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"
)

var initialPort int
var portMutex *sync.Mutex

func init() {
	seed := rand.NewSource(time.Now().UnixNano())
	randomNumGenerator := rand.New(seed)
	initialPort = 8080 + randomNumGenerator.Intn(1000)
	portMutex = new(sync.Mutex)
}

type MockMiner struct {
}

func (m MockMiner) DeleteRecHandler(fname string) (errorType FailureType) {
	return NO_ERROR
}

func (m MockMiner) CreateFileHandler(fname string) (errorType FailureType) {
	return NO_ERROR
}

func (m MockMiner) ListFilesHandler() (fnames []string) {
	return []string{"File1", "File2", "File3"}
}

func (m MockMiner) TotalRecsHandler(fname string) (numRecs uint16, errorType FailureType) {
	return 3, NO_ERROR
}

func (m MockMiner) ReadRecHandler(fname string, recordNum uint16) (record [512]byte, errorType FailureType) {
	return [512]byte{}, NO_ERROR
}

func (m MockMiner) AppendRecHandler(fname string, record [512]byte) (recordNum uint16, errorType FailureType) {
	return 0, NO_ERROR
}

func TestListenForClients(t *testing.T) {

	t.Run("should return error if given address is invalid", func(t *testing.T) {
		wg := &sync.WaitGroup{}
		testInstance := ClientHandler{waitGroup: wg, ListenHost: "0"}
		wg.Add(1)
		err := testInstance.ListenForClients()
		assert(t, err != nil, "should return error")
	})

	t.Run("should loop infinitely if no errors occurring", func(t *testing.T) {
		wg := &sync.WaitGroup{}
		testInstance := ClientHandler{waitGroup: wg, ListenHost: fmt.Sprintf("127.0.0.1:%v", generateNextPort())}
		wg.Add(1)

		var err error = nil
		listenForClientsWrapper := func() {
			err = testInstance.ListenForClients()
		}
		go listenForClientsWrapper()
		time.Sleep(time.Second * 5)
		assert(t, err == nil, "should not return error, looping infinitely")
	})
}

func TestServiceClientRequest(t *testing.T) {
	minerAddr := fmt.Sprintf("127.0.0.1:%v", generateNextPort())
	maddr, _ := net.ResolveTCPAddr("tcp", minerAddr)


	var minerInstance Miner = MockMiner{}
	wg := &sync.WaitGroup{}
	testInstance := ClientHandler{waitGroup: wg, miner: &minerInstance}

	var serviceError error = nil
	serviceClientRequestWrapper := func(testInstance ClientHandler, conn net.Conn) {
		serviceError = testInstance.ServiceClientRequest(conn)
	}
	go func() {
		testInstance.ListenHost = minerAddr
		addr, err := net.ResolveTCPAddr("tcp", minerAddr)
		ok(t, err)
		listener, err := net.ListenTCP("tcp", addr)
		ok(t, err)
		for {
			conn, err := listener.Accept()
			ok(t, err)
			go serviceClientRequestWrapper(testInstance, conn)
		}
	}()

	t.Run("should close connection if client leaves", func(t *testing.T) {
		clientAddr := fmt.Sprintf("127.0.0.1:%v", generateNextPort())
		caddr, _ := net.ResolveTCPAddr("tcp", clientAddr)
		assert(t, serviceError == nil, "error should be nil before execution of function")
		connClient, err := net.DialTCP("tcp", caddr, maddr)
		ok(t, err)
		connClient.Close()
		time.Sleep(time.Second / 2)
		assert(t, serviceError == io.EOF, "error should be EOF when client connection closes")
	})

	t.Run("should fail to parse the current request if decoding error occurs", func(t *testing.T) {
		clientAddr := fmt.Sprintf("127.0.0.1:%v", generateNextPort())
		caddr, _ := net.ResolveTCPAddr("tcp", clientAddr)
		serviceError = nil
		connClient, err := net.DialTCP("tcp", caddr, maddr)
		ok(t, err)
		invalidRequest := 0
		validRequest := RFSClientRequest{RequestType: CREATE_FILE, FileName: "FileName"}
		sendRequest(validRequest, connClient, t)
		sendRequest(invalidRequest, connClient, t)
		assert(t, serviceError == nil, "error should still be nil since no failure occurred")
		_, timeout := getResponseOrTimeout(connClient, t)
		assert(t, !timeout, "should get response for valid request")
		_, timeout = getResponseOrTimeout(connClient, t)
		assert(t, timeout, "should timeout the second time since this request was never parsed")
	})

	t.Run("should respond to create file request", func(t *testing.T) {
		clientAddr := fmt.Sprintf("127.0.0.1:%v", generateNextPort())
		caddr, _ := net.ResolveTCPAddr("tcp", clientAddr)
		serviceError = nil
		connClient, err := net.DialTCP("tcp", caddr, maddr)
		ok(t, err)
		validRequest := RFSClientRequest{RequestType: CREATE_FILE, FileName: "FileName"}
		sendRequest(validRequest, connClient, t)
		_, timeout := getResponseOrTimeout(connClient, t)
		assert(t, !timeout, "should get response for create file request")
	})

	t.Run("should respond to delete file request", func(t *testing.T) {
		clientAddr := fmt.Sprintf("127.0.0.1:%v", generateNextPort())
		caddr, _ := net.ResolveTCPAddr("tcp", clientAddr)
		serviceError = nil
		connClient, err := net.DialTCP("tcp", caddr, maddr)
		ok(t, err)
		validRequest := RFSClientRequest{RequestType: DELETE_FILE, FileName: "FileName"}
		sendRequest(validRequest, connClient, t)
		_, timeout := getResponseOrTimeout(connClient, t)
		assert(t, !timeout, "should get response for create file request")
	})

	t.Run("should respond to list files request", func(t *testing.T) {
		clientAddr := fmt.Sprintf("127.0.0.1:%v", generateNextPort())
		caddr, _ := net.ResolveTCPAddr("tcp", clientAddr)
		serviceError = nil
		connClient, err := net.DialTCP("tcp", caddr, maddr)
		ok(t, err)
		validRequest := RFSClientRequest{RequestType: LIST_FILES}
		sendRequest(validRequest, connClient, t)
		_, timeout := getResponseOrTimeout(connClient, t)
		assert(t, !timeout, "should get response for list files request")
	})

	t.Run("should respond to total records request", func(t *testing.T) {
		clientAddr := fmt.Sprintf("127.0.0.1:%v", generateNextPort())
		caddr, _ := net.ResolveTCPAddr("tcp", clientAddr)
		serviceError = nil
		connClient, err := net.DialTCP("tcp", caddr, maddr)
		ok(t, err)
		validRequest := RFSClientRequest{RequestType: TOTAL_RECS, FileName: "FileName"}
		sendRequest(validRequest, connClient, t)
		_, timeout := getResponseOrTimeout(connClient, t)
		assert(t, !timeout, "should get response for total records request")
	})

	t.Run("should respond to read record request", func(t *testing.T) {
		clientAddr := fmt.Sprintf("127.0.0.1:%v", generateNextPort())
		caddr, _ := net.ResolveTCPAddr("tcp", clientAddr)
		serviceError = nil
		connClient, err := net.DialTCP("tcp", caddr, maddr)
		ok(t, err)
		validRequest := RFSClientRequest{RequestType: READ_REC, FileName: "FileName", RecordNum: 0}
		sendRequest(validRequest, connClient, t)
		_, timeout := getResponseOrTimeout(connClient, t)
		assert(t, !timeout, "should get response for read record request")
	})

	t.Run("should respond to append record request", func(t *testing.T) {
		clientAddr := fmt.Sprintf("127.0.0.1:%v", generateNextPort())
		caddr, _ := net.ResolveTCPAddr("tcp", clientAddr)
		serviceError = nil
		connClient, err := net.DialTCP("tcp", caddr, maddr)
		ok(t, err)
		var record [512]byte
		validRequest := RFSClientRequest{RequestType: APPEND_REC, FileName: "FileName", AppendRecord: record}
		sendRequest(validRequest, connClient, t)
		_, timeout := getResponseOrTimeout(connClient, t)
		assert(t, !timeout, "should get response for append record request")
	})

	t.Run("should fail to parse the current request if invalid request type", func(t *testing.T) {
		clientAddr := fmt.Sprintf("127.0.0.1:%v", generateNextPort())
		caddr, _ := net.ResolveTCPAddr("tcp", clientAddr)
		serviceError = nil
		connClient, err := net.DialTCP("tcp", caddr, maddr)
		ok(t, err)
		validRequest := RFSClientRequest{RequestType: -99}
		sendRequest(validRequest, connClient, t)
		_, timeout := getResponseOrTimeout(connClient, t)
		assert(t, timeout, "should timeout for invalid request type")
	})
}

func generateNextPort() int {
	portMutex.Lock()
	defer portMutex.Unlock()
	initialPort++
	return initialPort
}

func sendRequest(request interface{}, tcpConn *net.TCPConn, t *testing.T) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(request)
	ok(t, err)
	_, err = tcpConn.Write(buf.Bytes())
	ok(t, err)
}

// Returns the response and/or true if the read timed out
func getResponseOrTimeout(tcpConn *net.TCPConn, t *testing.T) (RFSMinerResponse, bool) {
	minerResponse := RFSMinerResponse{}
	responseBuf := make([]byte, 1024)
	tcpConn.SetReadDeadline(time.Now().Add(time.Second * 3))
	readLen, err := tcpConn.Read(responseBuf)
	if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
		return minerResponse, true
	}
	ok(t, err)

	var reader = bytes.NewReader(responseBuf[:readLen])
	dec := gob.NewDecoder(reader)
	err = dec.Decode(&minerResponse)
	ok(t, err)
	return minerResponse, false
}

// Taken from https://github.com/benbjohnson/testing
// assert fails the test if the condition is false.
func assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%str:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%str:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}

// ok fails the test if an err is not nil.
func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%str:%d: unexpected error: %str\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}
