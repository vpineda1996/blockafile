package api

import (
	"../../crypto"
	"errors"
	"fmt"
	"log"
	"net/rpc"
	"os"
	"time"
)

type MinerClient struct {
	client *rpc.Client
}

var lg = log.New(os.Stdout, "minerC: ", log.Lshortfile)

func (m MinerClient) GetBlock(id string) (*crypto.Block, bool, error) {
	args := GetNodeArgs{Id: id}
	ans := new(GetNodeRes)

	c := make(chan error, 1)
	go func() { c <- m.client.Call("MinerServer.GetBlock", args, &ans) } ()

	// todo vpineda tcp should detect a failure on the connection or just wait for 5 seconds
	select {
	case err := <-c:
		// use err and result
		if err != nil {
			lg.Fatalf("getNode error: %v", err)
			return nil, false, err
		}
	case <-time.After(time.Duration(time.Second * 5)):
		// call timed out
		lg.Fatal("getNode timeout")
		return nil, false, errors.New("timeout error: getNode")
	}

	return &ans.Block, ans.Found, nil
}

func (m MinerClient) GetRoots() ([]*crypto.Block, error) {
	args := EmptyArgs{}
	ans := make([]*crypto.Block, 0, 1)

	c := make(chan error, 1)
	go func() { c <- m.client.Call("MinerServer.GetRoots", args, &ans) } ()

	select {
	case err := <-c:
		// use err and result
		if err != nil {
			lg.Fatal("GetRoots error" + fmt.Sprint(err))
			return nil, err
		}
	case <-time.After(time.Duration(time.Second * 5)):
		// call timed out
		lg.Fatal("GetRoots timeout")
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
	}
	ans := new(bool)

	c := make(chan *rpc.Call, 1)
	m.client.Go("MinerServer.ReceiveNode", args, &ans, c)
	// dicard c as we are just flooding
}

func (m MinerClient) SendJob(block *crypto.BlockOp) {
	args := ReceiveJobArgs{
		BlockOp: *block,
	}
	ans := new(bool)

	c := make(chan *rpc.Call, 1)
	m.client.Go("MinerServer.ReceiveJob", args, &ans, c)
}

func NewMinerCliet(addr string) (MinerClient, error) {
	c, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		return MinerClient{}, err
	}
	return MinerClient{
		client: c,
	}, nil
}