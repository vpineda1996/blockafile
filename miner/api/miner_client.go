package api

import (
	"../../crypto"
	"net/rpc"
)

type MinerClient struct {
	client *rpc.Client
}

func (MinerClient) GetNode(id string) *crypto.Block {
	panic("")
}

func (MinerClient) GetRoots() []*crypto.Block {
	panic("")
}

func (MinerClient) GetOtherHosts() []string {
	panic("")
}

func (MinerClient) BroadcastNode() []string {
	panic("")
}

func NewMinerCliet(addr string) (MinerClient, error) {
	c, err := rpc.Dial("tcp", addr)
	if err != nil {
		return MinerClient{}, err
	}
	return MinerClient{
		client: c,
	}, nil
}