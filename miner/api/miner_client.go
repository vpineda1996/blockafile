package api

import (
	"../../crypto"
	"errors"
	"fmt"
	"github.com/DistributedClocks/GoVector/govec"
	"github.com/DistributedClocks/GoVector/govec/vrpc"
	"log"
	"net"
	"net/rpc"
	"os"
	"time"
)

type MinerClient struct {
	client *rpc.Client
	logger *govec.GoLog
	lAddr  string
}

var lg = log.New(os.Stdout, "api: ", log.Lshortfile)

func (m MinerClient) GetBlock(id string) (*crypto.Block, bool, error) {
	args := GetNodeArgs{
		Id: id,
		Host: m.lAddr,
	}
	ans := new(GetNodeRes)

	c := make(chan error, 1)
	go func() { c <- m.client.Call("MinerServer.GetBlock", args, &ans) }()

	select {
	case err := <-c:
		// use err and result
		if err != nil {
			lg.Printf("getNode error: %v", err)
			return nil, false, err
		}
	case <-time.After(time.Duration(time.Second * 5)):
		// call timed out
		lg.Println("getNode timeout")
		return nil, false, errors.New("timeout error: getNode")
	}

	return &ans.Block, ans.Found, nil
}

func (m MinerClient) GetRoots() ([]*crypto.Block, error) {
	args := EmptyArgs{
		Host: m.lAddr,
	}
	ans := make([]*crypto.Block, 0, 1)

	c := make(chan error, 1)
	go func() { c <- m.client.Call("MinerServer.GetRoots", args, &ans) }()

	select {
	case err := <-c:
		// use err and result
		if err != nil {
			lg.Println("GetRoots error" + fmt.Sprint(err))
			return nil, err
		}
	case <-time.After(time.Duration(time.Second * 5)):
		// call timed out
		lg.Println("GetRoots timeout")
		return nil, errors.New("GetRoots error: getNode")
	}

	return ans, nil
}

func (MinerClient) GetOtherHosts() []string {
	panic("")
}

func (m MinerClient) SendBlock(block *crypto.Block) {
	args := ReceiveNodeArgs{
		Block: *block,
		Host: m.lAddr,
	}
	ans := new(bool)

	c := make(chan *rpc.Call, 1)
	m.client.Go("MinerServer.ReceiveNode", args, &ans, c)
	// dicard c as we are just flooding
}

func (m MinerClient) SendJob(block *crypto.BlockOp) {
	args := ReceiveJobArgs{
		BlockOp: *block,
		Host: m.lAddr,
	}
	ans := new(bool)

	c := make(chan *rpc.Call, 1)
	m.client.Go("MinerServer.ReceiveJob", args, &ans, c)
}

func NewMinerClient(clientAddr string, incomingAddr string, outgoingIp string, logger *govec.GoLog) (MinerClient, error) {
	inAddrA, err := net.ResolveTCPAddr("tcp", incomingAddr)
	if err != nil {
		return MinerClient{}, err
	}

	inAddrB, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%v", inAddrA.IP.String(), 0))
	if err != nil {
		return MinerClient{}, err
	}

	raddr, err := net.ResolveTCPAddr("tcp", clientAddr)
	if err != nil {
		return MinerClient{}, err
	}

	conn, err := net.DialTCP("tcp", inAddrB, raddr)
	if err != nil {
		return MinerClient{}, err
	}
	c := vrpc.NewClient(conn, logger, govec.GetDefaultLogOptions())
	outgoingAddr := fmt.Sprintf("%s:%v", outgoingIp, inAddrA.Port)
	return MinerClient{
		client: c,
		logger: logger,
		lAddr:  outgoingAddr,
	}, nil
}
