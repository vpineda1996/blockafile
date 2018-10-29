package fdlib

/*

This package specifies the API to the failure detector library to be
used in assignment 1 of UBC CS 416 2018W1.

You are *not* allowed to change the API below. For example, you can
modify this file by adding an implementation to Initialize, but you
cannot change its API.

*/

import (
	"bytes"
	"encoding/gob"
	"errors"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

//////////////////////////////////////////////////////
// Define the message types fdlib has to use to communicate to other
// fdlib instances. We use Go's type declarations for this:
// https://golang.org/ref/spec#Type_declarations

// Heartbeat message.
type HBeatMessage struct {
	EpochNonce uint64 // Identifies this fdlib instance/epoch.
	SeqNum     uint64 // Unique for each heartbeat in an epoch.
}

// An ack message; response to a heartbeat.
type AckMessage struct {
	HBEatEpochNonce uint64 // Copy of what was received in the heartbeat.
	HBEatSeqNum     uint64 // Copy of what was received in the heartbeat.
}

// Notification of a failure, signal back to the client using this
// library.
type FailureDetected struct {
	UDPIpPort string    // The RemoteIP:RemotePort of the failed node.
	Timestamp time.Time // The time when the failure was detected.
}

//////////////////////////////////////////////////////

// An FD interface represents an instance of the fd
// library. Interfaces are everywhere in Go:
// https://gobyexample.com/interfaces
type FD interface {
	// Tells the library to start responding to heartbeat messages on
	// a local UDP IP:port. Can return an error that is related to the
	// underlying UDP connection.
	StartResponding(LocalIpPort string) (err error)

	// Tells the library to stop responding to heartbeat
	// messages. Always succeeds.
	StopResponding()

	// Tells the library to start monitoring a particular UDP IP:port
	// with a specific lost messages threshold. Can return an error
	// that is related to the underlying UDP connection.
	AddMonitor(LocalIpPort string, RemoteIpPort string, LostMsgThresh uint8) (err error)

	// Tells the library to stop monitoring a particular remote UDP
	// IP:port. Always succeeds.
	RemoveMonitor(RemoteIpPort string)

	// Tells the library to stop monitoring all nodes.
	StopMonitoring()
}

// Private instance of implemented FD interface.
var fdlibInstance *fdlib

// Logger
var lg = log.New(os.Stdout, "fdlib: ", log.Ltime)

// The constructor for a new FD object instance. Note that notifyCh
// can only be received on by the client that receives it from
// initialize:
// https://www.golang-book.com/books/intro/10
func InitializeFDLib(EpochNonce uint64, ChCapacity uint8) (fd FD, notifyCh <-chan FailureDetected, err error) {
	// Initializes the library with an epoch nonce with value epoch-nonce. Initialize can only be called once.
	// Multiple invocations should set fd and notify-channel to nil, and return an appropriate err.
	// The fd is an FD interface instance that implements the rest of the API below.
	// The returned notify-channel channel must have capacity ChCapacity and must be used by
	// fdlib to deliver all failure notifications for nodes being monitored.
	if fdlibInstance != nil {
		return nil, nil, errors.New("multiple invocations of Initialize() are not permitted")
	}

	notifyChannel := make(chan FailureDetected, ChCapacity)
	fdlibInstance =
		&fdlib{
			epochNonce:      EpochNonce,
			notifyCh:        notifyChannel,
			monitoringNodes: SafeMonitoringMap{monitoringNodes: make(map[string]monitoringInfo)},
			respondingOn:    respondingInfo{localIpPort: ""},
			currSeqNum:      SafeCounter{counterValue: 0},
			roundTripTimes:  SafeTimesMap{roundTripTimes: make(map[string]time.Duration)}}

	return fdlibInstance, notifyChannel, nil
}

// For testing purposes.
func TearDownFDLib() {
	close(fdlibInstance.notifyCh)
	fdlibInstance = nil
}

//////////////////////////////////////////////////////
// Functionality private to the module.

// Implementation of the FD interface.
type fdlib struct {
	epochNonce      uint64
	notifyCh        chan FailureDetected
	monitoringNodes SafeMonitoringMap
	// monitoringNodes map[string]monitoringInfo
	respondingOn   respondingInfo
	currSeqNum     SafeCounter
	roundTripTimes SafeTimesMap
}

func (fd *fdlib) StartResponding(LocalIpPort string) (err error) {
	// Tells the library to start responding to heartbeat messages on a local UDP LocalIP:LocalPort.
	// The responses (ack messages) should be always directed to the source LocalIP:LocalPort of the
	// corresponding heartbeat message. Note that fdlib can only be responding on a single LocalIP:LocalPort.
	// Multiple invocations of StartResponding without intermediate calls to StopResponding should result in an error.
	// The err should also be appropriately set if LocalIP:LocalPort combination results in a socket-level error.
	if fd.respondingOn.localIpPort != "" {
		return errors.New("multiple invocations of StartResponding() " +
			"without intermediate calls to StopResponding are not permitted")
	}

	laddr, err := net.ResolveUDPAddr("udp", LocalIpPort)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return err
	}

	stopChannel := make(chan int)
	rInfo := respondingInfo{udpConn: conn, localIpPort: LocalIpPort, stopChannel: stopChannel}
	fd.respondingOn = rInfo
	go respondToNodes(fd, rInfo)

	return nil
}

func (fd *fdlib) StopResponding() {
	// Stops the library from responding to heartbeat messages. This call always succeeds.
	if fd.respondingOn.localIpPort == "" {
		return
	}

	fd.respondingOn.stopChannel <- 0
	close(fd.respondingOn.stopChannel)
	fd.respondingOn.udpConn.Close()
	fd.respondingOn = respondingInfo{localIpPort: ""}
}

func (fd *fdlib) AddMonitor(LocalIpPort string, RemoteIpPort string, LostMsgThresh uint8) (err error) {
	// Tells the library to start monitoring (sending heartbeats to) a node with remote UDP RemoteIP:RemotePort
	// using UDP LocalIP:LocalPort. The lost-msgs-thresh specifies the number of consecutive and un-acked
	// heartbeats messages that the library should send before triggering a failure notification.
	// Multiple invocations of AddMonitor with the same RemoteIP:RemotePort or the same LocalIP:LocalPort
	// should update the lost-msgs-thresh, if it is different; otherwise this scenario is a no-op.
	if oldMInfo, exists := fd.monitoringNodes.Get(RemoteIpPort); exists {
		if oldMInfo.lostMsgThresh == LostMsgThresh {
			// Nothing has changed about the configuration. This scenario is a no-op.
			return nil
		} else {
			updateThreshChannel := oldMInfo.updateThreshChannel
			updateThreshChannel <- LostMsgThresh
			return nil
		}
	}
	laddr, err := net.ResolveUDPAddr("udp", LocalIpPort)
	if err != nil {
		return err
	}
	raddr, err := net.ResolveUDPAddr("udp", RemoteIpPort)
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", laddr, raddr)
	if err != nil {
		return err
	}

	stopChannel := make(chan int)
	updateThreshChannel := make(chan uint8)
	mInfo :=
		monitoringInfo{
			udpConn:             conn,
			lostMsgThresh:       LostMsgThresh,
			stopChannel:         stopChannel,
			updateThreshChannel: updateThreshChannel}
	fd.monitoringNodes.Set(RemoteIpPort, mInfo)
	fd.roundTripTimes.Set(RemoteIpPort, 3*time.Second)
	go monitorNode(fd, RemoteIpPort, mInfo)
	return nil
}

func (fd *fdlib) RemoveMonitor(RemoteIpPort string) {
	// Tells the library to stop monitoring (sending heartbeats to) a node with UDP RemoteIP:RemotePort.
	// This call always succeeds (e.g., it should succeed even if RemoteIP:RemotePort was never
	// passed to AddMonitor).
	if mInfo, exists := fd.monitoringNodes.Get(RemoteIpPort); exists {
		stopChannel := mInfo.stopChannel
		stopChannel <- 0
		mInfo.udpConn.Close()
		close(stopChannel)
		fd.monitoringNodes.Delete(RemoteIpPort)
	}
}

func (fd *fdlib) StopMonitoring() {
	// Tells the library to stop monitoring all nodes (if any). This call always succeeds
	// (e.g., it should succeed even if AddMonitor was never invoked).
	// TODO: Maaaybe doing a range without locking is sketchy. But I don't want to lock up while doing this.
	for remoteIpPort, mInfo := range fd.monitoringNodes.monitoringNodes {
		stopChannel := mInfo.stopChannel
		stopChannel <- 0
		mInfo.udpConn.Close()
		close(stopChannel)
		fd.monitoringNodes.Delete(remoteIpPort)
	}
}

// Helper types
type monitoringInfo struct {
	udpConn             *net.UDPConn
	lostMsgThresh       uint8
	stopChannel         chan int
	updateThreshChannel chan uint8
}

type respondingInfo struct {
	udpConn     *net.UDPConn
	localIpPort string
	stopChannel chan int
}

type SafeCounter struct {
	counterValue uint64
	mux          sync.Mutex
}

func (sc *SafeCounter) GetAndInc() uint64 {
	sc.mux.Lock()
	returnValue := sc.counterValue
	sc.counterValue++
	sc.mux.Unlock()
	return returnValue
}

type SafeTimesMap struct {
	roundTripTimes map[string]time.Duration
	mux            sync.Mutex
}

func (stm *SafeTimesMap) Get(key string) time.Duration {
	stm.mux.Lock()
	defer stm.mux.Unlock()
	return stm.roundTripTimes[key]
}

func (stm *SafeTimesMap) Set(key string, value time.Duration) {
	stm.mux.Lock()
	stm.roundTripTimes[key] = value
	stm.mux.Unlock()
}

type SafeMonitoringMap struct {
	monitoringNodes map[string]monitoringInfo
	mux             sync.Mutex
}

func (smm *SafeMonitoringMap) Set(key string, value monitoringInfo) {
	smm.mux.Lock()
	smm.monitoringNodes[key] = value
	smm.mux.Unlock()
}

func (smm *SafeMonitoringMap) Get(key string) (monitoringInfo, bool) {
	smm.mux.Lock()
	defer smm.mux.Unlock()
	mInfo, ok := smm.monitoringNodes[key]
	return mInfo, ok
}

func (smm *SafeMonitoringMap) Delete(key string) {
	smm.mux.Lock()
	delete(smm.monitoringNodes, key)
	smm.mux.Unlock()
}

// Helper functions
func monitorNode(
	fdlib *fdlib,
	remoteIpPort string,
	mInfo monitoringInfo) {
	recvBuf := make([]byte, 1024)
	udpConnection := mInfo.udpConn
	lostMsgThresh := mInfo.lostMsgThresh
	var countLostMsgs uint8 = 0
	var oneHeartbeatSent = false
	var lastHeartbeatSentAt time.Time
	for {
		select {
		case <-mInfo.stopChannel:
			return
		case newThresh := <-mInfo.updateThreshChannel:
			lostMsgThresh = newThresh
		default:
			// Determine if it has been more than RTT since the last heartbeat was sent.
			timeout := false
			if oneHeartbeatSent && time.Since(lastHeartbeatSentAt) > fdlib.roundTripTimes.Get(remoteIpPort) {
				timeout = true
				countLostMsgs++
			}

			// Check if there's an ACK to receive
			err := udpConnection.SetReadDeadline(time.Now().Add(time.Millisecond * 100))
			if err != nil {
				// Try again. Reset the countLostMsgs state if it changed.
				lg.Println("an error occurred in fdlib (SetReadDeadline)")
				if timeout {
					countLostMsgs--
				}
				continue
			}

			numBytesRead, err := udpConnection.Read(recvBuf)
			readWasSuccessful := false
			if err == nil {
				var ackMsg AckMessage
				err := decodeAckData(recvBuf, numBytesRead, &ackMsg)
				if err != nil {
					// Try again. Reset the countLostMsgs state if it changed.
					lg.Println("an error occurred in fdlib (decoding)")
					if timeout {
						countLostMsgs--
					}
					continue
				}
				if ackMsg.HBEatEpochNonce == fdlib.epochNonce {
					countLostMsgs = 0
					readWasSuccessful = true
					// Calculate the new RTT.
					timeSinceLastHeartbeat := time.Since(lastHeartbeatSentAt)
					fdlib.roundTripTimes.Set(
						remoteIpPort,
						(fdlib.roundTripTimes.Get(remoteIpPort)+timeSinceLastHeartbeat)/2)
				}
			}

			// Check if we are past the threshold for lost messages and stop monitoring if we are
			if countLostMsgs >= lostMsgThresh {
				fdlib.notifyCh <- FailureDetected{UDPIpPort: remoteIpPort, Timestamp: time.Now()}
				mInfo.udpConn.Close()
				close(mInfo.stopChannel)
				fdlib.monitoringNodes.Delete(remoteIpPort)
				return
			}

			// If read was successful or a timeout occurred (or we've never even sent out one heartbeat!),
			// we want to send out a heartbeat
			if readWasSuccessful || timeout || !oneHeartbeatSent {
				encodedData, err :=
					encodeData(HBeatMessage{EpochNonce: fdlib.epochNonce, SeqNum: fdlib.currSeqNum.GetAndInc()})
				if err != nil {
					// Try again. Reset the countLostMsgs state if it changed.
					lg.Println("an error occurred in fdlib (encoding)")
					if timeout {
						countLostMsgs--
					}
					continue
				}
				_, err = udpConnection.Write(encodedData)
				if err != nil {
					// Try again. Reset the countLostMsgs state if it changed.
					lg.Println("an error occurred in fdlib (writing)")
					if timeout {
						countLostMsgs--
					}
					continue
				}
				oneHeartbeatSent = true
				lastHeartbeatSentAt = time.Now()
			}
		}
	}
}

func respondToNodes(fd *fdlib, rInfo respondingInfo) {
	recvBuf := make([]byte, 1024)
	udpConnection := rInfo.udpConn

	for {
		select {
		case <-rInfo.stopChannel:
			return
		default:
			numBytesRead, udpAddr, err := udpConnection.ReadFromUDP(recvBuf)
			if err != nil {
				lg.Println("an error occurred in fdlib (reading)")
				continue
			}

			var hbeatMsg HBeatMessage
			err = decodeHeartbeatData(recvBuf, numBytesRead, &hbeatMsg)
			if err != nil {
				lg.Println("an error occurred in fdlib (decoding)")
				continue
			}

			ackMessage := AckMessage{HBEatEpochNonce: hbeatMsg.EpochNonce, HBEatSeqNum: hbeatMsg.SeqNum}
			encodedAck, err := encodeData(ackMessage)
			if err != nil {
				lg.Println("an error occurred in fdlib (encoding)")
				continue
			}

			_, err = udpConnection.WriteTo(encodedAck, udpAddr)
			if err != nil {
				lg.Println("an error occurred in fdlib (writing)")
				continue
			}
		}
	}
}

func encodeData(data interface{}) ([]byte, error) {
	var buf bytes.Buffer

	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	return buf.Bytes(), err
}

func decodeAckData(data []byte, bytesRead int, ackMsg *AckMessage) error {
	bufDecoder := bytes.NewBuffer(data[:bytesRead])
	decoder := gob.NewDecoder(bufDecoder)
	err := decoder.Decode(&ackMsg)
	return err
}

func decodeHeartbeatData(data []byte, bytesRead int, hbeatMsg *HBeatMessage) error {
	bufDecoder := bytes.NewBuffer(data[:bytesRead])
	decoder := gob.NewDecoder(bufDecoder)
	err := decoder.Decode(&hbeatMsg)
	return err
}

